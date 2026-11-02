package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/http/handlers"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

type mockDeviceService struct {
	getDevicesFn           func(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error)
	getDeviceCommandsFn    func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	deleteDeviceCommandFn func(ctx context.Context, serial, commandUUID, token, reqID, corrID string) error
	getDeviceHealthChecksFn func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	getDeviceLogsFn        func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	getDeviceStatisticsFn  func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error)
	getDeviceStatusFn       func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceFn             func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	updateDeviceFn          func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceCapabilitiesFn func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
}

func (m *mockDeviceService) GetDevices(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error) {
	if m.getDevicesFn != nil {
		return m.getDevicesFn(ctx, token, reqID, corrID, pageIndex, pageSize, unassignedOnly)
	}
	return nil, nil
}

func (m *mockDeviceService) GetDeviceCommands(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if m.getDeviceCommandsFn != nil {
		return m.getDeviceCommandsFn(ctx, serial, token, reqID, corrID, queryParams)
	}
	return json.RawMessage(`{"commands":[]}`), nil
}

func (m *mockDeviceService) DeleteDeviceCommand(ctx context.Context, serial, commandUUID, token, reqID, corrID string) error {
	if m.deleteDeviceCommandFn != nil {
		return m.deleteDeviceCommandFn(ctx, serial, commandUUID, token, reqID, corrID)
	}
	return nil
}

func (m *mockDeviceService) GetDeviceHealthChecks(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if m.getDeviceHealthChecksFn != nil {
		return m.getDeviceHealthChecksFn(ctx, serial, token, reqID, corrID, queryParams)
	}
	return json.RawMessage(`{"values":[]}`), nil
}

func (m *mockDeviceService) GetDeviceLogs(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if m.getDeviceLogsFn != nil {
		return m.getDeviceLogsFn(ctx, serial, token, reqID, corrID, queryParams)
	}
	return json.RawMessage(`{"values":[]}`), nil
}

func (m *mockDeviceService) GetDeviceStatistics(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	if m.getDeviceStatisticsFn != nil {
		return m.getDeviceStatisticsFn(ctx, serial, token, reqID, corrID, queryParams)
	}
	return json.RawMessage(`{"data":[]}`), nil
}

func (m *mockDeviceService) GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getDeviceStatusFn != nil {
		return m.getDeviceStatusFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{"status":{}}`), nil
}

func (m *mockDeviceService) GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getDeviceFn != nil {
		return m.getDeviceFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{"serialNumber":"60cf84f22290"}`), nil
}

func (m *mockDeviceService) UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	if m.updateDeviceFn != nil {
		return m.updateDeviceFn(ctx, serial, body, token, reqID, corrID)
	}
	return json.RawMessage(`{"serialNumber":"` + serial + `"}`), nil
}

func (m *mockDeviceService) GetDeviceCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getDeviceCapabilitiesFn != nil {
		return m.getDeviceCapabilitiesFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{"capabilities":{"model":"ASUS RT-AX53U"}}`), nil
}

func (m *mockDeviceService) ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"result":"ok"}`), nil
}

