package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

type fakePropertyProv struct {
	tree         *models.OWProvTreeNode
	inventory    map[string][]models.InventoryTag
	inventoryErr map[string]error
}

func (f *fakePropertyProv) GetPropertyTree(context.Context, string, string, string) (*models.OWProvTreeNode, error) {
	return f.tree, nil
}

func (f *fakePropertyProv) GetInventoryForScope(_ context.Context, entityID, venueID, _, _, _ string) ([]models.InventoryTag, error) {
	key := "entity:" + entityID
	if venueID != "" {
		key = "venue:" + venueID
	}
	if err := f.inventoryErr[key]; err != nil {
		return nil, err
	}
	return append([]models.InventoryTag(nil), f.inventory[key]...), nil
}

type fakePropertyGW struct {
	statuses []models.GatewayDeviceStatus
	err      error
}

func (f *fakePropertyGW) GetDevicesStatus(context.Context, []string, string, string, string) ([]models.GatewayDeviceStatus, error) {
	return append([]models.GatewayDeviceStatus(nil), f.statuses...), f.err
}

func TestPropertyTreeFlattensAssignedScopesWrapper(t *testing.T) {
	service := NewPropertyService(&fakePropertyProv{tree: &models.OWProvTreeNode{
		Type: "entity", Name: "Assigned Scopes", UUID: "0000-0000-0000",
		Children: []models.OWProvTreeNode{{Type: "entity", Name: "Property A", UUID: "property-a"}},
		Venues:   []models.OWProvTreeNode{{Type: "venue", Name: "Venue B", UUID: "venue-b"}},
	}}, &fakePropertyGW{})

	tree, err := service.GetTree(context.Background(), "token", "req", "corr")
	if err != nil {
		t.Fatalf("GetTree returned error: %v", err)
	}
	if len(tree.Nodes) != 2 {
		t.Fatalf("expected 2 real roots, got %d", len(tree.Nodes))
	}
	if tree.Nodes[0].Type != "property" || tree.Nodes[1].Type != "venue" {
		t.Fatalf("unexpected normalized types: %+v", tree.Nodes)
	}
	for _, node := range tree.Nodes {
		if node.ID == "0000-0000-0000" {
			t.Fatal("synthetic Assigned Scopes wrapper must not be selectable")
		}
	}
}

func TestPropertyTreeResponseOmitsDeviceSerials(t *testing.T) {
	service := NewPropertyService(&fakePropertyProv{tree: &models.OWProvTreeNode{
		Type: "entity", Name: "Property A", UUID: "property-a", Devices: []string{"sensitive-serial"},
		Venues: []models.OWProvTreeNode{{
			Type: "venue", Name: "Venue A", UUID: "venue-a", Devices: []string{"nested-sensitive-serial"},
		}},
	}}, &fakePropertyGW{})

	tree, err := service.GetTree(context.Background(), "token", "req", "corr")
	if err != nil {
		t.Fatalf("GetTree returned error: %v", err)
	}
	payload, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("marshal tree response: %v", err)
	}
	serialized := string(payload)
	if strings.Contains(serialized, "devices") || strings.Contains(serialized, "sensitive-serial") {
		t.Fatalf("structural tree exposed Device inventory: %s", serialized)
	}
	if len(tree.Nodes) != 1 || len(tree.Nodes[0].Devices) != 1 {
		t.Fatalf("internal tree must retain Device serials for aggregation: %+v", tree.Nodes)
	}
}

func TestPropertyPortfolioSummarizesEveryHierarchyRowInOneSnapshot(t *testing.T) {
	service := NewPropertyService(&fakePropertyProv{
		tree: &models.OWProvTreeNode{
			Type: "entity", Name: "Property A", UUID: "property-a",
			Venues: []models.OWProvTreeNode{{
				Type: "venue", Name: "Lobby", UUID: "venue-a",
				Children: []models.OWProvTreeNode{{Type: "venue", Name: "Reception", UUID: "venue-b"}},
			}},
		},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {{SerialNumber: "aa"}},
			"venue:venue-a":     {{SerialNumber: "bb"}},
			"venue:venue-b":     {{SerialNumber: "cc"}},
		},
	}, &fakePropertyGW{statuses: []models.GatewayDeviceStatus{
		{SerialNumber: "aa", Connected: true},
		{SerialNumber: "bb", Connected: true},
		{SerialNumber: "cc", Connected: false},
	}})

	portfolio, err := service.GetPortfolio(context.Background(), "token", "req", "corr")
	if err != nil {
		t.Fatalf("GetPortfolio returned error: %v", err)
	}
	if portfolio.PropertyCount != 1 || portfolio.VenueCount != 2 || portfolio.NestedVenueCount != 1 {
		t.Fatalf("unexpected scope counts: %+v", portfolio)
	}
	if portfolio.Summary.TotalDevices != 3 || portfolio.Summary.OnlineDevices != 2 || portfolio.Summary.OfflineDevices != 1 {
		t.Fatalf("unexpected portfolio summary: %+v", portfolio.Summary)
	}
	root := portfolio.Nodes[0]
	if root.Summary == nil || root.Summary.TotalDevices != 3 || root.Children[0].Summary == nil || root.Children[0].Summary.TotalDevices != 2 {
		t.Fatalf("expected recursive row summaries, got %+v", root)
	}
}

