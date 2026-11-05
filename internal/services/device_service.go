package services

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owgw"
	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owprov"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

// DeviceService defines business operations for managing fleet devices.
type DeviceService interface {
	GetDevices(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error)
	GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	GetDeviceCommands(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	DeleteDeviceCommand(ctx context.Context, serial, commandUUID, token, reqID, corrID string) error
	GetDeviceHealthChecks(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	GetDeviceLogs(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	GetDeviceStatistics(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	GetDeviceCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error
	GetBlacklist(ctx context.Context, token, reqID, corrID string) ([]models.BlacklistDevice, error)
	AddBlacklistDevice(ctx context.Context, serial, reason string, token, reqID, corrID string) (json.RawMessage, error)
	DeleteBlacklistDevice(ctx context.Context, serial, token, reqID, corrID string) error
}

type deviceService struct {
	provClient owprov.Client
	gwClient   owgw.Client
	logger     *slog.Logger
}

// NewDeviceService constructs a new DeviceService instance.
func NewDeviceService(provClient owprov.Client, gwClient owgw.Client, logger ...*slog.Logger) DeviceService {
	var l *slog.Logger
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	} else {
		l = slog.Default()
	}
	return &deviceService{
		provClient: provClient,
		gwClient:   gwClient,
		logger:     l,
	}
}

func normalizeSerial(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ":", "")
	s = strings.ReplaceAll(s, "-", "")
	return strings.TrimSpace(s)
}

func parseLastContact(v any) *int64 {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case float64:
		n := int64(val)
		return &n
	case int64:
		return &val
	case int:
		n := int64(val)
		return &n
	case string:
		if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
			return &parsed
		}
	}
	return nil
}