func (m *mockDeviceService) GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"server":"tty.example.com","viewport":5912,"connectionId":"conn-123"}`), nil
}

func (m *mockDeviceService) DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	return nil
}

func setupDeviceTestApp(svc *mockDeviceService) *fiber.App {
	app := fiber.New()
	handler := handlers.NewDeviceHandler(svc)
	app.Get("/api/v1/devices", handler.GetDevices)
	app.Get("/api/v1/devices/:serial/commands", handler.GetDeviceCommands)
	app.Delete("/api/v1/devices/:serial/commands/:uuid", handler.DeleteDeviceCommand)
	app.Get("/api/v1/devices/:serial/healthchecks", handler.GetDeviceHealthChecks)
	app.Get("/api/v1/devices/:serial/logs", handler.GetDeviceLogs)
	app.Get("/api/v1/devices/:serial/statistics", handler.GetDeviceStatistics)
	app.Get("/api/v1/devices/:serial/status", handler.GetDeviceStatus)
	app.Get("/api/v1/devices/:serial/capabilities", handler.GetDeviceCapabilities)
	app.Get("/api/v1/devices/:serial/rtty", handler.GetDeviceRtty)
	app.Get("/api/v1/devices/:serial", handler.GetDevice)
	app.Put("/api/v1/devices/:serial", handler.UpdateDevice)
	app.Delete("/api/v1/devices/:serial", handler.DeleteDevice)
	app.Post("/api/v1/devices/:serial/:command", handler.ExecuteDeviceCommand)
	return app
}

func TestDeviceHandler_GetDevices_MissingAuth(t *testing.T) {
	svc := &mockDeviceService{}
	app := setupDeviceTestApp(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestDeviceHandler_GetDevices_Success(t *testing.T) {
	svc := &mockDeviceService{
		getDevicesFn: func(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error) {
			if token != "Bearer my-jwt" {
				t.Errorf("expected token 'Bearer my-jwt', got %q", token)
			}
			if pageIndex != 1 || pageSize != 20 || !unassignedOnly {
				t.Errorf("unexpected params: pageIndex=%d pageSize=%d unassigned=%v", pageIndex, pageSize, unassignedOnly)
			}
			return &models.PaginatedDevicesResponse{
				Devices: []models.DeviceFleetItem{
					{
						SerialNumber: "60cf84f22290",
						Name:         "AP-Lobby",
						Connected:    true,
					},
				},
				TotalCount: 1,
				PageIndex:  1,
				PageSize:   20,
				TotalPages: 1,
			}, nil
		},
	}

	app := setupDeviceTestApp(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices?pageIndex=1&pageSize=20&unassignedOnly=true", nil)
	req.Header.Set("Authorization", "Bearer my-jwt")
	req.Header.Set("X-Request-Id", "req-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var data models.PaginatedDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(data.Devices) != 1 || data.Devices[0].SerialNumber != "60cf84f22290" {
		t.Errorf("unexpected devices payload: %+v", data)
	}
}

func TestDeviceHandler_GetDevices_ApiError(t *testing.T) {
	svc := &mockDeviceService{
		getDevicesFn: func(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error) {
			return nil, models.NewApiError(http.StatusForbidden, "Forbidden", "Scope permission denied")
		},
	}

	app := setupDeviceTestApp(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer token-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

func TestDeviceHandler_GetDevices_GenericError(t *testing.T) {
	svc := &mockDeviceService{
		getDevicesFn: func(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error) {
			return nil, errors.New("unhandled internal crash")
		},
	}

	app := setupDeviceTestApp(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	req.Header.Set("Authorization", "Bearer token-123")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestDeviceHandler_ProxyEndpoints_MissingAuth(t *testing.T) {
	svc := &mockDeviceService{}
	app := setupDeviceTestApp(svc)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/devices/60cf84f22290/commands"},
		{http.MethodDelete, "/api/v1/devices/60cf84f22290/commands/cmd-123"},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/healthchecks"},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/logs"},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/statistics"},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/status"},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/capabilities"},
		{http.MethodGet, "/api/v1/devices/60cf84f22290"},
		{http.MethodPut, "/api/v1/devices/60cf84f22290"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed test request %s %s: %v", ep.method, ep.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for %s %s, got %d", ep.method, ep.path, resp.StatusCode)
		}
	}
}

func TestDeviceHandler_ProxyEndpoints_Success(t *testing.T) {
	svc := &mockDeviceService{
		getDeviceCommandsFn: func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
			if queryParams.Get("limit") != "25" || queryParams.Get("newest") != "true" {
				t.Errorf("unexpected query params: %v", queryParams)
			}
			return json.RawMessage(`{"commands":[{"command":"configure"}]}`), nil
		},
		deleteDeviceCommandFn: func(ctx context.Context, serial, commandUUID, token, reqID, corrID string) error {
			if commandUUID != "cmd-999" {
				t.Errorf("unexpected command uuid: %s", commandUUID)
			}
			return nil
		},
		getDeviceHealthChecksFn: func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
			return json.RawMessage(`{"values":[{"sanity":100}]}`), nil
		},
		getDeviceLogsFn: func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
			return json.RawMessage(`{"values":[{"log":"booting"}]}`), nil
		},
		getDeviceStatisticsFn: func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
			return json.RawMessage(`{"data":[{"tx":1000}]}`), nil
		},
		getDeviceStatusFn: func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
			return json.RawMessage(`{"serialNumber":"60cf84f22290","connected":true}`), nil
		},
		getDeviceFn: func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
			return json.RawMessage(`{"serialNumber":"60cf84f22290","compatible":"asus_rt-ax53u"}`), nil
		},
		getDeviceCapabilitiesFn: func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
			return json.RawMessage(`{"serialNumber":"60cf84f22290","capabilities":{"model":"ASUS RT-AX53U"}}`), nil
		},
	}

	app := setupDeviceTestApp(svc)

	tests := []struct {
		method         string
		path           string
		expectedStatus int
	}{
		{http.MethodGet, "/api/v1/devices/60cf84f22290/commands?limit=25&newest=true", http.StatusOK},
		{http.MethodDelete, "/api/v1/devices/60cf84f22290/commands/cmd-999", http.StatusOK},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/healthchecks?limit=10", http.StatusOK},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/logs?limit=50&logType=crash", http.StatusOK},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/statistics?lastOnly=true", http.StatusOK},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/status", http.StatusOK},
		{http.MethodGet, "/api/v1/devices/60cf84f22290/capabilities", http.StatusOK},
		{http.MethodGet, "/api/v1/devices/60cf84f22290", http.StatusOK},
		{http.MethodPut, "/api/v1/devices/60cf84f22290", http.StatusOK},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		req.Header.Set("Authorization", "Bearer valid-jwt")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request %s %s failed: %v", tt.method, tt.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != tt.expectedStatus {
			t.Errorf("expected %d for %s %s, got %d", tt.expectedStatus, tt.method, tt.path, resp.StatusCode)
		}
	}
}

func TestDeviceHandler_ProxyEndpoints_RBACForbidden(t *testing.T) {
	svc := &mockDeviceService{
		getDeviceCommandsFn: func(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
			return nil, models.NewApiError(http.StatusForbidden, "Forbidden", "Device is not assigned to user or access denied")
		},
	}

	app := setupDeviceTestApp(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/unassigned-device/commands", nil)
	req.Header.Set("Authorization", "Bearer rogue-jwt")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", resp.StatusCode)
	}
}