func TestPropertyMapReturnsDirectDevicesWithFactualAvailability(t *testing.T) {
	healthySanity := 97.0
	lowSanity := 65.0
	service := NewPropertyService(&fakePropertyProv{
		tree: &models.OWProvTreeNode{
			Type: "entity", Name: "Property A", UUID: "property-a",
			Venues: []models.OWProvTreeNode{{Type: "venue", Name: "Lobby", UUID: "venue-a"}},
		},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {{SerialNumber: "property-device", Name: "Property AP"}},
			"venue:venue-a":     {{SerialNumber: "venue-device", Name: "Venue AP"}},
		},
	}, &fakePropertyGW{statuses: []models.GatewayDeviceStatus{
		{SerialNumber: "property-device", Connected: true, Sanity: &healthySanity},
		{SerialNumber: "venue-device", Connected: false, Sanity: &lowSanity},
	}})

	result, err := service.GetMap(context.Background(), "token", "req", "corr")
	if err != nil {
		t.Fatalf("GetMap returned error: %v", err)
	}
	if len(result.Nodes) != 1 || len(result.Nodes[0].Devices) != 1 || len(result.Nodes[0].Children) != 1 {
		t.Fatalf("unexpected map hierarchy: %+v", result)
	}
	if got := result.Nodes[0].Devices[0]; got.SerialNumber != "property-device" || got.Name != "Property AP" || got.Availability != "online" || got.Sanity == nil || *got.Sanity != healthySanity {
		t.Fatalf("unexpected Property Device projection: %+v", got)
	}
	if got := result.Nodes[0].Children[0].Devices[0]; got.SerialNumber != "venue-device" || got.Name != "Venue AP" || got.Availability != "offline" || got.Sanity == nil || *got.Sanity != lowSanity {
		t.Fatalf("unexpected Venue Device projection: %+v", got)
	}
}

func TestScopedDevicesFiltersAttentionReasonsBeforePagination(t *testing.T) {
	lowSanity := 45.0
	healthySanity := 96.0
	service := NewPropertyService(&fakePropertyProv{
		tree: &models.OWProvTreeNode{Type: "entity", Name: "Property A", UUID: "property-a"},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {
				{SerialNumber: "low", Name: "Low Sanity AP"},
				{SerialNumber: "missing", Name: "Missing Sanity AP"},
				{SerialNumber: "offline", Name: "Offline AP"},
				{SerialNumber: "healthy", Name: "Healthy AP"},
			},
		},
	}, &fakePropertyGW{statuses: []models.GatewayDeviceStatus{
		{SerialNumber: "low", Connected: true, Sanity: &lowSanity},
		{SerialNumber: "missing", Connected: true},
		{SerialNumber: "offline", Connected: false},
		{SerialNumber: "healthy", Connected: true, Sanity: &healthySanity},
	}})

	low, err := service.GetDevices(context.Background(), "property", "property-a", "token", "req", "corr", models.ScopedDevicesQuery{PageSize: 25, Attention: "low-sanity"})
	if err != nil || low.TotalCount != 1 || low.Devices[0].SerialNumber != "low" {
		t.Fatalf("expected only the online low-sanity Device, result=%+v err=%v", low, err)
	}
	missing, err := service.GetDevices(context.Background(), "property", "property-a", "token", "req", "corr", models.ScopedDevicesQuery{PageSize: 25, Attention: "sanity-unavailable"})
	if err != nil || missing.TotalCount != 1 || missing.Devices[0].SerialNumber != "missing" {
		t.Fatalf("expected only the online Device with unavailable sanity, result=%+v err=%v", missing, err)
	}
}

func TestPropertyOverviewReportsFailedStatusAsUnknown(t *testing.T) {
	service := NewPropertyService(&fakePropertyProv{
		tree: &models.OWProvTreeNode{Type: "entity", Name: "Property A", UUID: "property-a"},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {{SerialNumber: "aa"}, {SerialNumber: "bb"}},
		},
	}, &fakePropertyGW{err: errors.New("owgw unavailable")})

	overview, err := service.GetOverview(context.Background(), "property", "property-a", "token", "req", "corr")
	if err != nil {
		t.Fatalf("GetOverview returned error: %v", err)
	}
	if overview.Summary.OfflineDevices != 0 || overview.Summary.UnknownDevices != 2 {
		t.Fatalf("failed status fetch must be unknown, not offline: %+v", overview.Summary)
	}
	if !overview.DataQuality.Partial || overview.DataQuality.State != "partial" {
		t.Fatalf("expected partial data quality, got %+v", overview.DataQuality)
	}
}

