package owgw_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owgw"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

func TestOWGWClient_GetDevicesStatus_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/devices" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-jwt" {
			t.Errorf("expected Bearer test-jwt, got %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("deviceWithStatus") != "true" {
			t.Errorf("expected deviceWithStatus=true, got %q", r.URL.Query().Get("deviceWithStatus"))
		}
		if r.URL.Query().Get("select") != "60cf84f22290,dc62796520cd" {
			t.Errorf("unexpected select query: %s", r.URL.Query().Get("select"))
		}

		sanityVal := 100.0
		memVal := 45.5
		resp := models.GatewayDevicesWithStatusResponse{
			DevicesWithStatus: []models.GatewayDeviceStatus{
				{
					SerialNumber: "60cf84f22290",
					Connected:    true,
					Sanity:       &sanityVal,
					IPAddress:    "192.168.1.100",
					MemoryUsed:   &memVal,
					Firmware:     "TIP-v4.2.4",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owgw.NewClient(owgw.Config{URLResolver: func() string { return ts.URL }})
	statuses, err := client.GetDevicesStatus(context.Background(), []string{"60cf84f22290", "dc62796520cd"}, "test-jwt", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].SerialNumber != "60cf84f22290" {
		t.Errorf("expected serial 60cf84f22290, got %s", statuses[0].SerialNumber)
	}
	if !statuses[0].Connected {
		t.Errorf("expected connected=true, got false")
	}
}

func TestOWGWClient_GetDevicesStatus_EmptySerials(t *testing.T) {
	client := owgw.NewClient(owgw.Config{URLResolver: func() string { return "http://unused" }})
	statuses, err := client.GetDevicesStatus(context.Background(), []string{}, "token", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("expected 0 statuses, got %d", len(statuses))
	}
}

func TestOWGWClient_GetDevicesStatus_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	client := owgw.NewClient(owgw.Config{URLResolver: func() string { return ts.URL }})
	_, err := client.GetDevicesStatus(context.Background(), []string{"dev-1"}, "expired-token", "req", "corr")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(models.ApiError)
	if !ok || apiErr.ErrorCode != http.StatusUnauthorized {
		t.Errorf("expected 401 ApiError, got %+v", err)
	}
}

func TestOWGWClient_DeleteDevice_ExtractsRealReason(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/device/60cf84f22290" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"ErrorCode": 403, "ErrorDescription": "Device is managed by a different entity"}`))
	}))
	defer ts.Close()

	client := owgw.NewClient(owgw.Config{URLResolver: func() string { return ts.URL }})
	err := client.DeleteDevice(context.Background(), "60cf84f22290", "token", "req", "corr")
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
	if apiErr.ErrorDescription != "Device is managed by a different entity" {
		t.Errorf("expected exact error description, got %q", apiErr.ErrorDescription)
	}
}

func TestOWGWClient_GetCommand_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/command/cmd-uuid-123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"UUID":"cmd-uuid-123","serialNumber":"60cf84f22290","status":"completed"}`))
	}))
	defer ts.Close()

	client := owgw.NewClient(owgw.Config{URLResolver: func() string { return ts.URL }})
	raw, err := client.GetCommand(context.Background(), "cmd-uuid-123", "token", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if parsed["serialNumber"] != "60cf84f22290" {
		t.Errorf("expected serial 60cf84f22290, got %v", parsed["serialNumber"])
	}
}
