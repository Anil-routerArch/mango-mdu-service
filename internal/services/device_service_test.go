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
	executeDeviceCommandFn func(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceRttyFn        func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	getDevicesStatusFn     func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error)
	getCommandsFn          func(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getCommandFn           func(ctx context.Context, commandUUID, token, reqID, corrID string) (json.RawMessage, error)
	deleteCommandFn        func(ctx context.Context, commandUUID, token, reqID, corrID string) error
	getHealthChecksFn      func(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getLogsFn              func(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getStatisticsFn        func(ctx context.Context, serial string, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceStatusFn      func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceFn            func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	updateDeviceFn         func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	getCapabilitiesFn      func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	deleteDeviceFn         func(ctx context.Context, serial, token, reqID, corrID string) error
	getBlacklistFn         func(ctx context.Context, limit, offset int, token, reqID, corrID string) ([]models.BlacklistDevice, error)
	addBlacklistFn         func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	deleteBlacklistFn      func(ctx context.Context, serial, token, reqID, corrID string) error
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
	if m.executeDeviceCommandFn != nil {
		return m.executeDeviceCommandFn(ctx, serial, command, body, token, reqID, corrID)
	}
	return json.RawMessage(`{"result":"ok"}`), nil
}

func (m *mockGWClient) GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	if m.getDeviceRttyFn != nil {
		return m.getDeviceRttyFn(ctx, serial, token, reqID, corrID)
	}
	return json.RawMessage(`{"server":"tty.example.com","viewport":5912,"connectionId":"conn-123"}`), nil
}

func (m *mockGWClient) DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	if m.deleteDeviceFn != nil {
		return m.deleteDeviceFn(ctx, serial, token, reqID, corrID)
	}
	return nil
}

func (m *mockGWClient) GetBlacklist(ctx context.Context, limit, offset int, token, reqID, corrID string) ([]models.BlacklistDevice, error) {
	if m.getBlacklistFn != nil {
		return m.getBlacklistFn(ctx, limit, offset, token, reqID, corrID)
	}
	return []models.BlacklistDevice{}, nil
}

func (m *mockGWClient) AddBlacklistDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	if m.addBlacklistFn != nil {
		return m.addBlacklistFn(ctx, serial, body, token, reqID, corrID)
	}
	return json.RawMessage(`{"serialNumber":"` + serial + `","reason":"test"}`), nil
}