func TestScopedDevicesIncludesDescendantsAndFiltersBeforePagination(t *testing.T) {
	prov := &fakePropertyProv{
		tree: &models.OWProvTreeNode{
			Type: "entity", Name: "Property A", UUID: "property-a",
			Venues: []models.OWProvTreeNode{{Type: "venue", Name: "Venue A", UUID: "venue-a"}},
		},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {{SerialNumber: "aa", Name: "Lobby AP", Entity: "property-a"}},
			"venue:venue-a":     {{SerialNumber: "bb", Name: "Suite AP", Venue: "venue-a"}},
		},
	}
	gw := &fakePropertyGW{statuses: []models.GatewayDeviceStatus{
		{SerialNumber: "aa", Connected: true},
		{SerialNumber: "bb", Connected: false},
	}}
	service := NewPropertyService(prov, gw)

	result, err := service.GetDevices(context.Background(), "property", "property-a", "token", "req", "corr", models.ScopedDevicesQuery{
		PageIndex: 0, PageSize: 1, Availability: "online",
	})
	if err != nil {
		t.Fatalf("GetDevices returned error: %v", err)
	}
	if result.TotalCount != 1 || len(result.Devices) != 1 || result.Devices[0].SerialNumber != "aa" {
		t.Fatalf("expected filtered online result before pagination, got %+v", result)
	}
	if result.Summary.TotalDevices != 2 {
		t.Fatalf("summary must represent complete unfiltered scope, got %+v", result.Summary)
	}
	if result.Summary.OnlineDevices != 1 || result.Summary.OfflineDevices != 1 || result.Summary.UnknownDevices != 0 {
		t.Fatalf("summary must contain factual availability counts: %+v", result.Summary)
	}
}

func TestPropertyDeviceItemReportsOfflineAvailability(t *testing.T) {
	item := propertyDeviceItem(models.InventoryTag{SerialNumber: "aa"}, map[string]models.GatewayDeviceStatus{
		"aa": {SerialNumber: "aa", Connected: false},
	})
	if item.Availability != "offline" || item.Connected || !item.StatusKnown {
		t.Fatalf("offline Device must expose only its factual availability, got %+v", item)
	}

	summary := summarize(map[string]string{"aa": "aa"}, map[string]models.GatewayDeviceStatus{
		"aa": {SerialNumber: "aa", Connected: false},
	})
	if summary.TotalDevices != 1 || summary.OfflineDevices != 1 || summary.OnlineDevices != 0 {
		t.Fatalf("scope must aggregate availability without inventing health, got %+v", summary)
	}
}

func TestScopedDevicesFiltersUnknownBeforePagination(t *testing.T) {
	service := NewPropertyService(&fakePropertyProv{
		tree: &models.OWProvTreeNode{
			Type: "entity", Name: "Property A", UUID: "property-a",
			Venues: []models.OWProvTreeNode{{Type: "venue", Name: "Venue A", UUID: "venue-a"}},
		},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {
				{SerialNumber: "aa", Name: "Alpha", Entity: "property-a"},
				{SerialNumber: "cc", Name: "Charlie", Entity: "property-a"},
			},
			"venue:venue-a": {{SerialNumber: "bb", Name: "Bravo", Venue: "venue-a"}},
		},
	}, &fakePropertyGW{statuses: []models.GatewayDeviceStatus{
		{SerialNumber: "aa", Connected: true},
		{SerialNumber: "bb", Connected: false},
	}})

	result, err := service.GetDevices(context.Background(), "property", "property-a", "token", "req", "corr", models.ScopedDevicesQuery{
		PageIndex: 0, PageSize: 1, Availability: "unknown",
	})
	if err != nil {
		t.Fatalf("GetDevices returned error: %v", err)
	}
	if result.TotalCount != 1 || result.TotalPages != 1 || len(result.Devices) != 1 {
		t.Fatalf("expected one Unknown Device after complete-scope filtering, got %+v", result)
	}
	if result.Devices[0].SerialNumber != "cc" || result.Devices[0].Availability != "unknown" || result.Devices[0].StatusKnown {
		t.Fatalf("missing OWGW status must remain Unknown: %+v", result.Devices[0])
	}
}

func TestScopedDevicesRejectsIncompleteInventory(t *testing.T) {
	service := NewPropertyService(&fakePropertyProv{
		tree: &models.OWProvTreeNode{
			Type: "entity", Name: "Property A", UUID: "property-a",
			Venues: []models.OWProvTreeNode{{Type: "venue", Name: "Venue A", UUID: "venue-a"}},
		},
		inventory: map[string][]models.InventoryTag{
			"entity:property-a": {{SerialNumber: "aa", Name: "Alpha", Entity: "property-a"}},
		},
		inventoryErr: map[string]error{"venue:venue-a": errors.New("inventory page unavailable")},
	}, &fakePropertyGW{})

	result, err := service.GetDevices(context.Background(), "property", "property-a", "token", "req", "corr", models.ScopedDevicesQuery{
		PageIndex: 0, PageSize: 25,
	})
	if err == nil || result != nil {
		t.Fatalf("incomplete inventory must fail closed, result=%+v err=%v", result, err)
	}
}
