package owprov_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	client := owprov.NewClient(owprov.Config{BaseURL: ts.URL})
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

	client := owprov.NewClient(owprov.Config{BaseURL: ts.URL})
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

	client := owprov.NewClient(owprov.Config{BaseURL: ts.URL})
	roles, err := client.GetRolesByPolicy(context.Background(), "pol-1", "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 || roles[0].ID != "role-1" {
		t.Errorf("unexpected roles: %+v", roles)
	}
}

func TestOWProvClient_GetEntities(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.EntityListResponse{
			Entities: []models.Entity{
				{ID: "ent-1", Name: "Sunrise"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{BaseURL: ts.URL})
	entities, err := client.GetEntities(context.Background(), "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 1 || entities[0].Name != "Sunrise" {
		t.Errorf("unexpected entities: %+v", entities)
	}
}

func TestOWProvClient_GetVenues(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.VenueListResponse{
			Venues: []models.Venue{
				{ID: "ven-1", Name: "Tower A"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{BaseURL: ts.URL})
	venues, err := client.GetVenues(context.Background(), "test-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(venues) != 1 || venues[0].Name != "Tower A" {
		t.Errorf("unexpected venues: %+v", venues)
	}
}