func (m *mockGWClient) DeleteBlacklistDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	if m.deleteBlacklistFn != nil {
		return m.deleteBlacklistFn(ctx, serial, token, reqID, corrID)
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
	if d.Sanity == nil || *d.Sanity != 98.5 {
		t.Errorf("expected sanity 98.5, got %v", d.Sanity)
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

// TestDeviceService_GetDevices_NoSanityWhenConnectedButUnmeasured asserts that a connected
// device whose gateway status carries no sanity measurement returns nil (omitted) rather than
// a fabricated value such as 100.
func TestDeviceService_GetDevices_NoSanityWhenConnectedButUnmeasured(t *testing.T) {
	prov := &mockProvClient{
		getInventoryFn: func(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
			return []models.InventoryTag{
				{SerialNumber: "aabbccddeeff", DeviceType: "ap"},
			}, nil
		},
		getInventoryCountFn: func(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
			return 1, nil
		},
	}
	gw := &mockGWClient{
		getDevicesStatusFn: func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error) {
			return []models.GatewayDeviceStatus{
				{
					SerialNumber: "aabbccddeeff",
					Connected:    true,
					Sanity:       nil, // gateway reports no sanity measurement
				},
			}, nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	res, err := svc.GetDevices(context.Background(), "jwt", "req", "corr", 0, 10, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(res.Devices))
	}
	d := res.Devices[0]
	if !d.Connected {
		t.Errorf("expected connected=true")
	}
	if d.Sanity != nil {
		t.Errorf("expected nil sanity for connected-but-unmeasured device, got %v", *d.Sanity)
	}
}

// TestDeviceService_GetDevices_GatewayUnavailable_NoFakeFirmware asserts that when a device
// has no gateway data the Firmware field is empty (omitted in JSON) rather than the former
// placeholder value "TIP-v4.2".
func TestDeviceService_GetDevices_GatewayUnavailable_NoFakeFirmware(t *testing.T) {
	prov := &mockProvClient{
		getInventoryFn: func(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
			return []models.InventoryTag{
				{SerialNumber: "112233445566", DeviceType: "ap"},
			}, nil
		},
		getInventoryCountFn: func(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
			return 1, nil
		},
	}
	gw := &mockGWClient{
		getDevicesStatusFn: func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error) {
			// Gateway returns empty slice — device not known to GW yet
			return []models.GatewayDeviceStatus{}, nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	res, err := svc.GetDevices(context.Background(), "jwt", "req", "corr", 0, 10, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(res.Devices))
	}
	d := res.Devices[0]
	if d.Firmware != "" {
		t.Errorf("expected empty firmware when gateway data unavailable, got %q", d.Firmware)
	}
	if d.Sanity != nil {
		t.Errorf("expected nil sanity when gateway data unavailable, got %v", *d.Sanity)
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
		authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
			return &models.DeviceCommandAuthResult{
				Allowed: false,
				Reason:  "ACTION_NOT_GRANTED",
				Details: "Action is not granted in policy",
			}, nil
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
	gwDeleteCalled := false
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
			gwDeleteCalled = true
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
	if gwDeleteCalled {
		t.Errorf("OWGW DeleteDevice must NOT be called when OWPROV inventory deletion fails")
	}
}

func TestDeviceService_DeleteDevice_OWPROVUnauthorized_DoesNotDeleteFromOWGW(t *testing.T) {
	gwDeleteCalled := false
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			// Pre-flight READ access passes
			return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
		},
		deleteInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			// Caller only has READ permission, OWPROV rejects DELETE
			return models.NewApiError(http.StatusForbidden, "Forbidden", "delete inventory not permitted")
		},
	}
	gw := &mockGWClient{
		deleteDeviceFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			gwDeleteCalled = true
			return nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	err := svc.DeleteDevice(context.Background(), "60cf84f22290", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected 403 Forbidden error when OWPROV rejects delete")
	}

	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusForbidden {
		t.Errorf("expected 403 ApiError, got: %v", err)
	}
	if gwDeleteCalled {
		t.Errorf("SECURITY VULNERABILITY: OWGW DeleteDevice was called before OWPROV deletion authorization!")
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

func TestDeviceService_GetBlacklist_ScopesToUserInventory(t *testing.T) {
	prov := &mockProvClient{
		getInventorySerialsFn: func(ctx context.Context, token, reqID, corrID string) ([]string, error) {
			return []string{"60cf84f22290"}, nil
		},
	}
	gw := &mockGWClient{
		getBlacklistFn: func(ctx context.Context, limit, offset int, token, reqID, corrID string) ([]models.BlacklistDevice, error) {
			return []models.BlacklistDevice{
				{SerialNumber: "60cf84f22290", Reason: "Tenant AP rogue", Author: "tenant@example.com"},
				{SerialNumber: "aabbccddeeff", Reason: "Other tenant AP", Author: "other@example.com"},
			}, nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	devices, err := svc.GetBlacklist(context.Background(), "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(devices) != 1 {
		t.Fatalf("expected 1 scoped blacklist device, got %d", len(devices))
	}
	if devices[0].SerialNumber != "60cf84f22290" {
		t.Errorf("expected serial 60cf84f22290, got %s", devices[0].SerialNumber)
	}
}

func TestDeviceService_AddBlacklistDevice_AuthorizedAndUnauthorized(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			if serial == "60cf84f22290" {
				return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
			}
			return nil, models.NewApiError(http.StatusNotFound, "Not Found", "device not in inventory")
		},
	}
	addCalled := false
	gw := &mockGWClient{
		addBlacklistFn: func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
			addCalled = true
			return json.RawMessage(`{"serialNumber":"` + serial + `"}`), nil
		},
	}

	svc := services.NewDeviceService(prov, gw)

	// 1. Unauthorized device - should be blocked with 403 Forbidden
	_, err := svc.AddBlacklistDevice(context.Background(), "unauthorized-dev", "test reason", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error for unauthorized device, got nil")
	}
	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got: %v", err)
	}
	if addCalled {
		t.Errorf("AddBlacklistDevice in gateway should NOT have been called for unauthorized device")
	}

	// 2. Authorized device - should succeed
	res, err := svc.AddBlacklistDevice(context.Background(), "60cf84f22290", "test reason", "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected error for authorized device: %v", err)
	}
	if !addCalled {
		t.Errorf("AddBlacklistDevice in gateway should have been called")
	}
	if len(res) == 0 {
		t.Errorf("expected non-empty response")
	}
}

