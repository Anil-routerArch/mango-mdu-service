package owprov_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

	client := owprov.NewClient(testOWProvConfig(ts.URL))
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

func TestOWProvClient_DeleteInventoryItem_Success(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNoContent, http.StatusNotFound} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			if r.URL.Path != "/api/v1/inventory/60cf84f22290" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(status)
		}))

		client := owprov.NewClient(testOWProvConfig(ts.URL))
		err := client.DeleteInventoryItem(context.Background(), "60:cf:84:f2:22:90", "token", "req-1", "corr-1")
		ts.Close()
		if err != nil {
			t.Errorf("expected nil error for status %d, got %v", status, err)
		}
	}
}

func TestOWProvClient_DeleteInventoryItem_ExtractsRealReason(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"ErrorCode": 403, "ErrorDescription": "Access denied: insufficient permission to delete inventory item"}`))
	}))
	defer ts.Close()

	client := owprov.NewClient(testOWProvConfig(ts.URL))
	err := client.DeleteInventoryItem(context.Background(), "60cf84f22290", "token", "req-1", "corr-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(models.ApiError)
	if !ok {
		t.Fatalf("expected ApiError, got %T: %v", err, err)
	}
	if apiErr.ErrorCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", apiErr.ErrorCode)
	}
	if apiErr.ErrorDescription != "Access denied: insufficient permission to delete inventory item" {
		t.Errorf("expected exact downstream reason, got %q", apiErr.ErrorDescription)
	}
	if strings.Contains(apiErr.Error(), "downstream OWPROV") {
		t.Errorf("error contains unwanted downstream wrapper: %s", apiErr.Error())
	}
}

func TestOWProvClient_AuthorizeDeviceCommand_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/authorization/deviceCommand" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("X-INTERNAL-NAME") != "mango-mdu-service" {
			t.Errorf("missing or incorrect X-INTERNAL-NAME: %s", r.Header.Get("X-INTERNAL-NAME"))
		}
		if r.Header.Get("X-API-KEY") != "secret-prov-key" {
			t.Errorf("missing or incorrect X-API-KEY: %s", r.Header.Get("X-API-KEY"))
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt-token" {
			t.Errorf("missing or incorrect Authorization header: %s", r.Header.Get("Authorization"))
		}

		var req models.DeviceCommandAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if req.SerialNumber != "60cf84f22290" || req.Action != "reboot" {
			t.Errorf("unexpected payload: %+v", req)
		}

		resp := models.DeviceCommandAuthResult{
			Allowed:      true,
			Resource:     "deviceCommand",
			Action:       "REBOOT",
			SerialNumber: "60cf84f22290",
			UserID:       "11111111-0000-0000-6666-999999999999",
			Reason:       "ALLOWED",
			Details:      "User is authorized for action 'REBOOT' on device '60cf84f22290' via policy 'Admin'.",
			PolicyID:     "pol-admin",
			PolicyName:   "Admin",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{
		InstanceResolver: func() (string, string, error) {
			return ts.URL, "secret-prov-key", nil
		},
		InternalName: "mango-mdu-service",
	})

	res, err := client.AuthorizeDeviceCommand(context.Background(), "60:cf:84:f2:22:90", "reboot", "test-jwt-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed {
		t.Errorf("expected allowed=true, got false")
	}
	if res.Action != "REBOOT" {
		t.Errorf("expected REBOOT, got %s", res.Action)
	}
	if res.Reason != "ALLOWED" {
		t.Errorf("expected ALLOWED, got %s", res.Reason)
	}
}

func TestOWProvClient_ApplyDeviceConfiguration(t *testing.T) {
	t.Run("forwards delegated identity and returns OWPROV response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/inventory/60cf84f22290" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			if r.URL.Query().Get("applyConfiguration") != "true" {
				t.Errorf("missing applyConfiguration=true query: %s", r.URL.RawQuery)
			}
			for header, expected := range map[string]string{
				"X-INTERNAL-NAME":  "mango-mdu-service",
				"X-API-KEY":        "secret-prov-key",
				"Authorization":    "Bearer user-token",
				"X-Request-Id":     "req-1",
				"X-Correlation-Id": "corr-1",
			} {
				if actual := r.Header.Get(header); actual != expected {
					t.Errorf("expected %s=%q, got %q", header, expected, actual)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"errorCode":0,"appliedConfiguration":true}`))
		}))
		defer ts.Close()

		client := owprov.NewClient(owprov.Config{
			InstanceResolver: func() (string, string, error) { return ts.URL, "secret-prov-key", nil },
			InternalName:     "mango-mdu-service",
		})
		result, err := client.ApplyDeviceConfiguration(context.Background(), "60:cf:84:f2:22:90", "Bearer user-token", "req-1", "corr-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result) != `{"errorCode":0,"appliedConfiguration":true}` {
			t.Fatalf("unexpected response: %s", result)
		}
	})

	t.Run("propagates downstream error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ErrorCode":403,"ErrorDescription":"inventory access denied"}`))
		}))
		defer ts.Close()

		client := owprov.NewClient(testOWProvConfig(ts.URL))
		_, err := client.ApplyDeviceConfiguration(context.Background(), "60cf84f22290", "token", "req", "corr")
		var apiErr models.ApiError
		if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusForbidden {
			t.Fatalf("expected downstream 403 ApiError, got %T %v", err, err)
		}
	})

	t.Run("rejects application failure returned with HTTP 200", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"errorCode":1,"errors":["configuration application failed"]}`))
		}))
		defer ts.Close()

		client := owprov.NewClient(testOWProvConfig(ts.URL))
		_, err := client.ApplyDeviceConfiguration(context.Background(), "60cf84f22290", "token", "req", "corr")
		var apiErr models.ApiError
		if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadGateway {
			t.Fatalf("expected application failure as 502 ApiError, got %T %v", err, err)
		}
		if apiErr.ErrorDetails != "configuration application failed" {
			t.Fatalf("expected OWPROV failure details, got %q", apiErr.ErrorDetails)
		}
	})

	t.Run("rejects malformed HTTP 200 response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"errorCode":`))
		}))
		defer ts.Close()

		client := owprov.NewClient(testOWProvConfig(ts.URL))
		_, err := client.ApplyDeviceConfiguration(context.Background(), "60cf84f22290", "token", "req", "corr")
		var apiErr models.ApiError
		if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadGateway {
			t.Fatalf("expected malformed response as 502 ApiError, got %T %v", err, err)
		}
	})
}

