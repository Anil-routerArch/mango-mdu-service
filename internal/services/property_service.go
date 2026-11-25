package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"golang.org/x/sync/singleflight"
)

const (
	propertyStatusBatchSize = 100
	propertyOWGWConcurrency = 6
	propertyCacheTTL        = 30 * time.Second
	propertyMaxStaleAge     = 120 * time.Second
	propertyMaxSnapshots    = 16
)

// PropertyService provides normalized hierarchy and current-snapshot reads for
// the Property workspace.
type PropertyService interface {
	GetTree(ctx context.Context, token, reqID, corrID string) (*models.PropertyTreeResponse, error)
	GetMap(ctx context.Context, token, reqID, corrID string) (*models.PropertyMapResponse, error)
	GetPortfolio(ctx context.Context, token, reqID, corrID string) (*models.PropertyPortfolioResponse, error)
	GetOverview(ctx context.Context, scopeType, scopeID, token, reqID, corrID string) (*models.ScopeOverviewResponse, error)
	GetDevices(ctx context.Context, scopeType, scopeID, token, reqID, corrID string, query models.ScopedDevicesQuery) (*models.ScopedDevicesResponse, error)
	UpdateDeviceStatus(eventType, serial string, connected *bool, sanity *float64, ipAddress string)
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
			Name:         node.DeviceNames[normalizeSerial(serial)],
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
	snapshot, dqState, err := s.getOperationalSnapshot(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	allSerials := make(map[string]string)
	for _, node := range snapshot.nodes {
		collectSerials(node, true, allSerials)
	}
	nodes := make([]models.PropertyMapNode, 0, len(snapshot.nodes))
	for _, node := range snapshot.nodes {
		nodes = append(nodes, buildPropertyMapNode(node, snapshot.statuses))
	}
	summary := summarize(allSerials, snapshot.statuses)
	return &models.PropertyMapResponse{Nodes: nodes, DataQuality: makeDataQuality(summary, snapshot.statusErrors, dqState, snapshot.fetchedAt)}, nil
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
	snapshot, dqState, err := s.getOperationalSnapshot(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	responseNodes := cloneTreeNodes(snapshot.nodes)
	allSerials := make(map[string]string)
	for _, node := range responseNodes {
		collectSerials(node, true, allSerials)
	}
	addPortfolioSummaries(responseNodes, snapshot.statuses)
	summary := summarize(allSerials, snapshot.statuses)
	properties, venues, nestedVenues := countPortfolioScopes(responseNodes, "")
	return &models.PropertyPortfolioResponse{
		Nodes: responseNodes, Summary: summary, DataQuality: makeDataQuality(summary, snapshot.statusErrors, dqState, snapshot.fetchedAt),
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
	provClient  propertyProvisioningClient
	gwClient    propertyGatewayClient
	logger      *slog.Logger
	sfGroup     singleflight.Group
	cacheMu     sync.RWMutex
	snapshots   map[string]*operationalSnapshot
	livePatches map[string]deviceStatusPatch
	owgwLimiter chan struct{}
}

func NewPropertyService(provClient propertyProvisioningClient, gwClient propertyGatewayClient, logger ...*slog.Logger) PropertyService {
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &propertyService{
		provClient:  provClient,
		gwClient:    gwClient,
		logger:      l,
		snapshots:   make(map[string]*operationalSnapshot),
		livePatches: make(map[string]deviceStatusPatch),
		owgwLimiter: make(chan struct{}, propertyOWGWConcurrency),
	}
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

// hydrateScopeInventory replaces tree-provided device references with the
// authoritative inventory serials resolved from each OWPROV scope record.
// The structural tree remains lightweight; operational views opt into this.
func (s *propertyService) hydrateScopeInventory(ctx context.Context, node *models.PropertyTreeNode, inventory map[string]models.InventoryTag, token, reqID, corrID string) error {
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
	node.DeviceNames = make(map[string]string, len(tags))
	for _, tag := range tags {
		if strings.TrimSpace(tag.SerialNumber) != "" {
			norm := normalizeSerial(tag.SerialNumber)
			node.Devices = append(node.Devices, tag.SerialNumber)
			node.DeviceNames[norm] = tag.Name
			inventory[norm] = tag
		}
	}
	for i := range node.Children {
		if err := s.hydrateScopeInventory(ctx, &node.Children[i], inventory, token, reqID, corrID); err != nil {
			return err
		}
	}
	return nil
}

func (s *propertyService) getStatuses(ctx context.Context, serials []string, token, reqID, corrID string) (map[string]models.GatewayDeviceStatus, []models.DataQualityError) {
	statuses := make(map[string]models.GatewayDeviceStatus, len(serials))
	if len(serials) == 0 {
		return statuses, nil
	}
	if s.gwClient == nil {
		return statuses, []models.DataQualityError{{Source: "owgw", Code: "STATUS_UNAVAILABLE", Retryable: true}}
	}

	batchCount := (len(serials) + propertyStatusBatchSize - 1) / propertyStatusBatchSize
	workerCount := min(propertyOWGWConcurrency, batchCount)
	jobs := make(chan []string)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errorsOut := make([]models.DataQualityError, 0)
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range jobs {
				select {
				case s.owgwLimiter <- struct{}{}:
				case <-ctx.Done():
					return
				}
				values, err := s.gwClient.GetDevicesStatus(ctx, batch, token, reqID, corrID)
				<-s.owgwLimiter
				mu.Lock()
				if err != nil {
					errorsOut = append(errorsOut, models.DataQualityError{Source: "owgw", Code: "STATUS_BATCH_UNAVAILABLE", Retryable: true})
				} else {
					for _, status := range values {
						statuses[normalizeSerial(status.SerialNumber)] = status
					}
				}
				mu.Unlock()
			}
		}()
	}

enqueueLoop:
	for start := 0; start < len(serials); start += propertyStatusBatchSize {
		end := min(start+propertyStatusBatchSize, len(serials))
		batch := append([]string(nil), serials[start:end]...)
		select {
		case jobs <- batch:
		case <-ctx.Done():
			break enqueueLoop
		}
	}
	close(jobs)
	wg.Wait()
	if ctx.Err() != nil {
		errorsOut = append(errorsOut, models.DataQualityError{Source: "owgw", Code: "STATUS_BATCH_CANCELLED", Retryable: true})
	}
	return statuses, errorsOut
}

type operationalSnapshot struct {
	nodes        []models.PropertyTreeNode
	inventory    map[string]models.InventoryTag
	statuses     map[string]models.GatewayDeviceStatus
	statusErrors []models.DataQualityError
	fetchedAt    time.Time
	fingerprint  string
}

type deviceStatusPatch struct {
	connected   *bool
	connectedAt time.Time
	sanity      *float64
	sanitySet   bool
	sanityAt    time.Time
	ipAddress   string
	ipAddressAt time.Time
}

func cloneTreeNodes(nodes []models.PropertyTreeNode) []models.PropertyTreeNode {
	out := make([]models.PropertyTreeNode, len(nodes))
	for i, node := range nodes {
		out[i] = node
		out[i].Devices = append([]string(nil), node.Devices...)
		out[i].Children = cloneTreeNodes(node.Children)
		if node.Summary != nil {
			summary := *node.Summary
			out[i].Summary = &summary
		}
		if node.DeviceNames != nil {
			out[i].DeviceNames = make(map[string]string, len(node.DeviceNames))
			for key, value := range node.DeviceNames {
				out[i].DeviceNames[key] = value
			}
		}
	}
	return out
}

// operationalSnapshotView captures immutable maps/slices while cacheMu is held.
// Cached snapshots use copy-on-write for live status updates, so requests do not
// need to copy a 50,000-Device inventory on every filter or pagination change.
func operationalSnapshotView(snapshot *operationalSnapshot) *operationalSnapshot {
	if snapshot == nil {
		return nil
	}
	return &operationalSnapshot{
		nodes:        snapshot.nodes,
		inventory:    snapshot.inventory,
		statuses:     snapshot.statuses,
		statusErrors: append([]models.DataQualityError(nil), snapshot.statusErrors...),
		fetchedAt:    snapshot.fetchedAt,
		fingerprint:  snapshot.fingerprint,
	}
}

func operationalFingerprint(nodes []models.PropertyTreeNode) string {
	parts := make([]string, 0)
	var visit func([]models.PropertyTreeNode)
	visit = func(items []models.PropertyTreeNode) {
		for _, node := range items {
			parts = append(parts, node.Type+":"+node.ID)
			for _, serial := range node.Devices {
				parts = append(parts, node.Type+":"+node.ID+"/"+normalizeSerial(serial))
			}
			visit(node.Children)
		}
	}
	visit(nodes)
	sort.Strings(parts)
	hash := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(hash[:])
}

func principalCacheKey(token string) string {
	if token == "" {
		return ""
	}
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func applyDeviceStatusPatch(status models.GatewayDeviceStatus, patch deviceStatusPatch, newerThan time.Time) models.GatewayDeviceStatus {
	if patch.connected != nil && patch.connectedAt.After(newerThan) {
		status.Connected = *patch.connected
	}
	if patch.sanitySet && patch.sanityAt.After(newerThan) {
		status.Sanity = patch.sanity
	}
	if patch.ipAddress != "" && patch.ipAddressAt.After(newerThan) {
		status.IPAddress = patch.ipAddress
	}
	return status
}

// UpdateDeviceStatus merges only fields carried by a Kafka event. State events
// contain no availability or sanity fields and therefore cannot overwrite a
// complete OWGW status with Go zero values.
func (s *propertyService) UpdateDeviceStatus(eventType, serial string, connected *bool, sanity *float64, ipAddress string) {
	norm := normalizeSerial(serial)
	if norm == "" {
		return
	}
	now := time.Now()
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	patch := s.livePatches[norm]
	changed := false
	if connected != nil {
		value := *connected
		patch.connected = &value
		patch.connectedAt = now
		changed = true
	}
	if eventType == "healthcheck" {
		patch.sanity = sanity
		patch.sanitySet = true
		patch.sanityAt = now
		changed = true
	}
	if ipAddress != "" {
		patch.ipAddress = ipAddress
		patch.ipAddressAt = now
		changed = true
	}
	if !changed {
		return
	}
	s.livePatches[norm] = patch
	for _, snap := range s.snapshots {
		if cur, ok := snap.statuses[norm]; ok {
			nextStatuses := make(map[string]models.GatewayDeviceStatus, len(snap.statuses))
			for key, value := range snap.statuses {
				nextStatuses[key] = value
			}
			nextStatuses[norm] = applyDeviceStatusPatch(cur, patch, time.Time{})
			snap.statuses = nextStatuses
		}
	}
}

func (s *propertyService) fetchOperationalSnapshot(ctx context.Context, token, reqID, corrID string) (*operationalSnapshot, error) {
	refreshStarted := time.Now()
	tree, err := s.GetTree(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	inventory := make(map[string]models.InventoryTag)
	for i := range tree.Nodes {
		if err := s.hydrateScopeInventory(ctx, &tree.Nodes[i], inventory, token, reqID, corrID); err != nil {
			return nil, err
		}
	}
	serials := make([]string, 0, len(inventory))
	for _, tag := range inventory {
		serials = append(serials, tag.SerialNumber)
	}
	statuses, statusErrors := s.getStatuses(ctx, serials, token, reqID, corrID)

	s.cacheMu.RLock()
	for norm, status := range statuses {
		if patch, ok := s.livePatches[norm]; ok {
			statuses[norm] = applyDeviceStatusPatch(status, patch, refreshStarted)
		}
	}
	s.cacheMu.RUnlock()

	return &operationalSnapshot{
		nodes: tree.Nodes, inventory: inventory, statuses: statuses,
		statusErrors: statusErrors, fetchedAt: time.Now(), fingerprint: operationalFingerprint(tree.Nodes),
	}, nil
}

func (s *propertyService) refreshAndStoreOperationalSnapshot(ctx context.Context, token, reqID, corrID, fullKey string) (*operationalSnapshot, error) {
	refreshed, err := s.fetchOperationalSnapshot(ctx, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if current := s.snapshots[fullKey]; current != nil && current.fingerprint == refreshed.fingerprint && time.Since(current.fetchedAt) < propertyMaxStaleAge && len(current.statusErrors) == 0 && len(refreshed.statusErrors) > 0 {
		return nil, fmt.Errorf("operational snapshot refresh was incomplete")
	}
	s.snapshots[fullKey] = refreshed
	now := time.Now()
	for key, snapshot := range s.snapshots {
		if now.Sub(snapshot.fetchedAt) > 10*time.Minute {
			delete(s.snapshots, key)
		}
	}
	for len(s.snapshots) > propertyMaxSnapshots {
		oldestKey := ""
		var oldestAt time.Time
		for key, snapshot := range s.snapshots {
			if key == fullKey {
				continue
			}
			if oldestKey == "" || snapshot.fetchedAt.Before(oldestAt) {
				oldestKey, oldestAt = key, snapshot.fetchedAt
			}
		}
		if oldestKey == "" {
			break
		}
		delete(s.snapshots, oldestKey)
	}
	for serial, patch := range s.livePatches {
		latest := patch.connectedAt
		if patch.sanityAt.After(latest) {
			latest = patch.sanityAt
		}
		if patch.ipAddressAt.After(latest) {
			latest = patch.ipAddressAt
		}
		if now.Sub(latest) > propertyMaxStaleAge {
			delete(s.livePatches, serial)
		}
	}
	return operationalSnapshotView(refreshed), nil
}

func (s *propertyService) getOperationalSnapshot(ctx context.Context, token, reqID, corrID string) (*operationalSnapshot, string, error) {
	principalKey := principalCacheKey(token)
	if principalKey == "" {
		snapshot, err := s.fetchOperationalSnapshot(ctx, token, reqID, corrID)
		if err != nil {
			return nil, "unavailable", err
		}
		state := "complete"
		if len(snapshot.statusErrors) > 0 {
			state = "partial"
		}
		return operationalSnapshotView(snapshot), state, nil
	}
	fullKey := principalKey + ":portfolio"
	s.cacheMu.RLock()
	cached := s.snapshots[fullKey]
	if cached != nil {
		age := time.Since(cached.fetchedAt)
		if age < propertyCacheTTL {
			out := operationalSnapshotView(cached)
			s.cacheMu.RUnlock()
			state := "complete"
			if len(out.statusErrors) > 0 {
				state = "partial"
			}
			return out, state, nil
		}
		if age < propertyMaxStaleAge {
			out := operationalSnapshotView(cached)
			s.cacheMu.RUnlock()
			_ = s.sfGroup.DoChan(fullKey, func() (any, error) {
				bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				return s.refreshAndStoreOperationalSnapshot(bgCtx, token, reqID, corrID, fullKey)
			})
			return out, "stale", nil
		}
	}
	s.cacheMu.RUnlock()

	value, err, _ := s.sfGroup.Do(fullKey, func() (any, error) {
		return s.refreshAndStoreOperationalSnapshot(ctx, token, reqID, corrID, fullKey)
	})
	if err != nil {
		return nil, "unavailable", err
	}
	snapshot, ok := value.(*operationalSnapshot)
	if !ok || snapshot == nil {
		return nil, "unavailable", fmt.Errorf("invalid operational snapshot")
	}
	state := "complete"
	if len(snapshot.statusErrors) > 0 {
		state = "partial"
	}
	return snapshot, state, nil
}

func (s *propertyService) getScopeOperationalSnapshot(ctx context.Context, scopeType, scopeID, token, reqID, corrID string) (*operationalSnapshot, *models.PropertyTreeNode, string, error) {
	snapshot, state, err := s.getOperationalSnapshot(ctx, token, reqID, corrID)
	if err != nil {
		return nil, nil, state, err
	}
	if node, ok := findScope(snapshot.nodes, scopeType, scopeID); ok {
		return snapshot, node, state, nil
	}

	// A newly created authorized scope may not be present in a still-fresh
	// snapshot. Revalidate the lightweight hierarchy before forcing a costly
	// operational refresh, so arbitrary unauthorized IDs cannot bypass caching.
	tree, treeErr := s.GetTree(ctx, token, reqID, corrID)
	if treeErr != nil {
		return nil, nil, "unavailable", treeErr
	}
	if _, authorized := findScope(tree.Nodes, scopeType, scopeID); !authorized {
		return nil, nil, state, models.NewApiError(404, "Not Found", "Property or venue not found in the authorized hierarchy")
	}
	principalKey := principalCacheKey(token)
	if principalKey == "" {
		return nil, nil, state, models.NewApiError(404, "Not Found", "Property or venue not found in the authorized hierarchy")
	}
	fullKey := principalKey + ":portfolio"
	s.cacheMu.Lock()
	delete(s.snapshots, fullKey)
	s.cacheMu.Unlock()
	snapshot, state, err = s.getOperationalSnapshot(ctx, token, reqID, corrID)
	if err != nil {
		return nil, nil, state, err
	}
	node, ok := findScope(snapshot.nodes, scopeType, scopeID)
	if !ok {
		return nil, nil, state, models.NewApiError(404, "Not Found", "Property or venue not found in the authorized hierarchy")
	}
	return snapshot, node, state, nil
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

func makeDataQuality(summary models.ScopeSummary, errs []models.DataQualityError, state string, asOf time.Time) models.DataQuality {
	if state == "" {
		state = "complete"
	}
	return models.DataQuality{
		State:              state,
		Partial:            len(errs) > 0,
		UnknownDeviceCount: summary.UnknownDevices,
		AsOf:               asOf.UTC().Format(time.RFC3339),
		Errors:             errs,
	}
}

func (s *propertyService) GetOverview(ctx context.Context, scopeType, scopeID, token, reqID, corrID string) (*models.ScopeOverviewResponse, error) {
	snapshot, node, dqState, err := s.getScopeOperationalSnapshot(ctx, scopeType, scopeID, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	serialMap := make(map[string]string)
	collectSerials(*node, true, serialMap)
	summary := summarize(serialMap, snapshot.statuses)

	children := make([]models.ChildScopeSummary, 0, len(node.Children))
	for _, child := range node.Children {
		childSerials := make(map[string]string)
		collectSerials(child, true, childSerials)
		children = append(children, models.ChildScopeSummary{
			ID: child.ID, Type: child.Type, Name: child.Name, Summary: summarize(childSerials, snapshot.statuses),
		})
	}
	return &models.ScopeOverviewResponse{
		ID: node.ID, Type: node.Type, Name: node.Name, Summary: summary, Children: children,
		DataQuality: makeDataQuality(summary, snapshot.statusErrors, dqState, snapshot.fetchedAt), IncludesDesc: true,
	}, nil
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
		item.Sanity = status.Sanity
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
	switch strings.ToLower(q.Attention) {
	case "low-sanity":
		if item.Availability == "offline" || item.Sanity == nil || *item.Sanity >= 90 {
			return false
		}
	case "sanity-unavailable":
		if item.Availability == "offline" || item.Sanity != nil {
			return false
		}
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
	snapshot, node, dqState, err := s.getScopeOperationalSnapshot(ctx, scopeType, scopeID, token, reqID, corrID)
	if err != nil {
		return nil, err
	}
	serialMap := make(map[string]string)
	collectSerials(*node, true, serialMap)
	tags := make([]models.InventoryTag, 0, len(serialMap))
	for norm := range serialMap {
		if tag, exists := snapshot.inventory[norm]; exists {
			tags = append(tags, tag)
		}
	}
	summary := summarize(serialMap, snapshot.statuses)
	items := make([]models.PropertyDeviceItem, 0, len(tags))
	for _, tag := range tags {
		item := propertyDeviceItem(tag, snapshot.statuses)
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
	var pageDevices []models.PropertyDeviceItem
	if total == 0 || q.PageIndex >= totalPages {
		pageDevices = []models.PropertyDeviceItem{}
	} else {
		start := q.PageIndex * q.PageSize
		if start < 0 || start > total {
			start = total
		}
		end := start + q.PageSize
		if end < start || end > total {
			end = total
		}
		pageDevices = items[start:end]
	}
	return &models.ScopedDevicesResponse{
		Devices: pageDevices, TotalCount: total, PageIndex: q.PageIndex, PageSize: q.PageSize,
		TotalPages: totalPages, Summary: summary, DataQuality: makeDataQuality(summary, snapshot.statusErrors, dqState, snapshot.fetchedAt),
	}, nil
}