func TestDeviceService_DeleteBlacklistDevice_AuthorizedAndUnauthorized(t *testing.T) {
	prov := &mockProvClient{
		getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
			if serial == "60cf84f22290" {
				return &models.InventoryTag{SerialNumber: "60cf84f22290"}, nil
			}
			return nil, models.NewApiError(http.StatusNotFound, "Not Found", "device not in inventory")
		},
	}
	deleteCalled := false
	gw := &mockGWClient{
		deleteBlacklistFn: func(ctx context.Context, serial, token, reqID, corrID string) error {
			deleteCalled = true
			return nil
		},
	}

	svc := services.NewDeviceService(prov, gw)

	// 1. Unauthorized device - should be blocked with 403 Forbidden
	err := svc.DeleteBlacklistDevice(context.Background(), "unauthorized-dev", "jwt", "req", "corr")
	if err == nil {
		t.Fatalf("expected error for unauthorized device, got nil")
	}
	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got: %v", err)
	}
	if deleteCalled {
		t.Errorf("DeleteBlacklistDevice in gateway should NOT have been called for unauthorized device")
	}

	// 2. Authorized device - should succeed
	err = svc.DeleteBlacklistDevice(context.Background(), "60cf84f22290", "jwt", "req", "corr")
	if err != nil {
		t.Fatalf("unexpected error for authorized device: %v", err)
	}
	if !deleteCalled {
		t.Errorf("DeleteBlacklistDevice in gateway should have been called")
	}
}

