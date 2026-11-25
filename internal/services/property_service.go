package services

import (
	"context"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

const propertyStatusBatchSize = 100

// PropertyService provides normalized hierarchy and current-snapshot reads for
// the Property workspace.
type PropertyService interface {
	GetTree(ctx context.Context, token, reqID, corrID string) (*models.PropertyTreeResponse, error)
	GetMap(ctx context.Context, token, reqID, corrID string) (*models.PropertyMapResponse, error)
	GetPortfolio(ctx context.Context, token, reqID, corrID string) (*models.PropertyPortfolioResponse, error)
	GetOverview(ctx context.Context, scopeType, scopeID, token, reqID, corrID string) (*models.ScopeOverviewResponse, error)
	GetDevices(ctx context.Context, scopeType, scopeID, token, reqID, corrID string, query models.ScopedDevicesQuery) (*models.ScopedDevicesResponse, error)
}

func mapAvailability(serial string, statuses map[string]models.GatewayDeviceStatus) string {
	status, exists := statuses[normalizeSerial(serial)]
	if !exists {
		return "unknown"
	}
	if status.Connected {
		return "online"
	}
	return "offline"
}

func buildPropertyMapNode(node models.PropertyTreeNode, statuses map[string]models.GatewayDeviceStatus) models.PropertyMapNode {
	devices := make([]models.PropertyMapDevice, 0, len(node.Devices))
	for _, serial := range node.Devices {
		status, hasStatus := statuses[normalizeSerial(serial)]
		var sanity *float64
		if hasStatus {
			sanity = status.Sanity
		}
		devices = append(devices, models.PropertyMapDevice{
			SerialNumber: serial,
			Availability: mapAvailability(serial, statuses),
			Sanity:       sanity,
		})
	}
	children := make([]models.PropertyMapNode, 0, len(node.Children))
	for _, child := range node.Children {
		children = append(children, buildPropertyMapNode(child, statuses))
	}
	return models.PropertyMapNode{
		ID: node.ID, Type: node.Type, Name: node.Name, Devices: devices, Children: children,
	}
}

// GetMap resolves direct Device assignments only when the hierarchy map is
// opened. The regular tree remains structural and lightweight.
func (s *propertyService) GetMap(ctx context.Context, token, reqID, corrID string) (*models.PropertyMapResponse, error) {
	tree, err := s.GetTree(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	for i := range tree.Nodes {
		if err := s.hydrateScopeInventory(ctx, &tree.Nodes[i], token, reqID, corrID); err != nil {
			return nil, err
		}
	}
	allSerials := make(map[string]string)
	for _, node := range tree.Nodes {
		collectSerials(node, true, allSerials)
	}
	serials := make([]string, 0, len(allSerials))
	for _, serial := range allSerials {
		serials = append(serials, serial)
	}
	statuses, statusErrors := s.getStatuses(ctx, serials, token, reqID, corrID)
	nodes := make([]models.PropertyMapNode, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		nodes = append(nodes, buildPropertyMapNode(node, statuses))
	}
	summary := summarize(allSerials, statuses)
	return &models.PropertyMapResponse{Nodes: nodes, DataQuality: makeDataQuality(summary, statusErrors)}, nil
}

func addPortfolioSummaries(nodes []models.PropertyTreeNode, statuses map[string]models.GatewayDeviceStatus) {
	for i := range nodes {
		serials := make(map[string]string)
		collectSerials(nodes[i], true, serials)
		summary := summarize(serials, statuses)
		nodes[i].Summary = &summary
		addPortfolioSummaries(nodes[i].Children, statuses)
	}
}

func countPortfolioScopes(nodes []models.PropertyTreeNode, parentType string) (properties, venues, nestedVenues int) {
	for _, node := range nodes {
		if node.Type == "property" {
			properties++
		} else {
			venues++
			if parentType == "venue" {
				nestedVenues++
			}
		}
		p, v, n := countPortfolioScopes(node.Children, node.Type)
		properties, venues, nestedVenues = properties+p, venues+v, nestedVenues+n
	}
	return
}

func (s *propertyService) GetPortfolio(ctx context.Context, token, reqID, corrID string) (*models.PropertyPortfolioResponse, error) {
	tree, err := s.GetTree(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	for i := range tree.Nodes {
		if err := s.hydrateScopeInventory(ctx, &tree.Nodes[i], token, reqID, corrID); err != nil {
			return nil, err
		}
	}
	allSerials := make(map[string]string)
	for _, node := range tree.Nodes {
		collectSerials(node, true, allSerials)
	}
	serials := make([]string, 0, len(allSerials))
	for _, serial := range allSerials {
		serials = append(serials, serial)
	}
	statuses, statusErrors := s.getStatuses(ctx, serials, token, reqID, corrID)
	addPortfolioSummaries(tree.Nodes, statuses)
	summary := summarize(allSerials, statuses)
	properties, venues, nestedVenues := countPortfolioScopes(tree.Nodes, "")
	return &models.PropertyPortfolioResponse{
		Nodes: tree.Nodes, Summary: summary, DataQuality: makeDataQuality(summary, statusErrors),
		PropertyCount: properties, VenueCount: venues, NestedVenueCount: nestedVenues,
	}, nil
}

type propertyProvisioningClient interface {
	GetPropertyTree(ctx context.Context, token, reqID, corrID string) (*models.OWProvTreeNode, error)
	GetInventoryForScope(ctx context.Context, entityID, venueID, token, reqID, corrID string) ([]models.InventoryTag, error)
}

type propertyGatewayClient interface {
	GetDevicesStatus(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error)
}

type propertyService struct {
	provClient propertyProvisioningClient
	gwClient   propertyGatewayClient
	logger     *slog.Logger
}

func NewPropertyService(provClient propertyProvisioningClient, gwClient propertyGatewayClient, logger ...*slog.Logger) PropertyService {
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &propertyService{provClient: provClient, gwClient: gwClient, logger: l}
}

func normalizeTreeNode(node models.OWProvTreeNode) models.PropertyTreeNode {
	nodeType := "venue"
	if strings.EqualFold(node.Type, "entity") {
		nodeType = "property"
	}
	children := make([]models.PropertyTreeNode, 0, len(node.Children)+len(node.Venues))
	for _, child := range node.Children {
		children = append(children, normalizeTreeNode(child))
	}
	for _, venue := range node.Venues {
		children = append(children, normalizeTreeNode(venue))
	}
	return models.PropertyTreeNode{
		ID:       node.UUID,
		Type:     nodeType,
		Name:     node.Name,
		Devices:  append([]string(nil), node.Devices...),
		Children: children,
	}
}

func (s *propertyService) GetTree(ctx context.Context, token, reqID, corrID string) (*models.PropertyTreeResponse, error) {
	root, err := s.provClient.GetPropertyTree(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	if root == nil || strings.TrimSpace(root.UUID) == "" {
		return &models.PropertyTreeResponse{Nodes: []models.PropertyTreeNode{}}, nil
	}
	// OWPROV uses this non-resource wrapper when the user has multiple assigned
	// root scopes. It must never become a selectable Property or an inventory
	// query target.
	if root.UUID == "0000-0000-0000" && strings.EqualFold(root.Name, "Assigned Scopes") {
		nodes := make([]models.PropertyTreeNode, 0, len(root.Children)+len(root.Venues))
		for _, child := range root.Children {
			nodes = append(nodes, normalizeTreeNode(child))
		}
		for _, venue := range root.Venues {
			nodes = append(nodes, normalizeTreeNode(venue))
		}
		return &models.PropertyTreeResponse{Nodes: nodes}, nil
	}
	return &models.PropertyTreeResponse{Nodes: []models.PropertyTreeNode{normalizeTreeNode(*root)}}, nil
}

func findScope(nodes []models.PropertyTreeNode, scopeType, id string) (*models.PropertyTreeNode, bool) {
	for i := range nodes {
		n := &nodes[i]
		if n.ID == id && n.Type == scopeType {
			return n, true
		}
		if found, ok := findScope(n.Children, scopeType, id); ok {
			return found, true
		}
	}
	return nil, false
}

func collectSerials(node models.PropertyTreeNode, recursive bool, out map[string]string) {
	for _, serial := range node.Devices {
		if norm := normalizeSerial(serial); norm != "" {
			out[norm] = serial
		}
	}
	if recursive {
		for _, child := range node.Children {
			collectSerials(child, true, out)
		}
	}
}

type scopeRef struct {
	typ string
	id  string
}

func collectScopeRefs(node models.PropertyTreeNode, recursive bool, refs *[]scopeRef) {
	*refs = append(*refs, scopeRef{typ: node.Type, id: node.ID})
	if recursive {
		for _, child := range node.Children {
			collectScopeRefs(child, true, refs)
		}
	}
}

// hydrateScopeInventory replaces tree-provided device references with the
// authoritative inventory serials resolved from each OWPROV scope record.
// The structural tree remains lightweight; operational views opt into this.
func (s *propertyService) hydrateScopeInventory(ctx context.Context, node *models.PropertyTreeNode, token, reqID, corrID string) error {
	entityID, venueID := "", ""
	if node.Type == "property" {
		entityID = node.ID
	} else {
		venueID = node.ID
	}
	tags, err := s.provClient.GetInventoryForScope(ctx, entityID, venueID, token, reqID, corrID)
	if err != nil {
		return err
	}
	node.Devices = node.Devices[:0]
	for _, tag := range tags {
		if strings.TrimSpace(tag.SerialNumber) != "" {
			node.Devices = append(node.Devices, tag.SerialNumber)
		}
	}
	for i := range node.Children {
		if err := s.hydrateScopeInventory(ctx, &node.Children[i], token, reqID, corrID); err != nil {
			return err
		}
	}
	return nil
}

func (s *propertyService) loadScope(ctx context.Context, scopeType, scopeID, token, reqID, corrID string) (*models.PropertyTreeNode, error) {
	tree, err := s.GetTree(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	node, ok := findScope(tree.Nodes, scopeType, scopeID)
	if !ok {
		return nil, models.NewApiError(404, "Not Found", "Property or venue not found in the authorized hierarchy")
	}
	return node, nil
}

func (s *propertyService) getStatuses(ctx context.Context, serials []string, token, reqID, corrID string) (map[string]models.GatewayDeviceStatus, []models.DataQualityError) {
	statuses := make(map[string]models.GatewayDeviceStatus, len(serials))
	if len(serials) == 0 {
		return statuses, nil
	}
	if s.gwClient == nil {
		return statuses, []models.DataQualityError{{Source: "owgw", Code: "STATUS_UNAVAILABLE", Retryable: true}}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	errorsOut := make([]models.DataQualityError, 0)
	for start := 0; start < len(serials); start += propertyStatusBatchSize {
		end := start + propertyStatusBatchSize
		if end > len(serials) {
			end = len(serials)
		}
		batch := append([]string(nil), serials[start:end]...)
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			values, err := s.gwClient.GetDevicesStatus(ctx, batch, token, reqID, corrID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errorsOut = append(errorsOut, models.DataQualityError{Source: "owgw", Code: "STATUS_BATCH_UNAVAILABLE", Retryable: true})
				return
			}
			for _, status := range values {
				statuses[normalizeSerial(status.SerialNumber)] = status
			}
		}()
	}
	wg.Wait()
	return statuses, errorsOut
}

func summarize(serials map[string]string, statuses map[string]models.GatewayDeviceStatus) models.ScopeSummary {
	summary := models.ScopeSummary{TotalDevices: len(serials)}
	for norm := range serials {
		status, exists := statuses[norm]
		if !exists {
			summary.UnknownDevices++
			continue
		}
		summary.StatusMeasuredFor++
		if status.Connected {
			summary.OnlineDevices++
		} else {
			summary.OfflineDevices++
		}
	}
	return summary
}

func makeDataQuality(summary models.ScopeSummary, errs []models.DataQualityError) models.DataQuality {
	state := "complete"
	partial := len(errs) > 0
	if partial {
		state = "partial"
	}
	return models.DataQuality{
		State:              state,
		Partial:            partial,
		UnknownDeviceCount: summary.UnknownDevices,
		AsOf:               time.Now().UTC().Format(time.RFC3339),
		Errors:             errs,
	}
}

func (s *propertyService) GetOverview(ctx context.Context, scopeType, scopeID, token, reqID, corrID string) (*models.ScopeOverviewResponse, error) {
	node, err := s.loadScope(ctx, scopeType, scopeID, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateScopeInventory(ctx, node, token, reqID, corrID); err != nil {
		return nil, err
	}
	serialMap := make(map[string]string)
	collectSerials(*node, true, serialMap)
	serials := make([]string, 0, len(serialMap))
	for _, serial := range serialMap {
		serials = append(serials, serial)
	}
	statuses, statusErrors := s.getStatuses(ctx, serials, token, reqID, corrID)
	summary := summarize(serialMap, statuses)

	children := make([]models.ChildScopeSummary, 0, len(node.Children))
	for _, child := range node.Children {
		childSerials := make(map[string]string)
		collectSerials(child, true, childSerials)
		children = append(children, models.ChildScopeSummary{
			ID: child.ID, Type: child.Type, Name: child.Name, Summary: summarize(childSerials, statuses),
		})
	}
	return &models.ScopeOverviewResponse{
		ID: node.ID, Type: node.Type, Name: node.Name, Summary: summary, Children: children,
		DataQuality: makeDataQuality(summary, statusErrors), IncludesDesc: true,
	}, nil
}

func (s *propertyService) loadInventory(ctx context.Context, node models.PropertyTreeNode, token, reqID, corrID string) ([]models.InventoryTag, error) {
	refs := make([]scopeRef, 0)
	collectScopeRefs(node, true, &refs)
	bySerial := make(map[string]models.InventoryTag)
	for _, ref := range refs {
		entityID, venueID := "", ""
		if ref.typ == "property" {
			entityID = ref.id
		} else {
			venueID = ref.id
		}
		tags, err := s.provClient.GetInventoryForScope(ctx, entityID, venueID, token, reqID, corrID)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			if norm := normalizeSerial(tag.SerialNumber); norm != "" {
				bySerial[norm] = tag
			}
		}
	}
	tags := make([]models.InventoryTag, 0, len(bySerial))
	for _, tag := range bySerial {
		tags = append(tags, tag)
	}
	return tags, nil
}

func propertyDeviceItem(tag models.InventoryTag, statuses map[string]models.GatewayDeviceStatus) models.PropertyDeviceItem {
	status, known := statuses[normalizeSerial(tag.SerialNumber)]
	availability := "unknown"
	if known && status.Connected {
		availability = "online"
	} else if known {
		availability = "offline"
	}
	item := models.PropertyDeviceItem{
		SerialNumber: tag.SerialNumber, Name: tag.Name, Description: tag.Description,
		DeviceType: tag.DeviceType, DeviceGroup: tag.DevClass, EntityID: tag.Entity,
		VenueID: tag.Venue, Availability: availability,
		Connected: known && status.Connected, StatusKnown: known,
	}
	if item.Name == "" {
		item.Name = tag.SerialNumber
	}
	if tag.ExtendedInfo != nil {
		if tag.ExtendedInfo.Entity != nil {
			item.EntityID, item.EntityName = tag.ExtendedInfo.Entity.ID, tag.ExtendedInfo.Entity.Name
		}
		if tag.ExtendedInfo.Venue != nil {
			item.VenueID, item.VenueName = tag.ExtendedInfo.Venue.ID, tag.ExtendedInfo.Venue.Name
		}
	}
	if known {
		item.IPAddress, item.Firmware, item.LastContact = status.IPAddress, status.Firmware, status.LastContact
		if item.DeviceType == "" {
			item.DeviceType = status.Compatible
		}
		if item.DeviceGroup == "" {
			item.DeviceGroup = strings.ToLower(status.DeviceType)
		}
	}
	return item
}

func matchesDevice(item models.PropertyDeviceItem, q models.ScopedDevicesQuery) bool {
	if q.Search != "" {
		needle := strings.ToLower(strings.TrimSpace(q.Search))
		haystack := strings.ToLower(item.Name + " " + item.SerialNumber + " " + item.Description)
		if !strings.Contains(haystack, needle) && !strings.Contains(normalizeSerial(item.SerialNumber), normalizeSerial(needle)) {
			return false
		}
	}
	if q.VenueID != "" && item.VenueID != q.VenueID {
		return false
	}
	if q.Availability != "" && item.Availability != strings.ToLower(q.Availability) {
		return false
	}
	if q.DeviceType != "" && !strings.EqualFold(item.DeviceType, q.DeviceType) && !strings.EqualFold(item.DeviceGroup, q.DeviceType) {
		return false
	}
	return true
}

func (s *propertyService) GetDevices(ctx context.Context, scopeType, scopeID, token, reqID, corrID string, q models.ScopedDevicesQuery) (*models.ScopedDevicesResponse, error) {
	if q.PageIndex < 0 {
		q.PageIndex = 0
	}
	if q.PageSize <= 0 || q.PageSize > 500 {
		q.PageSize = 25
	}
	node, err := s.loadScope(ctx, scopeType, scopeID, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	tags, err := s.loadInventory(ctx, *node, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	serials := make([]string, 0, len(tags))
	serialMap := make(map[string]string, len(tags))
	for _, tag := range tags {
		if norm := normalizeSerial(tag.SerialNumber); norm != "" {
			serials = append(serials, tag.SerialNumber)
			serialMap[norm] = tag.SerialNumber
		}
	}
	statuses, statusErrors := s.getStatuses(ctx, serials, token, reqID, corrID)
	summary := summarize(serialMap, statuses)
	items := make([]models.PropertyDeviceItem, 0, len(tags))
	for _, tag := range tags {
		item := propertyDeviceItem(tag, statuses)
		if matchesDevice(item, q) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if strings.EqualFold(items[i].Name, items[j].Name) {
			return items[i].SerialNumber < items[j].SerialNumber
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	total := len(items)
	totalPages := 1
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(q.PageSize)))
	}
	start := q.PageIndex * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}
	return &models.ScopedDevicesResponse{
		Devices: items[start:end], TotalCount: total, PageIndex: q.PageIndex, PageSize: q.PageSize,
		TotalPages: totalPages, Summary: summary, DataQuality: makeDataQuality(summary, statusErrors),
	}, nil
}