// GetDevices orchestrates fetching user-scoped inventory from OWPROV and live telemetry from OWGW.
func (s *deviceService) GetDevices(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error) {
	if pageIndex < 0 {
		pageIndex = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := pageIndex * pageSize

	var (
		tags       []models.InventoryTag
		totalCount int
		tagsErr    error
		countErr   error
		wg         sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		tags, tagsErr = s.provClient.GetInventory(ctx, token, reqID, corrID, pageSize, offset, unassignedOnly)
	}()

	go func() {
		defer wg.Done()
		totalCount, countErr = s.provClient.GetInventoryCount(ctx, token, reqID, corrID, unassignedOnly)
	}()

	wg.Wait()

	if tagsErr != nil {
		return nil, tagsErr
	}

	if countErr != nil {
		if s.logger != nil {
			s.logger.Warn("failed to fetch total inventory count, defaulting to active page count", "error", countErr)
		}
		totalCount = offset + len(tags)
	}

	totalPages := 1
	if pageSize > 0 && totalCount > 0 {
		totalPages = int(math.Ceil(float64(totalCount) / float64(pageSize)))
	}
	if totalPages < 1 {
		totalPages = 1
	}

	if len(tags) == 0 {
		return &models.PaginatedDevicesResponse{
			Devices:    []models.DeviceFleetItem{},
			TotalCount: totalCount,
			PageIndex:  pageIndex,
			PageSize:   pageSize,
			TotalPages: totalPages,
		}, nil
	}

	// Collect unique normalized serial numbers
	serials := make([]string, 0, len(tags))
	for _, tag := range tags {
		if strings.TrimSpace(tag.SerialNumber) != "" {
			serials = append(serials, tag.SerialNumber)
		}
	}

	// Concurrently query OWGW for live status of only these scoped serials
	var gwStatuses []models.GatewayDeviceStatus
	if len(serials) > 0 && s.gwClient != nil {
		var gwErr error
		gwStatuses, gwErr = s.gwClient.GetDevicesStatus(ctx, serials, token, reqID, corrID)
		if gwErr != nil && s.logger != nil {
			s.logger.Warn("failed to fetch live gateway statuses, continuing with inventory data", "error", gwErr)
		}
	}

	gwMap := make(map[string]models.GatewayDeviceStatus, len(gwStatuses))
	for _, st := range gwStatuses {
		gwMap[normalizeSerial(st.SerialNumber)] = st
	}

	now := time.Now().Unix()
	devices := make([]models.DeviceFleetItem, 0, len(tags))

	for _, tag := range tags {
		norm := normalizeSerial(tag.SerialNumber)
		liveGw, hasGw := gwMap[norm]

		// Resolve hardware model
		var model string
		if hasGw && liveGw.Compatible != "" {
			model = liveGw.Compatible
		} else if tag.DeviceType != "" {
			model = tag.DeviceType
		} else {
			model = "Universal AP"
		}

		// Resolve platform group/type (ap, switch, olg) matching OWGW
		platformType := ""
		if hasGw && liveGw.DeviceType != "" {
			platformType = strings.ToLower(strings.TrimSpace(liveGw.DeviceType))
		} else if tag.DevClass != "" && (tag.DevClass == "ap" || tag.DevClass == "switch" || tag.DevClass == "olg") {
			platformType = strings.ToLower(strings.TrimSpace(tag.DevClass))
		} else {
			lowerType := strings.ToLower(tag.DeviceType)
			if strings.Contains(lowerType, "olg") || strings.Contains(lowerType, "gateway") {
				platformType = "olg"
			} else if strings.Contains(lowerType, "switch") || strings.Contains(lowerType, "ecs") {
				platformType = "switch"
			} else {
				platformType = "ap"
			}
		}

		item := models.DeviceFleetItem{
			SerialNumber: tag.SerialNumber,
			Name:         tag.Name,
			Description:  tag.Description,
			Model:        model,
			DeviceGroup:  platformType,
			DeviceType:   platformType,
			IPAddress:    "-",
			Firmware:     "TIP-v4.2",
			Sanity:       0,
		}

		if item.Name == "" {
			if hasGw && liveGw.Compatible != "" {
				item.Name = liveGw.Compatible
			} else {
				item.Name = tag.SerialNumber
			}
		}

		// Extended info mapping
		if tag.ExtendedInfo != nil {
			if tag.ExtendedInfo.Entity != nil {
				item.EntityID = tag.ExtendedInfo.Entity.ID
				item.EntityName = tag.ExtendedInfo.Entity.Name
			}
			if tag.ExtendedInfo.Venue != nil {
				item.VenueID = tag.ExtendedInfo.Venue.ID
				item.VenueName = tag.ExtendedInfo.Venue.Name
			}
			if tag.ExtendedInfo.Subscriber != nil {
				item.SubscriberID = tag.ExtendedInfo.Subscriber.ID
				item.SubscriberEmail = tag.ExtendedInfo.Subscriber.Email
			}
			if tag.ExtendedInfo.DeviceConfiguration != nil {
				item.ConfigID = tag.ExtendedInfo.DeviceConfiguration.ID
				item.ConfigName = tag.ExtendedInfo.DeviceConfiguration.Name
			}
		} else {
			item.EntityID = tag.Entity
			item.VenueID = tag.Venue
			item.SubscriberID = tag.Subscriber
			item.ConfigID = tag.DeviceConfiguration
		}

		if tag.RealMacAddress != "" {
			item.MacAddress = tag.RealMacAddress
		}

		// Gateway live metrics merge
		if hasGw {
			item.Connected = liveGw.Connected
			if liveGw.Sanity != nil {
				item.Sanity = *liveGw.Sanity
			} else if liveGw.Connected {
				item.Sanity = 100
			}

			if liveGw.IPAddress != "" {
				item.IPAddress = liveGw.IPAddress
			}
			if liveGw.MacAddress != "" {
				item.MacAddress = liveGw.MacAddress
			}
			if liveGw.Firmware != "" {
				item.Firmware = liveGw.Firmware
			}

			item.Locale = liveGw.Locale
			item.MemoryUsedPct = liveGw.MemoryUsed
			item.Load = liveGw.Load
			item.Temperature = liveGw.Temperature
			item.Started = liveGw.Started
			if liveGw.Started != nil && *liveGw.Started > 0 && *liveGw.Started <= now {
				uptime := now - *liveGw.Started
				item.UptimeSeconds = &uptime
			}

			item.LastContact = liveGw.LastContact
			item.LastContactTimestamp = parseLastContact(liveGw.LastContact)
			item.LastRecordedContact = liveGw.LastRecordedContact
			item.LastFWUpdate = liveGw.LastFWUpdate
			item.ConnectReason = liveGw.ConnectReason
			item.CertificateExpiryDate = liveGw.CertificateExpiryDate
			item.HasGPS = liveGw.HasGPS
			item.Blacklisted = liveGw.BlackListed
			item.Simulated = liveGw.Simulated
			item.Restricted = liveGw.RestrictedDevice

			if b, ok := liveGw.HasRADIUSSessions.(bool); ok {
				item.HasRADIUSSessions = b
			} else if n, ok := liveGw.HasRADIUSSessions.(float64); ok {
				item.HasRADIUSSessions = n > 0
			}

			item.Associations2G = liveGw.Associations2G
			item.Associations5G = liveGw.Associations5G
			item.Associations6G = liveGw.Associations6G
			item.AssociationsTotal = liveGw.Associations2G + liveGw.Associations5G + liveGw.Associations6G
			item.RxBytes = liveGw.RxBytes
			item.TxBytes = liveGw.TxBytes
		}

		devices = append(devices, item)
	}

	return &models.PaginatedDevicesResponse{
		Devices:    devices,
		TotalCount: totalCount,
		PageIndex:  pageIndex,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// verifyDeviceAccess enforces strict multi-tenant RBAC by verifying via OWPROV that
// the device is assigned to an entity, venue, or subscriber accessible to the authenticated user.
func (s *deviceService) verifyDeviceAccess(ctx context.Context, serial, token, reqID, corrID string) error {
	norm := normalizeSerial(serial)
	if norm == "" {
		return models.NewApiError(http.StatusBadRequest, "Bad Request", "serial number is required")
	}

	if s.provClient == nil {
		return models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owprov client unavailable")
	}

	_, err := s.provClient.GetInventoryItem(ctx, norm, token, reqID, corrID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("RBAC access check rejected: device not found or not assigned to caller's scope in OWPROV",
				"serial", norm,
				"error", err,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return models.NewApiError(http.StatusForbidden, "Forbidden", "Device is not assigned to user or access denied")
	}

	return nil
}

// GetDeviceCommands proxies commands history from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDeviceCommands(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	if queryParams == nil {
		queryParams = url.Values{}
	}
	queryParams.Set("serialNumber", norm)
	return s.gwClient.GetCommands(ctx, queryParams, token, reqID, corrID)
}

// DeleteDeviceCommand deletes a specific command record in OWGW after verifying user authorization in OWPROV
// and verifying that the command record actually belongs to the specified device serial.
func (s *deviceService) DeleteDeviceCommand(ctx context.Context, serial, commandUUID, token, reqID, corrID string) error {
	normSerial := normalizeSerial(serial)
	if normSerial == "" {
		return models.NewApiError(http.StatusBadRequest, "Bad Request", "serial number is required")
	}
	cleanUUID := strings.TrimSpace(commandUUID)
	if cleanUUID == "" {
		return models.NewApiError(http.StatusBadRequest, "Bad Request", "command UUID is required")
	}

	if err := s.verifyDeviceAccess(ctx, normSerial, token, reqID, corrID); err != nil {
		return err
	}
	if s.gwClient == nil {
		return models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}

	// Fetch command details to verify ownership before deletion
	cmdRaw, err := s.gwClient.GetCommand(ctx, cleanUUID, token, reqID, corrID)
	if err != nil {
		return err
	}

	var cmd struct {
		SerialNumber string `json:"serialNumber"`
	}
	if err := json.Unmarshal(cmdRaw, &cmd); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to parse command record from OWGW", "commandUUID", cleanUUID, "error", err)
		}
		return models.NewApiError(http.StatusInternalServerError, "Internal Server Error", "failed to parse command record")
	}

	if normalizeSerial(cmd.SerialNumber) != normSerial {
		if s.logger != nil {
			s.logger.Warn("rejected command deletion: command does not belong to specified device",
				"commandUUID", cleanUUID,
				"requestedSerial", normSerial,
				"commandSerial", normalizeSerial(cmd.SerialNumber),
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return models.NewApiError(http.StatusNotFound, "Not Found", "command not found for this device")
	}

	return s.gwClient.DeleteCommand(ctx, cleanUUID, token, reqID, corrID)
}

// GetDeviceHealthChecks proxies healthcheck history from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDeviceHealthChecks(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	if queryParams == nil {
		queryParams = url.Values{}
	}
	return s.gwClient.GetHealthChecks(ctx, norm, queryParams, token, reqID, corrID)
}

