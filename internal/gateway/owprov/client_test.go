package owprov_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owprov"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

func TestOWProvClient_GetPolicy_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/managementPolicy/pol-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or invalid auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "mango-mdu-service/1.0" {
			t.Errorf("missing user-agent: %s", r.Header.Get("User-Agent"))
		}

		resp := models.ManagementPolicy{
			ID:   "pol-1",
			Name: "Operator",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	policy, err := client.GetPolicy(context.Background(), "pol-1", "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.Name != "Operator" {
		t.Errorf("expected 'Operator', got %q", policy.Name)
	}
}

func TestOWProvClient_GetPolicy_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	_, err := client.GetPolicy(context.Background(), "not-found", "test-token", "req-1", "corr-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(models.ApiError)
	if !ok || apiErr.ErrorCode != http.StatusNotFound {
		t.Errorf("expected 404 ApiError, got %+v", err)
	}
}

func TestOWProvClient_GetRolesByPolicy(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/managementRole" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("policyId") != "pol-1" {
			t.Errorf("missing policyId query: %s", r.URL.Query().Get("policyId"))
		}

		resp := models.ManagementRoleListResponse{
			Roles: []models.ManagementRole{
				{ID: "role-1", ManagementPolicy: "pol-1"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	roles, err := client.GetRolesByPolicy(context.Background(), "pol-1", "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 || roles[0].ID != "role-1" {
		t.Errorf("unexpected roles: %+v", roles)
	}
}

func TestOWProvClient_GetEntities(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Query().Get("limit") != "500" {
			t.Errorf("expected limit=500, got %s", r.URL.Query().Get("limit"))
		}
		offset := r.URL.Query().Get("offset")
		if offset == "0" {
			resp := models.EntityListResponse{
				Entities: []models.Entity{
					{ID: "ent-1", Name: "Sunrise"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			t.Errorf("unexpected offset: %s", offset)
		}
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	entities, err := client.GetEntities(context.Background(), "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 1 || entities[0].Name != "Sunrise" {
		t.Errorf("unexpected entities: %+v", entities)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestOWProvClient_GetEntities_Pagination(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")
		if offset == "0" {
			entities := make([]models.Entity, 500)
			for i := range entities {
				entities[i] = models.Entity{ID: "ent-" + string(rune('a'+i%26)), Name: "Prop"}
			}
			json.NewEncoder(w).Encode(models.EntityListResponse{Entities: entities})
		} else if offset == "500" {
			json.NewEncoder(w).Encode(models.EntityListResponse{
				Entities: []models.Entity{{ID: "ent-last", Name: "Last Prop"}},
			})
		} else {
			t.Errorf("unexpected offset: %s", offset)
		}
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	entities, err := client.GetEntities(context.Background(), "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 501 {
		t.Errorf("expected 501 entities, got %d", len(entities))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestOWProvClient_GetVenues(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Query().Get("limit") != "500" {
			t.Errorf("expected limit=500, got %s", r.URL.Query().Get("limit"))
		}
		offset := r.URL.Query().Get("offset")
		if offset == "0" {
			resp := models.VenueListResponse{
				Venues: []models.Venue{
					{ID: "ven-1", Name: "Tower A"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else {
			t.Errorf("unexpected offset: %s", offset)
		}
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	venues, err := client.GetVenues(context.Background(), "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(venues) != 1 || venues[0].Name != "Tower A" {
		t.Errorf("unexpected venues: %+v", venues)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestOWProvClient_GetVenues_Pagination(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		offset := r.URL.Query().Get("offset")
		if offset == "0" {
			venues := make([]models.Venue, 500)
			for i := range venues {
				venues[i] = models.Venue{ID: "ven-" + string(rune('a'+i%26)), Name: "Venue"}
			}
			json.NewEncoder(w).Encode(models.VenueListResponse{Venues: venues})
		} else if offset == "500" {
			json.NewEncoder(w).Encode(models.VenueListResponse{
				Venues: []models.Venue{{ID: "ven-last", Name: "Last Venue"}},
			})
		} else {
			t.Errorf("unexpected offset: %s", offset)
		}
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	venues, err := client.GetVenues(context.Background(), "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(venues) != 501 {
		t.Errorf("expected 501 venues, got %d", len(venues))
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestOWProvClient_DownstreamError_Sanitized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"internal_error":"FATAL: connection to db-internal-01.local:5432 failed"}`))
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	_, err := client.GetPolicy(context.Background(), "pol-1", "token", "req-1", "corr-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(models.ApiError)
	if !ok {
		t.Fatalf("expected ApiError, got %T: %v", err, err)
	}
	if apiErr.ErrorCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", apiErr.ErrorCode)
	}
	if apiErr.ErrorDescription != "Downstream Error" {
		t.Errorf("expected 'Downstream Error', got %q", apiErr.ErrorDescription)
	}
	// Verify raw body is NOT leaked
	if strings.Contains(apiErr.ErrorDetails, "db-internal-01.local") {
		t.Errorf("sensitive body leaked in ErrorDetails: %s", apiErr.ErrorDetails)
	}
	if apiErr.ErrorDetails != "downstream OWPROV returned status 500" {
		t.Errorf("expected sanitized message, got %q", apiErr.ErrorDetails)
	}
}

func TestOWProvClient_GetInventory_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/inventory" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt-token" {
			t.Errorf("expected Bearer test-jwt-token, got %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("withExtendedInfo") != "true" {
			t.Errorf("expected withExtendedInfo=true, got %q", r.URL.Query().Get("withExtendedInfo"))
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %q", r.URL.Query().Get("limit"))
		}
		if r.URL.Query().Get("offset") != "0" {
			t.Errorf("expected offset=0, got %q", r.URL.Query().Get("offset"))
		}

		resp := models.InventoryTagListResponse{
			TagList: []models.InventoryTag{
				{
					SerialNumber: "60cf84f22290",
					Name:         "AP-Main-Lobby",
					DeviceType:   "ap",
					Entity:       "ent-1",
					Venue:        "ven-1",
					ExtendedInfo: &models.InventoryExtendedInfo{
						Venue: &models.ExtendedVenue{
							ID:   "ven-1",
							Name: "Main Building",
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	tags, err := client.GetInventory(context.Background(), "test-jwt-token", "req-1", "corr-1", 10, 0, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
	if tags[0].SerialNumber != "60cf84f22290" {
		t.Errorf("expected serial 60cf84f22290, got %s", tags[0].SerialNumber)
	}
	if tags[0].ExtendedInfo == nil || tags[0].ExtendedInfo.Venue.Name != "Main Building" {
		t.Errorf("expected venue name Main Building, got %+v", tags[0].ExtendedInfo)
	}
}

func TestOWProvClient_GetInventoryCount_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/inventory" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt-token" {
			t.Errorf("expected Bearer test-jwt-token, got %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("countOnly") != "true" {
			t.Errorf("expected countOnly=true, got %q", r.URL.Query().Get("countOnly"))
		}

		resp := models.CountOnlyResponse{
			Count: 42,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	count, err := client.GetInventoryCount(context.Background(), "test-jwt-token", "req-1", "corr-1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 42 {
		t.Errorf("expected count 42, got %d", count)
	}
}

func TestOWProvClient_GetInventorySerials_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/inventory" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt-token" {
			t.Errorf("expected Bearer test-jwt-token, got %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("serialOnly") != "true" {
			t.Errorf("expected serialOnly=true, got %q", r.URL.Query().Get("serialOnly"))
		}

		resp := models.InventorySerialsResponse{
			SerialNumbers: []string{"24f5a201b1a0", "24f5a201b1a1", "24f5a201b1a2"},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	serials, err := client.GetInventorySerials(context.Background(), "test-jwt-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(serials) != 3 {
		t.Fatalf("expected 3 serials, got %d", len(serials))
	}
	if serials[0] != "24f5a201b1a0" || serials[1] != "24f5a201b1a1" || serials[2] != "24f5a201b1a2" {
		t.Errorf("unexpected serials: %v", serials)
	}
}

func TestOWProvClient_GetInventorySerials_Pagination_Exhaustive(t *testing.T) {
	pageRequests := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageRequests++
		offset := r.URL.Query().Get("offset")
		limit := r.URL.Query().Get("limit")

		if limit != "500" {
			t.Errorf("expected limit=500, got %s", limit)
		}

		var serials []string
		if offset == "0" {
			// Page 1: full page of 500 serials
			serials = make([]string, 500)
			for i := 0; i < 500; i++ {
				serials[i] = fmt.Sprintf("serial-page1-%03d", i)
			}
		} else if offset == "500" {
			// Page 2: remaining 150 serials (< 500 terminates loop)
			serials = make([]string, 150)
			for i := 0; i < 150; i++ {
				serials[i] = fmt.Sprintf("serial-page2-%03d", i)
			}
		} else {
			t.Errorf("unexpected offset: %s", offset)
		}

		resp := models.InventorySerialsResponse{
			SerialNumbers: serials,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	serials, err := client.GetInventorySerials(context.Background(), "test-jwt-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pageRequests != 2 {
		t.Errorf("expected 2 page requests, got %d", pageRequests)
	}
	if len(serials) != 650 {
		t.Fatalf("expected 650 serials across pages, got %d", len(serials))
	}
	if serials[0] != "serial-page1-000" || serials[499] != "serial-page1-499" || serials[500] != "serial-page2-000" || serials[649] != "serial-page2-149" {
		t.Errorf("serials boundary elements mismatch")
	}
}

func TestOWProvClient_GetInventorySerials_Pagination_FailClosed(t *testing.T) {
	pageRequests := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageRequests++
		offset := r.URL.Query().Get("offset")

		if offset == "0" {
			// Page 1 succeeds with 500 serials
			serials := make([]string, 500)
			for i := 0; i < 500; i++ {
				serials[i] = fmt.Sprintf("serial-%03d", i)
			}
			json.NewEncoder(w).Encode(models.InventorySerialsResponse{SerialNumbers: serials})
		} else {
			// Page 2 fails with 500 internal server error
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal error"}`))
		}
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{URLResolver: func() string { return ts.URL }})
	serials, err := client.GetInventorySerials(context.Background(), "test-jwt-token", "req-1", "corr-1")
	if err == nil {
		t.Fatalf("expected error on downstream failure, got nil error with %d serials", len(serials))
	}
	if serials != nil {
		t.Errorf("expected nil serials on fail-closed error, got %v", serials)
	}
	if pageRequests != 2 {
		t.Errorf("expected 2 page requests before aborting, got %d", pageRequests)
	}
}
