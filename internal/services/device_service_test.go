package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/mango-mdu-service/internal/services"
)

type mockGWClient struct {
	executeDeviceCommandFn func(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error)
	getDeviceRttyFn        func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error)
	getDevicesStatusFn     func(ctx context.Context, serialNumbers []string, token, reqID, corrID string) ([]models.GatewayDeviceStatus, error)
	getCommandsFn          func(ctx context.Context, queryParams url.Values, token, reqID, corrID string) (json.RawMessage, error)
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
	return json.RawMessage(fmt.Sprintf(`{"UUID":"cmd-1","serialNumber":%q,"command":%q,"status":"completed","errorCode":0}`, serial, command)), nil
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
	if d.DeviceGroup != "ap" {
		t.Errorf("expected DeviceGroup ap, got %s", d.DeviceGroup)
	}
	if d.DeviceType != "asus_rt-ax53u" {
		t.Errorf("expected DeviceType asus_rt-ax53u, got %s", d.DeviceType)
	}
}

func TestDeviceService_GetDevices_PreservesHardwareSkuAndPlatformGroup(t *testing.T) {
	prov := &mockProvClient{
		getInventoryFn: func(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
			return []models.InventoryTag{
				{
					SerialNumber: "085531024340",
					Name:         "Switch-Core",
					DeviceType:   "edgecore_ecs4100_12ph",
					DevClass:     "switch",
				},
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
					SerialNumber: "085531024340",
					Connected:    true,
					DeviceType:   "switch",
					Compatible:   "edgecore_ecs4100_12ph",
				},
			}, nil
		},
	}

	svc := services.NewDeviceService(prov, gw)
	res, err := svc.GetDevices(context.Background(), "valid-jwt", "req-1", "corr-1", 0, 10, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(res.Devices))
	}
	d := res.Devices[0]
	if d.DeviceGroup != "switch" {
		t.Errorf("expected DeviceGroup 'switch', got '%s'", d.DeviceGroup)
	}
	if d.DeviceType != "edgecore_ecs4100_12ph" {
		t.Errorf("expected DeviceType 'edgecore_ecs4100_12ph', got '%s'", d.DeviceType)
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

func TestDeviceService_ApplyProvisionedDeviceConfiguration_DelegatesOnlyToOWPROV(t *testing.T) {
	applyCalls := 0
	gwCalls := 0
	prov := &mockProvClient{
		applyDeviceConfigFn: func(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
			applyCalls++
			if serial != "60cf84f22290" || token != "Bearer jwt" || reqID != "req" || corrID != "corr" {
				t.Fatalf("unexpected apply arguments: serial=%q token=%q reqID=%q corrID=%q", serial, token, reqID, corrID)
			}
			return json.RawMessage(`{"errorCode":0,"appliedConfiguration":true}`), nil
		},
		authorizeDeviceCommandFn: func(context.Context, string, string, string, string, string) (*models.DeviceCommandAuthResult, error) {
			t.Fatal("provisioned apply must rely on OWPROV's inventory authorization, not command authorization")
			return nil, nil
		},
	}
	gw := &mockGWClient{
		executeDeviceCommandFn: func(context.Context, string, string, []byte, string, string, string) (json.RawMessage, error) {
			gwCalls++
			return nil, nil
		},
	}

	result, err := services.NewDeviceService(prov, gw).ApplyProvisionedDeviceConfiguration(
		context.Background(), "60:cf:84:f2:22:90", "Bearer jwt", "req", "corr",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applyCalls != 1 || gwCalls != 0 {
		t.Fatalf("expected one OWPROV call and no OWGW call, got apply=%d gw=%d", applyCalls, gwCalls)
	}
	if string(result) != `{"errorCode":0,"appliedConfiguration":true}` {
		t.Fatalf("unexpected response: %s", result)
	}
}

func TestDeviceService_PushDirectDeviceConfiguration_AuthorizesAndBuildsEnvelope(t *testing.T) {
	authCalls := 0
	gwCalls := 0
	prov := &mockProvClient{
		authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
			authCalls++
			if serial != "60cf84f22290" || action != "CONFIGURE" {
				t.Fatalf("unexpected authorization target: serial=%q action=%q", serial, action)
			}
			return &models.DeviceCommandAuthResult{Allowed: true, Resource: "deviceCommand", Action: "CONFIGURE", SerialNumber: serial}, nil
		},
	}
	gw := &mockGWClient{
		executeDeviceCommandFn: func(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
			gwCalls++
			if serial != "60cf84f22290" || command != "configure" {
				t.Fatalf("unexpected gateway target: serial=%q command=%q", serial, command)
			}
			if !strings.Contains(string(body), `"counter":9007199254740993`) {
				t.Fatalf("large configuration integer was not preserved: %s", body)
			}
			var payload struct {
				SerialNumber  string         `json:"serialNumber"`
				UUID          uint64         `json:"UUID"`
				When          uint64         `json:"when"`
				Configuration map[string]any `json:"configuration"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("decode gateway payload: %v", err)
			}
			if payload.SerialNumber != serial || payload.UUID == 0 || payload.When != 42 {
				t.Fatalf("unexpected gateway payload metadata: %+v", payload)
			}
			if payload.Configuration["radios"] == nil || payload.Configuration["uuid"] != float64(payload.UUID) {
				t.Fatalf("unexpected gateway configuration: %+v", payload.Configuration)
			}
			uuidKeys := 0
			for key := range payload.Configuration {
				if strings.EqualFold(strings.TrimSpace(key), "uuid") {
					uuidKeys++
				}
			}
			if uuidKeys != 1 || len(payload.Configuration) != 2 {
				t.Fatalf("expected one canonical uuid and the supplied settings, got %+v", payload.Configuration)
			}
			return json.RawMessage(`{"UUID":"cmd-1","serialNumber":"60cf84f22290","command":"configure","status":"pending","errorCode":0}`), nil
		},
	}
	requestConfig := map[string]any{
		"uuid":   float64(7),
		"UUID":   float64(8),
		" Uuid ": float64(9),
		"radios": map[string]any{"band": "5G", "counter": json.Number("9007199254740993")},
	}
	result, err := services.NewDeviceService(prov, gw).PushDirectDeviceConfiguration(
		context.Background(), "60-cf-84-f2-22-90", models.DirectDeviceConfigurationRequest{Configuration: requestConfig, When: 42}, "jwt", "req", "corr",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if authCalls != 1 || gwCalls != 1 {
		t.Fatalf("expected one authorization and one gateway call, got auth=%d gw=%d", authCalls, gwCalls)
	}
	if requestConfig["uuid"] != float64(7) {
		t.Fatalf("service mutated caller configuration: %+v", requestConfig)
	}
	if requestConfig["UUID"] != float64(8) || requestConfig[" Uuid "] != float64(9) {
		t.Fatalf("service mutated caller UUID variants: %+v", requestConfig)
	}
	if string(result) != `{"UUID":"cmd-1","serialNumber":"60cf84f22290","command":"configure","status":"pending","errorCode":0}` {
		t.Fatalf("unexpected response: %s", result)
	}
}

func TestDeviceService_ExecuteDeviceCommand_RejectsLegacyConfigureEndpoint(t *testing.T) {
	for _, command := range []string{"configure", "CONFIGURE"} {
		t.Run(command, func(t *testing.T) {
			authCalls := 0
			gatewayCalls := 0
			prov := &mockProvClient{authorizeDeviceCommandFn: func(context.Context, string, string, string, string, string) (*models.DeviceCommandAuthResult, error) {
				authCalls++
				return &models.DeviceCommandAuthResult{Allowed: true}, nil
			}}
			gw := &mockGWClient{executeDeviceCommandFn: func(context.Context, string, string, []byte, string, string, string) (json.RawMessage, error) {
				gatewayCalls++
				return json.RawMessage(`{}`), nil
			}}

			_, err := services.NewDeviceService(prov, gw).ExecuteDeviceCommand(
				context.Background(), "60cf84f22290", command, []byte(`{"configuration":{"radios":{}}}`), "jwt", "req", "corr",
			)
			var apiErr models.ApiError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != http.StatusBadRequest {
				t.Fatalf("expected legacy configure endpoint to return 400, got %v", err)
			}
			if !strings.Contains(apiErr.ErrorDetails, "/configuration/push-direct") {
				t.Fatalf("expected canonical endpoint guidance, got %q", apiErr.ErrorDetails)
			}
			if authCalls != 0 || gatewayCalls != 0 {
				t.Fatalf("legacy configure reached downstream services: auth=%d gateway=%d", authCalls, gatewayCalls)
			}
		})
	}
}

func TestDeviceService_PushDirectDeviceConfiguration_RejectsInvalidOrDeniedRequestsBeforeOWGW(t *testing.T) {
	tests := []struct {
		name          string
		configuration map[string]any
		allowed       bool
		wantStatus    int
		wantAuthCalls int
	}{
		{name: "empty", configuration: map[string]any{}, allowed: true, wantStatus: http.StatusBadRequest},
		{name: "uuid only", configuration: map[string]any{"uuid": 1}, allowed: true, wantStatus: http.StatusBadRequest},
		{name: "denied", configuration: map[string]any{"radios": map[string]any{}}, allowed: false, wantStatus: http.StatusForbidden, wantAuthCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			authCalls := 0
			gwCalls := 0
			prov := &mockProvClient{authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
				authCalls++
				return &models.DeviceCommandAuthResult{Allowed: tc.allowed, Reason: "ACTION_NOT_GRANTED", Details: "configure denied"}, nil
			}}
			gw := &mockGWClient{executeDeviceCommandFn: func(context.Context, string, string, []byte, string, string, string) (json.RawMessage, error) {
				gwCalls++
				return nil, nil
			}}
			_, err := services.NewDeviceService(prov, gw).PushDirectDeviceConfiguration(
				context.Background(), "60cf84f22290", models.DirectDeviceConfigurationRequest{Configuration: tc.configuration}, "jwt", "req", "corr",
			)
			var apiErr models.ApiError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != tc.wantStatus {
				t.Fatalf("expected ApiError status %d, got %T %v", tc.wantStatus, err, err)
			}
			if authCalls != tc.wantAuthCalls || gwCalls != 0 {
				t.Fatalf("unexpected calls: auth=%d gw=%d", authCalls, gwCalls)
			}
		})
	}
}

func TestDeviceService_ExecuteDeviceCommand_InterpretsOWGWStatus(t *testing.T) {
	prov := &mockProvClient{authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
		return &models.DeviceCommandAuthResult{Allowed: true, Resource: "deviceCommand", Action: "REBOOT", SerialNumber: serial}, nil
	}}

	tests := []struct {
		name       string
		response   string
		wantStatus int
	}{
		{name: "pending is preserved", response: `{"UUID":"cmd-1","serialNumber":"60cf84f22290","command":"reboot","status":"pending","errorCode":0}`},
		{name: "completed is preserved", response: `{"UUID":"cmd-2","serialNumber":"60cf84f22290","command":"reboot","status":"completed","errorCode":0}`},
		{name: "failed status is rejected", response: `{"UUID":"cmd-3","serialNumber":"60cf84f22290","command":"reboot","status":"failed","errorCode":0,"errorText":"device rejected command"}`, wantStatus: http.StatusBadRequest},
		{name: "nonzero lowercase error code is rejected", response: `{"UUID":"cmd-4","serialNumber":"60cf84f22290","command":"reboot","status":"completed","errorCode":2,"errorText":"configuration invalid"}`, wantStatus: http.StatusBadRequest},
		{name: "nonzero uppercase string error code is rejected", response: `{"UUID":"cmd-5","serialNumber":"60cf84f22290","command":"reboot","status":"completed","ErrorCode":"7","ErrorDescription":"command failed"}`, wantStatus: http.StatusBadRequest},
		{name: "empty object is rejected", response: `{}`, wantStatus: http.StatusBadGateway},
		{name: "unrelated object is rejected", response: `{"unexpected":"value"}`, wantStatus: http.StatusBadGateway},
		{name: "missing command identity is rejected", response: `{"UUID":"cmd-6","status":"completed","errorCode":0}`, wantStatus: http.StatusBadGateway},
		{name: "different serial is rejected", response: `{"UUID":"cmd-7","serialNumber":"60cf84f22291","command":"reboot","status":"completed","errorCode":0}`, wantStatus: http.StatusBadGateway},
		{name: "null error code is rejected", response: `{"UUID":"cmd-8","serialNumber":"60cf84f22290","command":"reboot","status":"completed","errorCode":null}`, wantStatus: http.StatusBadGateway},
		{name: "concatenated JSON objects are rejected", response: `{"UUID":"cmd-9","serialNumber":"60cf84f22290","command":"reboot","status":"completed","errorCode":0}{"errorCode":7}`, wantStatus: http.StatusBadGateway},
		{name: "trailing non-whitespace content is rejected", response: `{"UUID":"cmd-10","serialNumber":"60cf84f22290","command":"reboot","status":"completed","errorCode":0} trailing`, wantStatus: http.StatusBadGateway},
		{name: "invalid JSON is rejected", response: `not-json`, wantStatus: http.StatusBadGateway},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gw := &mockGWClient{executeDeviceCommandFn: func(context.Context, string, string, []byte, string, string, string) (json.RawMessage, error) {
				return json.RawMessage(tc.response), nil
			}}
			result, err := services.NewDeviceService(prov, gw).ExecuteDeviceCommand(
				context.Background(), "60cf84f22290", "reboot", []byte(`{"when":0}`), "jwt", "req", "corr",
			)
			if tc.wantStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(result) != tc.response {
					t.Fatalf("expected raw OWGW response to be preserved, got %s", result)
				}
				return
			}
			var apiErr models.ApiError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != tc.wantStatus {
				t.Fatalf("expected ApiError status %d, got %T %v", tc.wantStatus, err, err)
			}
		})
	}
}

func TestDeviceService_ExecuteDeviceCommand_ValidatesTelemetryResponseShape(t *testing.T) {
	prov := &mockProvClient{authorizeDeviceCommandFn: func(ctx context.Context, serial, action, token, reqID, corrID string) (*models.DeviceCommandAuthResult, error) {
		return &models.DeviceCommandAuthResult{Allowed: true, Resource: "deviceCommand", Action: "TELEMETRY", SerialNumber: serial}, nil
	}}

	for _, tc := range []struct {
		name       string
		response   string
		wantStatus int
	}{
		{name: "telemetry status is preserved", response: `{"action":"WebSocket telemetry started.","status":{"running":true,"interval":5}}`},
		{name: "missing telemetry status is rejected", response: `{"action":"WebSocket telemetry started."}`, wantStatus: http.StatusBadGateway},
		{name: "missing telemetry action is rejected", response: `{"status":{"running":true}}`, wantStatus: http.StatusBadGateway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := &mockGWClient{executeDeviceCommandFn: func(context.Context, string, string, []byte, string, string, string) (json.RawMessage, error) {
				return json.RawMessage(tc.response), nil
			}}
			result, err := services.NewDeviceService(prov, gw).ExecuteDeviceCommand(
				context.Background(), "60cf84f22290", "telemetry", []byte(`{"serialNumber":"60cf84f22290","types":["health"]}`), "jwt", "req", "corr",
			)
			if tc.wantStatus == 0 {
				if err != nil || string(result) != tc.response {
					t.Fatalf("expected telemetry response to be preserved, got result=%s err=%v", result, err)
				}
				return
			}
			var apiErr models.ApiError
			if !errors.As(err, &apiErr) || apiErr.ErrorCode != tc.wantStatus {
				t.Fatalf("expected ApiError status %d, got %T %v", tc.wantStatus, err, err)
			}
		})
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
	response := json.RawMessage(`{"UUID":"cmd-1","serialNumber":"60cf84f22290","command":"reboot","status":"completed","errorCode":0}`)
	downstreamFailure := errors.New("OWPROV connection failed")

	for _, operation := range []string{"reboot", "rtty", "eventqueue"} {
		t.Run(operation, func(t *testing.T) {
			body := []byte(`{"when":0}`)
			if operation == "eventqueue" {
				body = []byte(`{"serialNumber":"60cf84f22290","types":["dhcp-snooping","wifi-frames"]}`)
			}
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
	for _, operation := range []string{"reboot", "rtty", "eventqueue", "preflight"} {
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
		{"reboot", "REBOOT"}, {"telemetry", "TELEMETRY"},
		{"upgrade", "UPGRADE"}, {"trace", "TRACE"}, {"rtty", "RTTY"}, {"script", "SCRIPT"},
		{"eventqueue", "EVENTQUEUE"}, {"eventqueue", "eventqueue"},
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
				if command == "telemetry" {
					return json.RawMessage(`{"action":"Telemetry status only.","status":{"running":false}}`), nil
				}
				return json.RawMessage(fmt.Sprintf(`{"UUID":"cmd-1","serialNumber":%q,"command":%q,"status":"completed","errorCode":0}`, serial, command)), nil
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