// GetDeviceLogs proxies device logs from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDeviceLogs(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	if queryParams == nil {
		queryParams = url.Values{}
	}
	return s.gwClient.GetLogs(ctx, norm, queryParams, token, reqID, corrID)
}

// GetDeviceStatistics proxies device statistics from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDeviceStatistics(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	if queryParams == nil {
		queryParams = url.Values{}
	}
	return s.gwClient.GetStatistics(ctx, norm, queryParams, token, reqID, corrID)
}

// GetDeviceStatus proxies live device status from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	return s.gwClient.GetDeviceStatus(ctx, norm, token, reqID, corrID)
}

// GetDevice proxies device metadata from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	return s.gwClient.GetDevice(ctx, norm, token, reqID, corrID)
}

// GetDeviceCapabilities proxies device hardware/firmware capabilities from OWGW after verifying user authorization in OWPROV.
func (s *deviceService) GetDeviceCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	return s.gwClient.GetCapabilities(ctx, norm, token, reqID, corrID)
}

// UpdateDevice proxies device updates/notes to OWGW after verifying user authorization in OWPROV.
func (s *deviceService) UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	return s.gwClient.UpdateDevice(ctx, norm, body, token, reqID, corrID)
}

// ExecuteDeviceCommand forwards a device command to OWGW after pre-flight RBAC verification against OWPROV.
func (s *deviceService) ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	return s.gwClient.ExecuteDeviceCommand(ctx, norm, command, body, token, reqID, corrID)
}

