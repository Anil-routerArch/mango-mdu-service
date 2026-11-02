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

