package owgw

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/ra-common-mods/apperror"
)

const (
	userAgent        = "mango-mdu-service/1.0"
	maxErrorBytes    = 2048
	maxResponseBytes = 10 << 20 // 10 MB
)

// Client defines the contract for communicating with OWGW.
type Client interface {
	GetDevicesStatus(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error)
	GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	GetCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	GetCommands(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	GetCommand(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error)
	DeleteCommand(ctx context.Context, commandUUID, token, reqID, corrID string) error
	GetHealthChecks(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	GetLogs(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	GetStatistics(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error
	GetBlacklist(ctx context.Context, limit, offset int, token, reqID, corrID string) ([]models.BlacklistDevice, error)
	AddBlacklistDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	DeleteBlacklistDevice(ctx context.Context, serial, token, reqID, corrID string) error
}

type client struct {
	urlResolver func() string
	httpClient  *http.Client
	logger      *slog.Logger
}

// Config holds configuration for creating an OWGW client.
type Config struct {
	URLResolver func() string
	Timeout     time.Duration
	TLSConfig   *tls.Config
	Logger      *slog.Logger
}

// NewClient creates a new OWGW client instance.
func NewClient(cfg Config) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.TLSConfig != nil {
		transport.TLSClientConfig = cfg.TLSConfig
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &client{
		urlResolver: cfg.URLResolver,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		logger: logger,
	}
}

func (c *client) getBaseURL() (string, error) {
	if c.urlResolver != nil {
		if resolved := c.urlResolver(); resolved != "" {
			return strings.TrimRight(resolved, "/"), nil
		}
	}
	return "", models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owgw service endpoint not discovered or available")
}

// GetDevicesStatus retrieves real-time status and telemetry for the specified serial numbers from OWGW.
func (c *client) GetDevicesStatus(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error) {
	if len(serialNumbers) == 0 {
		return []models.GatewayDeviceStatus{}, nil
	}

	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	selectList := strings.Join(serialNumbers, ",")
	endpoint := fmt.Sprintf("%s/api/v1/devices?deviceWithStatus=true&select=%s", baseURL, selectList)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create devices status request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWGW request failed",
				"service", "owgw",
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWGW service unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		if c.logger != nil {
			c.logger.Error("downstream OWGW returned non-200 status",
				"service", "owgw",
				"status", resp.StatusCode,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.ParseResponseError(resp, maxErrorBytes)
	}

	var statusResp models.GatewayDevicesWithStatusResponse
	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(limitedReader).Decode(&statusResp); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse gateway devices status response", err)
	}

	return statusResp.DevicesWithStatus, nil
}

func (c *client) doJSONRequest(ctx context.Context, method, endpoint, token, reqID, corrID string) (json.RawMessage, error) {
	return c.doJSONRequestWithBody(ctx, method, endpoint, nil, token, reqID, corrID)
}

func (c *client) doJSONRequestWithBody(ctx context.Context, method, endpoint string, bodyPayload []byte, token, reqID, corrID string) (json.RawMessage, error) {
	var bodyReader io.Reader
	if len(bodyPayload) > 0 {
		bodyReader = bytes.NewReader(bodyPayload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create gateway request", err)
	}

	c.setHeaders(req, token, reqID, corrID)
	if len(bodyPayload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWGW request failed",
				"service", "owgw",
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWGW service unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode >= 400 {
		if c.logger != nil {
			c.logger.Error("downstream OWGW returned non-200 status",
				"service", "owgw",
				"status", resp.StatusCode,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.ParseResponseError(resp, maxErrorBytes)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to read gateway response", err)
	}

	return json.RawMessage(body), nil
}

// GetDevice retrieves device metadata from OWGW.
func (c *client) GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s", baseURL, url.PathEscape(serial))
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// UpdateDevice updates device metadata or appends notes in OWGW.
func (c *client) UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s", baseURL, url.PathEscape(serial))
	return c.doJSONRequestWithBody(ctx, http.MethodPut, endpoint, body, token, reqID, corrID)
}

// GetDeviceStatus retrieves device live status from OWGW.
func (c *client) GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/status", baseURL, url.PathEscape(serial))
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// GetCapabilities retrieves device hardware/firmware capabilities from OWGW.
func (c *client) GetCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/capabilities", baseURL, url.PathEscape(serial))
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// GetCommands retrieves commands history from OWGW.
func (c *client) GetCommands(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/commands?%s", baseURL, queryParams.Encode())
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// GetCommand retrieves a specific command record by UUID from OWGW: GET /api/v1/command/{commandUUID}.
func (c *client) GetCommand(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/command/%s", baseURL, url.PathEscape(commandUUID))
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// DeleteCommand deletes a specific command record by UUID in OWGW.
func (c *client) DeleteCommand(ctx context.Context, commandUUID, token, reqID, corrID string) error {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/command/%s", baseURL, url.PathEscape(commandUUID))
	_, err = c.doJSONRequest(ctx, http.MethodDelete, endpoint, token, reqID, corrID)
	return err
}

// GetHealthChecks retrieves healthcheck history from OWGW.
func (c *client) GetHealthChecks(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/healthchecks?%s", baseURL, url.PathEscape(serial), queryParams.Encode())
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// GetLogs retrieves device logs from OWGW.
func (c *client) GetLogs(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/logs?%s", baseURL, url.PathEscape(serial), queryParams.Encode())
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// GetStatistics retrieves device statistics from OWGW.
func (c *client) GetStatistics(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/statistics?%s", baseURL, url.PathEscape(serial), queryParams.Encode())
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// ExecuteDeviceCommand sends a device action/command to OWGW: POST /api/v1/device/{serial}/{command}.
func (c *client) ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/%s", baseURL, url.PathEscape(serial), url.PathEscape(command))
	return c.doJSONRequestWithBody(ctx, http.MethodPost, endpoint, body, token, reqID, corrID)
}

// GetDeviceRtty initiates or queries an RTTY session in OWGW: GET /api/v1/device/{serial}/rtty.
func (c *client) GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s/rtty", baseURL, url.PathEscape(serial))
	return c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
}

// DeleteDevice deletes a device from OWGW: DELETE /api/v1/device/{serial}.
func (c *client) DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/device/%s", baseURL, url.PathEscape(serial))
	_, err = c.doJSONRequest(ctx, http.MethodDelete, endpoint, token, reqID, corrID)
	return err
}

// GetBlacklist retrieves blacklisted devices from OWGW: GET /api/v1/blacklist.
func (c *client) GetBlacklist(ctx context.Context, limit, offset int, token, reqID, corrID string) ([]models.BlacklistDevice, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 500
	}
	endpoint := fmt.Sprintf("%s/api/v1/blacklist?limit=%d&offset=%d", baseURL, limit, offset)
	raw, err := c.doJSONRequest(ctx, http.MethodGet, endpoint, token, reqID, corrID)
	if err != nil {
		return nil, err
	}

	var resp models.BlacklistListResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse gateway blacklist response", err)
	}

	return resp.Devices, nil
}

// AddBlacklistDevice adds a device to OWGW blacklist: POST /api/v1/blacklist/{serial}.
func (c *client) AddBlacklistDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/blacklist/%s", baseURL, url.PathEscape(serial))
	return c.doJSONRequestWithBody(ctx, http.MethodPost, endpoint, body, token, reqID, corrID)
}

// DeleteBlacklistDevice deletes a device from OWGW blacklist: DELETE /api/v1/blacklist/{serial}.
func (c *client) DeleteBlacklistDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/api/v1/blacklist/%s", baseURL, url.PathEscape(serial))
	_, err = c.doJSONRequest(ctx, http.MethodDelete, endpoint, token, reqID, corrID)
	return err
}

func (c *client) setHeaders(req *http.Request, token, reqID, corrID string) {
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			req.Header.Set("Authorization", "Bearer "+token)
		} else {
			req.Header.Set("Authorization", token)
		}
	}
	req.Header.Set("User-Agent", userAgent)
	if reqID != "" {
		req.Header.Set("X-Request-Id", reqID)
	}
	if corrID != "" {
		req.Header.Set("X-Correlation-Id", corrID)
	}
}