// GetDeviceRtty initiates or fetches RTTY connection details from OWGW after verifying authorization.
func (s *deviceService) GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return nil, err
	}
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}
	norm := normalizeSerial(serial)
	return s.gwClient.GetDeviceRtty(ctx, norm, token, reqID, corrID)
}

// DeleteDevice deletes a device from OWGW and OWPROV inventory after verifying authorization.
func (s *deviceService) DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	if err := s.verifyDeviceAccess(ctx, serial, token, reqID, corrID); err != nil {
		return err
	}
	norm := normalizeSerial(serial)

	// 1. Attempt delete in OWGW (best effort cleanup; ignore 404 Not Found since offline/unassigned devices may not be in gateway)
	var gwErr error
	if s.gwClient != nil {
		if err := s.gwClient.DeleteDevice(ctx, norm, token, reqID, corrID); err != nil {
			var apiErr models.ApiError
			if errors.As(err, &apiErr) && apiErr.ErrorCode == http.StatusNotFound {
				if s.logger != nil {
					s.logger.Debug("device not present in OWGW during deletion", "serial", norm)
				}
			} else {
				gwErr = err
				if s.logger != nil {
					s.logger.Warn("failed to delete device from OWGW", "serial", norm, "error", err)
				}
			}
		}
	}

	// 2. Authoritative deletion from OWPROV inventory
	if s.provClient == nil {
		return models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owprov client unavailable")
	}
	if err := s.provClient.DeleteInventoryItem(ctx, norm, token, reqID, corrID); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to delete device from OWPROV inventory", "serial", norm, "error", err)
		}
		return err
	}

	// 3. If OWPROV succeeded but OWGW had a non-404 hard failure, return the gateway error so caller knows cleanup was incomplete
	if gwErr != nil {
		return gwErr
	}

	return nil
}

// GetBlacklist returns blacklisted devices filtered by caller's inventory scope.
func (s *deviceService) GetBlacklist(ctx context.Context, token, reqID, corrID string) ([]models.BlacklistDevice, error) {
	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}

	// 1. Fetch blacklisted devices from OWGW
	gwDevices, err := s.gwClient.GetBlacklist(ctx, 500, 0, token, reqID, corrID)
	if err != nil {
		return nil, err
	}

	if len(gwDevices) == 0 {
		return []models.BlacklistDevice{}, nil
	}

	// 2. Fetch accessible inventory serial numbers for caller to enforce tenant scoping
	if s.provClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owprov client unavailable")
	}

	accessibleSerials, err := s.provClient.GetInventorySerials(ctx, token, reqID, corrID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("failed to fetch user inventory serials for blacklist scoping", "error", err)
		}
		return nil, err
	}

	allowed := make(map[string]bool, len(accessibleSerials))
	for _, sn := range accessibleSerials {
		allowed[normalizeSerial(sn)] = true
	}

	// 3. Filter only devices that belong to the caller's inventory
	scoped := make([]models.BlacklistDevice, 0, len(gwDevices))
	for _, dev := range gwDevices {
		if allowed[normalizeSerial(dev.SerialNumber)] {
			scoped = append(scoped, dev)
		}
	}

	return scoped, nil
}

// AddBlacklistDevice verifies device inventory access before adding it to OWGW blacklist.
func (s *deviceService) AddBlacklistDevice(ctx context.Context, serial, reason string, token, reqID, corrID string) (json.RawMessage, error) {
	norm := normalizeSerial(serial)
	if norm == "" {
		return nil, models.NewApiError(http.StatusBadRequest, "Bad Request", "serial number is required")
	}

	// Authorize device against user's inventory in OWPROV
	if err := s.verifyDeviceAccess(ctx, norm, token, reqID, corrID); err != nil {
		return nil, err
	}

	if s.gwClient == nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}

	payload := map[string]string{
		"serialNumber": norm,
		"reason":       reason,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, models.NewApiError(http.StatusInternalServerError, "Internal Server Error", "failed to serialize blacklist payload: "+err.Error())
	}

	return s.gwClient.AddBlacklistDevice(ctx, norm, body, token, reqID, corrID)
}

// DeleteBlacklistDevice verifies device inventory access before removing it from OWGW blacklist.
func (s *deviceService) DeleteBlacklistDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	norm := normalizeSerial(serial)
	if norm == "" {
		return models.NewApiError(http.StatusBadRequest, "Bad Request", "serial number is required")
	}

	// Authorize device against user's inventory in OWPROV
	if err := s.verifyDeviceAccess(ctx, norm, token, reqID, corrID); err != nil {
		return err
	}

	if s.gwClient == nil {
		return models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw client unavailable")
	}

	return s.gwClient.DeleteBlacklistDevice(ctx, norm, token, reqID, corrID)
}