func TestDeviceService_CommandAuthorizationEnforcement(t *testing.T) {
	const serial = "60cf84f22290"
	const token = "Bearer delegated-user-token"
	const reqID = "req-authz"
	const corrID = "corr-authz"
	body := []byte(`{"when":0}`)
	response := json.RawMessage(`{"result":"ok"}`)
	downstreamFailure := errors.New("OWPROV connection failed")

	for _, operation := range []string{"reboot", "rtty"} {
		t.Run(operation, func(t *testing.T) {
			for _, tc := range []struct {
				name        string
				allowed     bool
				reason      string
				details     string
				authErr     error
				noProv      bool
				wantStatus  int
				wantDetails string
			}{
				{name: "allowed", allowed: true},
				{name: "action denied", reason: "ACTION_NOT_GRANTED", details: "Policy does not grant this command", wantStatus: http.StatusForbidden, wantDetails: "Policy does not grant this command"},
				{name: "out of scope", reason: "OUT_OF_SCOPE", wantStatus: http.StatusForbidden, wantDetails: "OUT_OF_SCOPE"},
				{name: "authorization connection failure", authErr: downstreamFailure},
				{name: "invalid user token", authErr: models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid token"), wantStatus: http.StatusUnauthorized},
				{name: "authorization service unavailable", authErr: models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "OWPROV unavailable"), wantStatus: http.StatusServiceUnavailable},
				{name: "missing authorization client", noProv: true, wantStatus: http.StatusServiceUnavailable},
			} {
				t.Run(tc.name, func(t *testing.T) {
					authCalls, commandCalls, rttyCalls := 0, 0, 0
					authorized := false
					checkArgs := func(gotSerial, gotToken, gotReqID, gotCorrID string) {
						t.Helper()
						if gotSerial != serial || gotToken != token || gotReqID != reqID || gotCorrID != corrID {
							t.Fatalf("incorrect forwarded arguments: serial=%q token=%q reqID=%q corrID=%q", gotSerial, gotToken, gotReqID, gotCorrID)
						}
					}
					prov := &mockProvClient{
						authorizeDeviceCommandFn: func(ctx context.Context, gotSerial, action, gotToken, gotReqID, gotCorrID string) (*models.DeviceCommandAuthResult, error) {
							authCalls++
							checkArgs(gotSerial, gotToken, gotReqID, gotCorrID)
							if action != operation {
								t.Fatalf("expected authorization for %q, got %q", operation, action)
							}
							if commandCalls != 0 || rttyCalls != 0 {
								t.Fatal("OWGW was called before OWPROV authorization")
							}
							if tc.authErr != nil {
								return nil, tc.authErr
							}
							authorized = tc.allowed
							return &models.DeviceCommandAuthResult{
								Allowed: tc.allowed, Reason: tc.reason, Details: tc.details,
								Resource: "deviceCommand", SerialNumber: gotSerial, Action: action,
							}, nil
						},
					}
					gw := &mockGWClient{
						executeDeviceCommandFn: func(ctx context.Context, gotSerial, command string, gotBody []byte, gotToken, gotReqID, gotCorrID string) (json.RawMessage, error) {
							commandCalls++
							if !authorized {
								t.Fatal("OWGW command called without an allowed authorization decision")
							}
							checkArgs(gotSerial, gotToken, gotReqID, gotCorrID)
							if command != operation || string(gotBody) != string(body) {
								t.Fatalf("incorrect forwarded command/body: %q %s", command, gotBody)
							}
							return response, nil
						},
						getDeviceRttyFn: func(ctx context.Context, gotSerial, gotToken, gotReqID, gotCorrID string) (json.RawMessage, error) {
							rttyCalls++
							if !authorized {
								t.Fatal("OWGW RTTY called without an allowed authorization decision")
							}
							checkArgs(gotSerial, gotToken, gotReqID, gotCorrID)
							return response, nil
						},
					}
					svc := services.NewDeviceService(prov, gw)
					if tc.noProv {
						svc = services.NewDeviceService(nil, gw)
					}
					var result json.RawMessage
					var err error
					if operation == "rtty" {
						result, err = svc.GetDeviceRtty(context.Background(), "60:cf:84:f2:22:90", token, reqID, corrID)
					} else {
						result, err = svc.ExecuteDeviceCommand(context.Background(), "60:cf:84:f2:22:90", operation, body, token, reqID, corrID)
					}
					wantAuthCalls := 1
					if tc.noProv {
						wantAuthCalls = 0
					}
					if authCalls != wantAuthCalls {
						t.Fatalf("expected %d OWPROV calls, got %d", wantAuthCalls, authCalls)
					}
					if tc.allowed {
						if err != nil || string(result) != string(response) {
							t.Fatalf("expected gateway response, got result=%s err=%v", result, err)
						}
						if (operation == "rtty" && (rttyCalls != 1 || commandCalls != 0)) || (operation != "rtty" && (commandCalls != 1 || rttyCalls != 0)) {
							t.Fatalf("incorrect gateway call counts: commands=%d rtty=%d", commandCalls, rttyCalls)
						}
						return
					}
					if commandCalls != 0 || rttyCalls != 0 {
						t.Fatalf("authorization failure reached OWGW: commands=%d rtty=%d", commandCalls, rttyCalls)
					}
					if err == nil || len(result) != 0 {
						t.Fatalf("expected failure without gateway response, got result=%s err=%v", result, err)
					}
					if tc.authErr != nil && !errors.Is(err, tc.authErr) {
						t.Fatalf("authorization error not propagated: got %v, want %v", err, tc.authErr)
					}
					if tc.wantStatus != 0 {
						var apiErr models.ApiError
						if !errors.As(err, &apiErr) || apiErr.ErrorCode != tc.wantStatus {
							t.Fatalf("expected HTTP %d error, got %v", tc.wantStatus, err)
						}
						if tc.wantDetails != "" && apiErr.ErrorDetails != tc.wantDetails {
							t.Fatalf("expected denial details %q, got %q", tc.wantDetails, apiErr.ErrorDetails)
						}
					}
				})
			}
		})
	}
}