func TestOWProvClient_AuthorizeDeviceCommand_Denied(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := models.DeviceCommandAuthResult{
			Allowed:      false,
			Resource:     "deviceCommand",
			Action:       "REBOOT",
			SerialNumber: "60cf84f22290",
			UserID:       "user-1",
			Reason:       "ACTION_NOT_GRANTED",
			Details:      "Policy 'Operator' does not grant action 'REBOOT' on deviceCommand.",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owprov.NewClient(testOWProvConfig(ts.URL))

	res, err := client.AuthorizeDeviceCommand(context.Background(), "60cf84f22290", "reboot", "token", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Allowed {
		t.Errorf("expected allowed=false, got true")
	}
	if res.Reason != "ACTION_NOT_GRANTED" {
		t.Errorf("expected ACTION_NOT_GRANTED, got %s", res.Reason)
	}
}

func TestOWProvClient_InternalHeadersAndBearerPropagation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-INTERNAL-NAME") != "mango-mdu-service" {
			t.Errorf("%s %s: expected X-INTERNAL-NAME mango-mdu-service, got %q", r.Method, r.URL.Path, r.Header.Get("X-INTERNAL-NAME"))
		}
		if r.Header.Get("X-API-KEY") != "secret-prov-key" {
			t.Errorf("%s %s: expected X-API-KEY secret-prov-key, got %q", r.Method, r.URL.Path, r.Header.Get("X-API-KEY"))
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt-token" {
			t.Errorf("%s %s: expected Authorization Bearer test-jwt-token, got %q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/managementPolicy/pol-1":
			json.NewEncoder(w).Encode(models.ManagementPolicy{ID: "pol-1", Name: "Ops"})
		case r.URL.Path == "/api/v1/managementRole":
			json.NewEncoder(w).Encode(models.ManagementRoleListResponse{})
		case r.URL.Path == "/api/v1/entity":
			json.NewEncoder(w).Encode(models.EntityListResponse{})
		case r.URL.Path == "/api/v1/venue":
			json.NewEncoder(w).Encode(models.VenueListResponse{})
		case r.URL.Path == "/api/v1/inventory" && r.URL.Query().Get("countOnly") == "true":
			json.NewEncoder(w).Encode(models.CountOnlyResponse{Count: 1})
		case r.URL.Path == "/api/v1/inventory" && r.URL.Query().Get("serialOnly") == "true":
			json.NewEncoder(w).Encode(map[string][]string{"serialNumbers": {"60cf84f22290"}})
		case r.URL.Path == "/api/v1/inventory":
			json.NewEncoder(w).Encode(models.InventoryTagListResponse{})
		case r.URL.Path == "/api/v1/inventory/60cf84f22290" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(models.InventoryTag{SerialNumber: "60cf84f22290"})
		case r.URL.Path == "/api/v1/inventory/60cf84f22290" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/v1/authorization/deviceCommand":
			json.NewEncoder(w).Encode(models.DeviceCommandAuthResult{Allowed: true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	client := owprov.NewClient(owprov.Config{
		InstanceResolver: func() (string, string, error) {
			return ts.URL, "secret-prov-key", nil
		},
		InternalName: "mango-mdu-service",
	})
	ctx := context.Background()

	if _, err := client.GetPolicy(ctx, "pol-1", "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	if _, err := client.GetRolesByPolicy(ctx, "pol-1", "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("GetRolesByPolicy failed: %v", err)
	}
	if _, err := client.GetEntities(ctx, "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("GetEntities failed: %v", err)
	}
	if _, err := client.GetVenues(ctx, "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("GetVenues failed: %v", err)
	}
	if _, err := client.GetInventory(ctx, "test-jwt-token", "req-1", "corr-1", 10, 0, false); err != nil {
		t.Fatalf("GetInventory failed: %v", err)
	}
	if _, err := client.GetInventoryCount(ctx, "test-jwt-token", "req-1", "corr-1", false); err != nil {
		t.Fatalf("GetInventoryCount failed: %v", err)
	}
	if _, err := client.GetInventorySerials(ctx, "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("GetInventorySerials failed: %v", err)
	}
	if _, err := client.GetInventoryItem(ctx, "60cf84f22290", "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("GetInventoryItem failed: %v", err)
	}
	if err := client.DeleteInventoryItem(ctx, "60cf84f22290", "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("DeleteInventoryItem failed: %v", err)
	}
	if _, err := client.AuthorizeDeviceCommand(ctx, "60cf84f22290", "reboot", "test-jwt-token", "req-1", "corr-1"); err != nil {
		t.Fatalf("AuthorizeDeviceCommand failed: %v", err)
	}
}

func testOWProvConfig(endpoint string) owprov.Config {
	return owprov.Config{
		InstanceResolver: func() (string, string, error) { return endpoint, "test-owprov-key", nil },
		InternalName:     "mdu-test",
	}
}

func owprovClientOperations(client owprov.Client) []struct {
	name string
	call func() error
} {
	ctx := context.Background()
	return []struct {
		name string
		call func() error
	}{
		{"GetPolicy", func() error { _, err := client.GetPolicy(ctx, "pol-1", "valid-user-token", "req", "corr"); return err }},
		{"GetRolesByPolicy", func() error {
			_, err := client.GetRolesByPolicy(ctx, "pol-1", "valid-user-token", "req", "corr")
			return err
		}},
		{"GetEntities", func() error { _, err := client.GetEntities(ctx, "valid-user-token", "req", "corr"); return err }},
		{"GetVenues", func() error { _, err := client.GetVenues(ctx, "valid-user-token", "req", "corr"); return err }},
		{"GetInventory", func() error {
			_, err := client.GetInventory(ctx, "valid-user-token", "req", "corr", 10, 0, false)
			return err
		}},
		{"GetInventoryCount", func() error {
			_, err := client.GetInventoryCount(ctx, "valid-user-token", "req", "corr", false)
			return err
		}},
		{"GetInventorySerials", func() error { _, err := client.GetInventorySerials(ctx, "valid-user-token", "req", "corr"); return err }},
		{"GetInventoryItem", func() error {
			_, err := client.GetInventoryItem(ctx, "60cf84f22290", "valid-user-token", "req", "corr")
			return err
		}},
		{"DeleteInventoryItem", func() error {
			return client.DeleteInventoryItem(ctx, "60cf84f22290", "valid-user-token", "req", "corr")
		}},
		{"AuthorizeDeviceCommand", func() error {
			_, err := client.AuthorizeDeviceCommand(ctx, "60cf84f22290", "reboot", "valid-user-token", "req", "corr")
			return err
		}},
	}
}

func TestOWProvClient_ResolutionFailureStopsRequests(t *testing.T) {
	var requests atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()
	for _, tc := range []struct {
		name       string
		endpoint   string
		key        string
		err        error
		noResolver bool
		missingKey bool
	}{
		{name: "missing resolver", noResolver: true},
		{name: "resolver error with endpoint and key", endpoint: ts.URL, key: "must-not-be-used", err: errors.New("internal resolver failure")},
		{name: "empty endpoint", key: "must-not-be-used"},
		{name: "whitespace endpoint", endpoint: " \t\n", key: "must-not-be-used"},
		{name: "empty key", endpoint: ts.URL, missingKey: true},
		{name: "whitespace key", endpoint: ts.URL, key: " \t\n", missingKey: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			resolverCalls := 0
			cfg := owprov.Config{
				InstanceResolver: func() (string, string, error) {
					resolverCalls++
					// A second lookup could observe newly available discovery data.
					// A failure must stop this request, rather than retry with a different target/key.
					if resolverCalls%2 == 0 {
						return ts.URL, "must-not-be-used", nil
					}
					return tc.endpoint, tc.key, tc.err
				},
				InternalName: "mango-mdu-service",
				Logger:       slog.New(slog.NewTextHandler(&logs, nil)),
			}
			if tc.noResolver {
				cfg.InstanceResolver = nil
			}
			client := owprov.NewClient(cfg)
			for _, operation := range owprovClientOperations(client) {
				t.Run(operation.name, func(t *testing.T) {
					resolverCalls = 0
					err := operation.call()
					var apiErr models.ApiError
					if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusServiceUnavailable {
						t.Fatalf("expected 503 without fallback, got %v", err)
					}
					if tc.missingKey && apiErr.ErrorDetails != "owprov service credentials unavailable" {
						t.Fatalf("expected missing credential error, got %q", apiErr.ErrorDetails)
					}
					wantCalls := 1
					if tc.noResolver {
						wantCalls = 0
					}
					if resolverCalls != wantCalls || requests.Load() != 0 {
						t.Fatalf("failed resolution must not retry or send HTTP requests: resolver calls=%d requests=%d", resolverCalls, requests.Load())
					}
				})
			}
			if tc.missingKey && !strings.Contains(logs.String(), "OWPROV service key missing") {
				t.Fatal("missing service credentials were not logged")
			}
			if strings.Contains(logs.String(), "must-not-be-used") || strings.Contains(logs.String(), "valid-user-token") {
				t.Fatal("credentials leaked into logs")
			}
		})
	}
}

func TestOWProvClient_UsesEndpointAndKeyFromSameInstance(t *testing.T) {
	var requests atomic.Int32
	newServer := func(expectedKey string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			if r.Header.Get("X-API-KEY") != expectedKey || r.Header.Get("X-INTERNAL-NAME") != "mango-mdu-service" || r.Header.Get("Authorization") != "Bearer user-token" {
				t.Error("expected matching target credentials and delegated bearer token")
			}
			json.NewEncoder(w).Encode(models.DeviceCommandAuthResult{Allowed: true})
		}))
	}
	first := newServer("first-owprov-key")
	defer first.Close()
	second := newServer("second-owprov-key")
	defer second.Close()
	resolverCalls := 0
	client := owprov.NewClient(owprov.Config{
		InstanceResolver: func() (string, string, error) {
			resolverCalls++
			if resolverCalls == 1 {
				return first.URL, "first-owprov-key", nil
			}
			return second.URL, "second-owprov-key", nil
		},
		InternalName: "mango-mdu-service",
	})
	for i := 1; i <= 2; i++ {
		_, err := client.AuthorizeDeviceCommand(context.Background(), "60cf84f22290", "reboot", "user-token", "req", "corr")
		if err != nil || resolverCalls != i || requests.Load() != int32(i) {
			t.Fatalf("expected one paired resolution per request, got resolver calls=%d requests=%d err=%v", resolverCalls, requests.Load(), err)
		}
	}
}

