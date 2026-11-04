package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/mango-mdu-service/internal/services"
)

type mockGWClient struct {
	getDevicesStatusFn func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error)
	getCommandsFn      func(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getCommandFn       func(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error)
	deleteCommandFn    func(ctx context.Context, commandUUID, token, reqID, corrID string) error
	getHealthChecksFn  func(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getLogsFn          func(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getStatisticsFn    func(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceStatusFn  func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceFn        func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	updateDeviceFn     func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	getCapabilitiesFn  func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	deleteDeviceFn     func(ctx context.Context, serial, token, reqID, corrID string) error
}

func (m *mockGWClient) GetDevicesStatus(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error) {
	if m.getDevicesStatusFn != nil {
		return m.getDevicesStatusFn(ctx, serialNumbers, token, reqID, corrID)
	}
	return []models.GatewayDeviceStatus{}, nil
}

func (m *mockGWClient) GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getDeviceFn != nil {
		return m.getDeviceFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{}`), nil
}

func (m *mockGWClient) UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	if m.updateDeviceFn != nil {
		return m.updateDeviceFn(ctx, serial, body, token, reqID, corrID)
	}
	return json.RawMessage(`{"serialNumber":"` + serial + `"}`), nil
}

func (m *mockGWClient) GetCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getCapabilitiesFn != nil {
		return m.getCapabilitiesFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{"capabilities":{"model":"ASUS RT-AX53U"}}`), nil
}

func (m *mockGWClient) GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getDeviceStatusFn != nil {
		return m.getDeviceStatusFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{}`), nil
}

func (m *mockGWClient) GetCommands(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getCommandsFn != nil {
		return m.getCommandsFn(ctx, queryParams, token, reqID, corrID)
	}
	return json.RawMessage(`{"commands":[]}`), nil
}

func (m *mockGWClient) GetCommand(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getCommandFn != nil {
		return m.getCommandFn(ctx, commandUUID, token, reqID, corrID)
	}
	return json.RawMessage(`{"UUID":"` + commandUUID + `","serialNumber":"60cf84f22290"}`), nil
}

func (m *mockGWClient) DeleteCommand(ctx context.Context, commandUUID, token, reqID, corrID string) error {
	if m.deleteCommandFn != nil {
		return m.deleteCommandFn(ctx, commandUUID, token, reqID, corrID)
	}
	return nil
}

func (m *mockGWClient) GetHealthChecks(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getHealthChecksFn != nil {
		return m.getHealthChecksFn(ctx, serial, queryParams, token, reqID, corrID)
	}
	return json.RawMessage(`{"values":[]}`), nil
}

func (m *mockGWClient) GetLogs(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getLogsFn != nil {
		return m.getLogsFn(ctx, serial, queryParams, token, reqID, corrID)
	}
	return json.RawMessage(`{"values":[]}`), nil
}

func (m *mockGWClient) GetStatistics(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getStatisticsFn != nil {
		return m.getStatisticsFn(ctx, serial, queryParams, token, reqID, corrID)
	}
	return json.RawMessage(`{"data":[]}`), nil
}

func (m *mockGWClient) ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"result":"ok"}`), nil
}

func (m *mockGWClient) GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"server":"tty.example.com","viewport":5912,"connectionId":"conn-123"}`), nil
}

func (m *mockGWClient) DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	if m.deleteDeviceFn != nil {
		return m.deleteDeviceFn(ctx, serial, token, reqID, corrID)
	}
	return nil
}

func TestDeviceService_GetDevices_Success(t *testing.T) {
	prov := &mockProvClient{
		getInventoryFn: func(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
			if token != "valid-jwt" {
				t.Errorf("expected valid-jwt, got %s", token)
			}
			return []models.InventoryTag{
				{
					SerialNumber: "60cf84f22290",
					Name:         "AP-Lobby",
					DeviceType:   "asus_rt-ax53u",
					ExtendedInfo: &models.InventoryExtendedInfo{
						Venue: &models.ExtendedVenue{
							ID:   "ven-1",
							Name: "North Building",
						},
					},
				},
			}, nil
		},
		getInventoryCountFn: func(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
			return 1, nil
		},
	}

	sanityVal := 98.5
	gw := &mockGWClient{
		getDevicesStatusFn: func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error) {
			if len(serialNumbers) != 1 || serialNumbers[0] != "60cf84f22290" {
				t.Errorf("unexpected serialNumbers: %v", serialNumbers)
			}
			return []models.GatewayDeviceStatus{
				{
					SerialNumber:   "60cf84f22290",
					Connected:      true,
					Sanity:         &sanityVal,
					IPAddress:      "192.168.1.50",
					Firmware:       "TIP-v4.2.4-1a0ba9bf",
					Associations2G: 3,
					Associations5G: 5,
				},
			}, nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	res, err := svc.GetDevices(context.Background(), "valid-jwt", "req-1", "corr-1", 0, 10, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TotalCount != 1 {
		t.Errorf("expected TotalCount 1, got %d", res.TotalCount)
	}
	if len(res.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(res.Devices))
	}

	d := res.Devices[0]
	if d.SerialNumber != "60cf84f22290" {
		t.Errorf("expected 60cf84f22290, got %s", d.SerialNumber)
	}
	if !d.Connected {
		t.Errorf("expected connected=true")
	}
	if d.Sanity != 98.5 {
		t.Errorf("expected sanity 98.5, got %f", d.Sanity)
	}
	if d.IPAddress != "192.168.1.50" {
		t.Errorf("expected IP 192.168.1.50, got %s", d.IPAddress)
	}
	if d.VenueName != "North Building" {
		t.Errorf("expected VenueName North Building, got %s", d.VenueName)
	}
	if d.AssociationsTotal != 8 {
		t.Errorf("expected 8 associations total, got %d", d.AssociationsTotal)
	}
}

func TestDeviceService_GetDevices_GatewayGracefulDegradation(t *testing.T) {
	prov := &mockProvClient{
		getInventoryFn: func(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
			return []models.InventoryTag{
				{
					SerialNumber: "dc62796520cd",
					DeviceType:   "ap",
				},
			}, nil
		},
		getInventoryCountFn: func(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
			return 1, nil
		},
	}

	gw := &mockGWClient{
		getDevicesStatusFn: func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error) {
			return nil, errors.New("owgw unreachable")
		},
	}

	svc := services.NewDeviceService(prov, gw)
	res, err := svc.GetDevices(context.Background(), "jwt", "req", "corr", 0, 10, false)
	if err != nil {
		t.Fatalf("expected graceful degradation, got error: %v", err)
	}

	if len(res.Devices) != 1 {
		t.Fatalf("expected 1 device from inventory, got %d", len(res.Devices))
	}
	if res.Devices[0].Connected {
		t.Errorf("expected connected=false during degradation, got true")
	}
}

func TestDeviceService_RBAC_Authorized(t *testing.T) {
	calledGW := false
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			if serial != "60cf84f22290" {
				return nil, errors.New("not found")
			}
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
	}
	gw := &mockGWClient{
		getCommandsFn: func(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
			calledGW = true
			if queryParams.Get("serialNumber") != "60cf84f22290" {
				t.Errorf("expected serial 60cf84f22290, got %s", queryParams.Get("serialNumber"))
			}
			return json.RawMessage(`{"commands":[{"command":"Configure"}]}`), nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	res, err := svc.GetDeviceCommands(context.Background(), "60:cf:84:f2:22:90", "jwt", "req", "corr", url.Values{"limit": []string{"25"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledGW {
		t.Errorf("expected downstream OWGW to be queried for authorized device")
	}
	if string(res) != `{"commands":[{"command":"Configure"}]}` {
		t.Errorf("unexpected response: %s", string(res))
	}
}

func TestDeviceService_RBAC_Unauthorized(t *testing.T) {
	calledGW := false
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return nil, errors.New("device not assigned to user")
		},
	}
	gw := &mockGWClient{
		getCommandsFn: func(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error) {
			calledGW = true
			return nil, nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	_, err := svc.GetDeviceCommands(context.Background(), "unauthorized-serial", "jwt", "req", "corr", url.Values{})
	if err == nil {
		t.Fatalf("expected error for unauthorized device, got nil")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected ApiError, got %T: %v", err, err)
	}
	if apiErr.ErrorCode != 403 {
		t.Errorf("expected status 403 Forbidden, got %d", apiErr.ErrorCode)
	}
	if calledGW {
		t.Errorf("security violation: OWGW must NEVER be called when device is not assigned to user")
	}
}

func TestDeviceService_DeviceActions_RBAC(t *testing.T) {
	authorizedProv := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
	}
	gw := &mockGWClient{}
	svc := services.NewDeviceService(authorizedProv, gw)

	// Authorized RTTY
	rttyRes, err := svc.GetDeviceRtty(context.Background(), "60:cf:84:f2:22:90", "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected RTTY error: %v", err)
	}
	if len(rttyRes) == 0 {
		t.Errorf("expected non-empty RTTY response")
	}

	// Authorized Command (Reboot)
	cmdRes, err := svc.ExecuteDeviceCommand(context.Background(), "60cf84f22290", "reboot", []byte(`{"when":0}`), "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected Command error: %v", err)
	}
	if len(cmdRes) == 0 {
		t.Errorf("expected non-empty Command response")
	}

	// Authorized Delete
	err = svc.DeleteDevice(context.Background(), "60cf84f22290", "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected Delete error: %v", err)
	}

	// Unauthorized tests
	unauthorizedProv := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return nil, errors.New("forbidden")
		},
	}
	unauthSvc := services.NewDeviceService(unauthorizedProv, gw)

	_, err = unauthSvc.GetDeviceRtty(context.Background(), "attacker-serial", "jwt", "req", "corr")
	if err == nil {
		t.Errorf("expected 403 Forbidden for RTTY on unauthorized device")
	}

	_, err = unauthSvc.ExecuteDeviceCommand(context.Background(), "attacker-serial", "reboot", nil, "jwt", "req", "corr")
	if err == nil {
		t.Errorf("expected 403 Forbidden for Command on unauthorized device")
	}

	err = unauthSvc.DeleteDevice(context.Background(), "attacker-serial", "jwt", "req", "corr")
	if err == nil {
		t.Errorf("expected 403 Forbidden for Delete on unauthorized device")
	}
}

func TestDeviceService_DeleteDevice_OWPROVFailure(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
		deleteInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			return models.NewApiError(http.StatusInternalServerError, "Internal Server Error", "database error in owprov")
		},
	}
	gw := &mockGWClient{
		deleteDeviceFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			return nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDevice(context.Background(), "60cf84f22290", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error when OWPROV delete fails")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusInternalServerError {
		t.Errorf("expected 500 ApiError, got: %v", err)
	}
}

func TestDeviceService_DeleteDevice_OWGW404_Ignored(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
		deleteInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			return nil
		},
	}
	gw := &mockGWClient{
		deleteDeviceFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			// Device is offline / not in gateway cache
			return models.NewApiError(http.StatusNotFound, "Not Found", "device not in gateway")
		},
	}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDevice(context.Background(), "60cf84f22290", "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("expected success when OWGW returns 404 but OWPROV succeeds, got: %v", err)
	}
}

func TestDeviceService_DeleteDevice_OWGWFailure_ReturnsError(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
		deleteInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			return nil
		},
	}
	gw := &mockGWClient{
		deleteDeviceFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			return models.NewApiError(http.StatusBadGateway, "Bad Gateway", "gateway unreachable")
		},
	}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDevice(context.Background(), "60cf84f22290", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error when OWGW has non-404 hard failure")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadGateway {
		t.Errorf("expected 502 ApiError, got: %v", err)
	}
}

func TestDeviceService_DeleteDeviceCommand_Success(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
	}
	deleteCalled := false
	gw := &mockGWClient{
		getCommandFn: func(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error) {
			if commandUUID != "cmd-123" {
				t.Errorf("expected cmd-123, got %s", commandUUID)
			}
			return json.RawMessage(`{"UUID":"cmd-123","serialNumber":"60cf84f22290"}`), nil
		},
		deleteCommandFn: func(ctx context.Context, commandUUID, token, reqID, corrID string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDeviceCommand(context.Background(), "60:cf:84:f2:22:90", "cmd-123", "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if !deleteCalled {
		t.Errorf("expected deleteCommand to be invoked")
	}
}

func TestDeviceService_DeleteDeviceCommand_MismatchSerial_CrossTenantAttackRejected(t *testing.T) {
	// Attacker has access to Device A (60cf84f22290)
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
	}
	deleteCalled := false
	gw := &mockGWClient{
		getCommandFn: func(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error) {
			// Command actually belongs to Victim Device B (24f5a201b1a0)
			return json.RawMessage(`{"UUID":"cmd-victim-999","serialNumber":"24f5a201b1a0"}`), nil
		},
		deleteCommandFn: func(ctx context.Context, commandUUID, token, reqID, corrID string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	// Attacker attempts to delete victim's command through Device A's authorized path
	err := svc.DeleteDeviceCommand(context.Background(), "60cf84f22290", "cmd-victim-999", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error on cross-device command deletion attempt, got nil")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got: %v", err)
	}
	if deleteCalled {
		t.Errorf("SECURITY FAILURE: deleteCommand was called for another device's command!")
	}
}

func TestDeviceService_DeleteDeviceCommand_CommandNotFoundInGW(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
	}
	deleteCalled := false
	gw := &mockGWClient{
		getCommandFn: func(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error) {
			return nil, models.NewApiError(http.StatusNotFound, "Not Found", "command not found")
		},
		deleteCommandFn: func(ctx context.Context, commandUUID, token, reqID, corrID string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDeviceCommand(context.Background(), "60cf84f22290", "nonexistent-cmd", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error when command is not found in gateway, got nil")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got: %v", err)
	}
	if deleteCalled {
		t.Errorf("deleteCommand should not be called when command is not found")
	}
}

func TestDeviceService_DeleteDeviceCommand_DeviceUnauthorized(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			return nil, models.NewApiError(http.StatusForbidden, "Forbidden", "access denied")
		},
	}
	gw := &mockGWClient{}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDeviceCommand(context.Background(), "unauthorized-serial", "cmd-1", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error for unauthorized device access, got nil")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got: %v", err)
	}
}