func TestDeviceService_RejectsInvalidCommandAuthorizationTarget(t *testing.T) {
	for _, operation := range []string{"reboot", "rtty", "preflight"} {
		t.Run(operation, func(t *testing.T) {
			for _, tc := range []struct {
				name      string
				change    func(*models.DeviceCommandAuthResult)
				nilResult bool
			}{
				{name: "different serial", change: func(r *models.DeviceCommandAuthResult) { r.SerialNumber = "60cf84f22291" }},
				{name: "missing serial", change: func(r *models.DeviceCommandAuthResult) { r.SerialNumber = "" }},
				{name: "different resource", change: func(r *models.DeviceCommandAuthResult) { r.Resource = "inventory" }},
				{name: "missing resource", change: func(r *models.DeviceCommandAuthResult) { r.Resource = "" }},
				{name: "different action", change: func(r *models.DeviceCommandAuthResult) { r.Action = "UPGRADE" }},
				{name: "missing action", change: func(r *models.DeviceCommandAuthResult) { r.Action = "" }},
				{name: "unknown action", change: func(r *models.DeviceCommandAuthResult) { r.Action = "UNKNOWN" }},
				{name: "FULL is not a command", change: func(r *models.DeviceCommandAuthResult) { r.Action = "FULL" }},
				{name: "missing decision", nilResult: true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					gatewayCalls := 0
					prov := &mockProvClient{authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
						if tc.nilResult {
							return nil, nil
						}
						result := &models.DeviceCommandAuthResult{Allowed: true, Resource: "deviceCommand", SerialNumber: serial, Action: action}
						tc.change(result)
						return result, nil
					}}
					gw := &mockGWClient{
						executeDeviceCommandFn: func(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
							gatewayCalls++
							return json.RawMessage(`{}`), nil
						},
						getDeviceRttyFn: func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
							gatewayCalls++
							return json.RawMessage(`{}`), nil
						},
					}
					svc := services.NewDeviceService(prov, gw)
					var err error
					switch operation {
					case "rtty":
						_, err = svc.GetDeviceRtty(context.Background(), "60cf84f22290", "token", "req", "corr")
					case "preflight":
						_, err = svc.AuthorizeDeviceCommand(context.Background(), "60cf84f22290", "reboot", "token", "req", "corr")
					default:
						_, err = svc.ExecuteDeviceCommand(context.Background(), "60cf84f22290", operation, nil, "token", "req", "corr")
					}
					var apiErr models.ApiError
					if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadGateway {
						t.Fatalf("expected 502 for invalid authorization decision, got %v", err)
					}
					if gatewayCalls != 0 {
						t.Fatalf("invalid authorization decision reached OWGW %d times", gatewayCalls)
					}
				})
			}
		})
	}
}