func TestOWProvClient_DiscoveryRecoveryUsesNextRequest(t *testing.T) {
	var requests atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("X-API-KEY") != "recovered-owprov-key" {
			t.Error("wrong target service key")
		}
		json.NewEncoder(w).Encode(models.DeviceCommandAuthResult{Allowed: true})
	}))
	defer ts.Close()
	resolverCalls := 0
	client := owprov.NewClient(owprov.Config{
		InstanceResolver: func() (string, string, error) {
			resolverCalls++
			if resolverCalls == 1 {
				return "", "", errors.New("discovery unavailable")
			}
			return ts.URL, "recovered-owprov-key", nil
		},
		InternalName: "mango-mdu-service",
	})
	_, err := client.AuthorizeDeviceCommand(context.Background(), "60cf84f22290", "reboot", "user-token", "req", "corr")
	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusServiceUnavailable || resolverCalls != 1 || requests.Load() != 0 {
		t.Fatalf("expected first request to stop on discovery failure, got resolver calls=%d requests=%d err=%v", resolverCalls, requests.Load(), err)
	}
	_, err = client.AuthorizeDeviceCommand(context.Background(), "60cf84f22290", "reboot", "user-token", "req", "corr")
	if err != nil || resolverCalls != 2 || requests.Load() != 1 {
		t.Fatalf("expected next request to use recovered discovery, got resolver calls=%d requests=%d err=%v", resolverCalls, requests.Load(), err)
	}
}