func TestDeviceService_CommandAuthorizationAliases(t *testing.T) {
	for _, tc := range []struct{ requested, returned string }{
		{"reboot", "REBOOT"}, {"configure", "CONFIGURE"}, {"telemetry", "TELEMETRY"},
		{"upgrade", "UPGRADE"}, {"trace", "TRACE"}, {"rtty", "RTTY"}, {"script", "SCRIPT"},
		{"leds", "BLINK"}, {"blink", "LEDS"}, {"leds", "LEDS"},
		{"wifiscan", "WIFI_SCAN"}, {"wifi_scan", "WIFISCAN"}, {"wifiscan", "WIFISCAN"},
		{"factory", "FACTORY_RESET"}, {"factoryreset", "FACTORY"}, {"factory_reset", "FACTORYRESET"},
		{"reenroll", "RE_ENROLL"}, {"re_enroll", "REENROLL"},
	} {
		t.Run(tc.requested+"/"+tc.returned, func(t *testing.T) {
			gatewayCalls := 0
			prov := &mockProvClient{authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
				if serial != "aabbccddeeff" || action != tc.requested {
					t.Fatalf("unexpected authorization request: serial=%q action=%q", serial, action)
				}
				return &models.DeviceCommandAuthResult{Allowed: true, Resource: "deviceCommand", SerialNumber: "AA:BB:CC:DD:EE:FF", Action: tc.returned}, nil
			}}
			checkGateway := func(serial, command string) (json.RawMessage, error) {
				gatewayCalls++
				if serial != "aabbccddeeff" || command != tc.requested {
					t.Fatalf("unexpected gateway request: serial=%q command=%q", serial, command)
				}
				return json.RawMessage(`{"result":"ok"}`), nil
			}
			gw := &mockGWClient{
				executeDeviceCommandFn: func(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
					return checkGateway(serial, command)
				},
				getDeviceRttyFn: func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
					return checkGateway(serial, "rtty")
				},
			}
			svc := services.NewDeviceService(prov, gw)
			var err error
			if tc.requested == "rtty" {
				_, err = svc.GetDeviceRtty(context.Background(), "aa-bb-cc-dd-ee-ff", "token", "req", "corr")
			} else {
				_, err = svc.ExecuteDeviceCommand(context.Background(), "aa-bb-cc-dd-ee-ff", tc.requested, nil, "token", "req", "corr")
			}
			if err != nil || gatewayCalls != 1 {
				t.Fatalf("expected matching decision to execute once, got calls=%d err=%v", gatewayCalls, err)
			}
		})
	}
}

func TestDeviceService_UnknownRequestedCommandCannotMatchUnknownDecision(t *testing.T) {
	prov := &mockProvClient{authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
		return &models.DeviceCommandAuthResult{Allowed: true, Resource: "deviceCommand", SerialNumber: serial, Action: action}, nil
	}}
	gw := &mockGWClient{executeDeviceCommandFn: func(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
		t.Fatal("unknown command reached OWGW")
		return nil, nil
	}}
	svc := services.NewDeviceService(prov, gw)
	_, err := svc.ExecuteDeviceCommand(context.Background(), "60cf84f22290", "UNKNOWN", nil, "token", "req", "corr")
	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadGateway {
		t.Fatalf("expected 502 for unknown command decision, got %v", err)
	}
}

func TestDeviceService_UpdateDevice_NotesOnly(t *testing.T) {
	for _, body := range []string{
		`{"serialNumber":"60cf84f22290","notes":[{"note":"  Checked cabling  ","created":0}]}`,
		`{"serialNumber":"60:CF:84:F2:22:90","notes":[{"note":"Checked cabling","created":123}]}`,
		`{"notes":[{"note":"Checked cabling"}]}`,
	} {
		t.Run(body, func(t *testing.T) {
			readCalls, gatewayCalls := 0, 0
			prov := &mockProvClient{getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
				readCalls++
				if serial != "60cf84f22290" || token != "Bearer reader" || reqID != "req" || corrID != "corr" {
					t.Fatal("incorrect inventory READ request")
				}
				return &models.InventoryTag{SerialNumber: serial}, nil
			}, authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
				t.Fatal("notes must not require device-command authorization")
				return nil, nil
			}}
			gw := &mockGWClient{updateDeviceFn: func(ctx context.Context, serial string, payload []byte, token, reqID, corrID string) (json.RawMessage, error) {
				gatewayCalls++
				if readCalls != 1 || serial != "60cf84f22290" || token != "Bearer reader" || reqID != "req" || corrID != "corr" {
					t.Fatal("gateway update must follow inventory READ and preserve delegated arguments")
				}
				var sent struct {
					SerialNumber string           `json:"serialNumber"`
					Notes        []map[string]any `json:"notes"`
				}
				if err := json.Unmarshal(payload, &sent); err != nil {
					t.Fatal(err)
				}
				if sent.SerialNumber != serial || len(sent.Notes) != 1 || len(sent.Notes[0]) != 1 || sent.Notes[0]["note"] != "Checked cabling" {
					t.Fatalf("expected sanitized notes without client timestamp or author, got %s", payload)
				}
				return json.RawMessage(`{"notes":[{"note":"Checked cabling","created":100,"createdBy":"reader@example.com"}]}`), nil
			}}
			svc := services.NewDeviceService(prov, gw)
			result, err := svc.UpdateDevice(context.Background(), "60-cf-84-f2-22-90", []byte(body), "Bearer reader", "req", "corr")
			if err != nil || len(result) == 0 || readCalls != 1 || gatewayCalls != 1 {
				t.Fatalf("expected one authorized notes update, got result=%s err=%v readCalls=%d gatewayCalls=%d", result, err, readCalls, gatewayCalls)
			}
		})
	}
}

func TestDeviceService_UpdateDevice_RejectsInvalidOrUnsupportedUpdates(t *testing.T) {
	bodies := []string{
		``, `null`, `[]`, `{}`, `{"notes":null}`, `{"notes":[]}`,
		`{"notes":[{}]}`, `{"notes":[null]}`, `{"notes":[{"note":" \t "}]}`,
		`{"notes":[{"note":42}]}`, `{"notes":[{"note":"ok"},{"note":""}]}`,
		`{"notes":[{"note":"ok","createdBy":"root@example.com"}]}`,
		`{"notes":[{"note":"ok","configuration":"injected"}]}`,
		`{"notes":[{"note":"ok","created":"invalid"}]}`,
		`{"notes":[{"note":"ok"}]} {"configuration":"injected"}`,
		`{"notes":[{"note":"ok"}]} trailing`,
		`{"serialNumber":"different-device","notes":[{"note":"ok"}]}`,
	}
	for _, field := range []string{"configuration", "entity", "venue", "subscriber", "owner", "location", "unknown"} {
		bodies = append(bodies, `{"`+field+`":"injected"}`, `{"notes":[{"note":"ok"}],"`+field+`":"injected"}`)
	}
	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			prov := &mockProvClient{getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
				t.Fatal("invalid input must be rejected before downstream inventory requests")
				return nil, nil
			}}
			gw := &mockGWClient{updateDeviceFn: func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
				t.Fatal("invalid or non-note update reached OWGW")
				return nil, nil
			}}
			svc := services.NewDeviceService(prov, gw)
			_, err := svc.UpdateDevice(context.Background(), "60cf84f22290", []byte(body), "Bearer reader", "req", "corr")
			var apiErr models.ApiError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadRequest {
				t.Fatalf("expected 400 for invalid notes update, got %v", err)
			}
		})
	}
}

func TestDeviceService_UpdateDevice_ReadDenialStopsGateway(t *testing.T) {
	prov := &mockProvClient{getInventoryItemFn: func(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
		return nil, models.NewApiError(http.StatusForbidden, "Forbidden", "No inventory READ access")
	}}
	gw := &mockGWClient{updateDeviceFn: func(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
		t.Fatal("notes update without inventory READ reached OWGW")
		return nil, nil
	}}
	svc := services.NewDeviceService(prov, gw)
	_, err := svc.UpdateDevice(context.Background(), "60cf84f22290", []byte(`{"notes":[{"note":"Checked cabling"}]}`), "Bearer reader", "req", "corr")
	var apiErr models.ApiError
	if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusForbidden {
		t.Fatalf("expected 403 for missing inventory READ, got %v", err)
	}
}
