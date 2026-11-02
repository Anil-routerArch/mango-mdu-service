package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/http/handlers"
	"github.com/routerarchitects/mango-mdu-service/internal/http/middleware"
	"github.com/routerarchitects/mango-mdu-service/internal/http/routes"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	subsysteroutes "github.com/routerarchitects/ow-common-mods/fiber/system-routes"
)

type dummyPolicyService struct{}

func (d *dummyPolicyService) GetPolicyOverview(ctx context.Context, policyID, token, reqID, corrID string) (*models.PolicyOverviewResponse, error) {
	return &models.PolicyOverviewResponse{}, nil
}

type dummyDeviceService struct{}

func (d *dummyDeviceService) GetDevices(ctx context.Context, token, reqID, corrID string, pageIndex, pageSize int, unassignedOnly bool) (*models.PaginatedDevicesResponse, error) {
	return &models.PaginatedDevicesResponse{}, nil
}

func (d *dummyDeviceService) GetDeviceCommands(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	return json.RawMessage(`{"commands":[]}`), nil
}

func (d *dummyDeviceService) DeleteDeviceCommand(ctx context.Context, serial, commandUUID, token, reqID, corrID string) error {
	return nil
}

func (d *dummyDeviceService) GetDeviceHealthChecks(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	return json.RawMessage(`{"healthchecks":[]}`), nil
}

func (d *dummyDeviceService) GetDeviceLogs(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	return json.RawMessage(`{"values":[]}`), nil
}

func (d *dummyDeviceService) GetDeviceStatistics(ctx context.Context, serial, token, reqID, corrID string, queryParams url.Values) (json.RawMessage, error) {
	return json.RawMessage(`{"data":[]}`), nil
}

func (d *dummyDeviceService) GetDeviceStatus(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"status":{}}`), nil
}

func (d *dummyDeviceService) GetDevice(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"serialNumber":"60cf84f22290"}`), nil
}

func (d *dummyDeviceService) UpdateDevice(ctx context.Context, serial string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"serialNumber":"` + serial + `"}`), nil
}

func (d *dummyDeviceService) GetDeviceCapabilities(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"capabilities":{"model":"ASUS RT-AX53U"}}`), nil
}

func (d *dummyDeviceService) ExecuteDeviceCommand(ctx context.Context, serial, command string, body []byte, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"result":"ok"}`), nil
}

func (d *dummyDeviceService) GetDeviceRtty(ctx context.Context, serial, token, reqID, corrID string) (json.RawMessage, error) {
	return json.RawMessage(`{"server":"tty.example.com","viewport":5912,"connectionId":"conn-123"}`), nil
}

func (d *dummyDeviceService) DeleteDevice(ctx context.Context, serial, token, reqID, corrID string) error {
	return nil
}

func TestRouteVisibility(t *testing.T) {
	policyHdlr := handlers.NewPolicyHandler(&dummyPolicyService{})
	deviceHdlr := handlers.NewDeviceHandler(&dummyDeviceService{})
	eventsHdlr := handlers.NewDeviceEventsHandler(nil, nil)

	// 1. Setup Public App
	publicApp := fiber.New()
	publicDeps := routes.PublicDeps{
		AuthHandler:         func(c fiber.Ctx) error { return c.Next() },
		Subsystem:           subsysteroutes.Config{},
		PolicyHandler:       policyHdlr,
		DeviceHandler:       deviceHdlr,
		DeviceEventsHandler: eventsHdlr,
	}
	routes.RegisterPublic(publicApp, publicDeps)

	// 2. Setup Private App
	privateApp := fiber.New()
	privateDeps := routes.PrivateDeps{
		AuthHandler: func(c fiber.Ctx) error { return c.Next() },
		Subsystem:   subsysteroutes.Config{},
	}
	routes.RegisterPrivate(privateApp, privateDeps)

	// Check if public app has system route and policy overview route registered
	hasPublicSystemRoute := false
	hasPolicyOverviewRoute := false
	hasDevicesRoute := false
	hasEventsRoute := false
	hasSubscriptionsRoute := false
	hasWsRoute := false
	hasRootWsRoute := false
	for _, route := range publicApp.GetRoutes() {
		if route.Path == "/api/v1/system" {
			hasPublicSystemRoute = true
		}
		if route.Path == "/api/v1/policy/:id/overview" {
			hasPolicyOverviewRoute = true
		}
		if route.Path == "/api/v1/devices" {
			hasDevicesRoute = true
		}
		if route.Path == "/api/v1/devices/events" {
			hasEventsRoute = true
		}
		if route.Path == "/api/v1/devices/events/subscriptions" {
			hasSubscriptionsRoute = true
		}
		if route.Path == "/api/v1/ws" {
			hasWsRoute = true
		}
		if route.Path == "/ws" {
			hasRootWsRoute = true
		}
	}
	if !hasPublicSystemRoute {
		t.Errorf("expected public app to register /api/v1/system route, but it did not")
	}
	if !hasPolicyOverviewRoute {
		t.Errorf("expected public app to register /api/v1/policy/:id/overview route, but it did not")
	}
	if !hasDevicesRoute {
		t.Errorf("expected public app to register /api/v1/devices route, but it did not")
	}
	if !hasEventsRoute {
		t.Errorf("expected public app to register /api/v1/devices/events route, but it did not")
	}
	if !hasSubscriptionsRoute {
		t.Errorf("expected public app to register /api/v1/devices/events/subscriptions route, but it did not")
	}
	if !hasWsRoute {
		t.Errorf("expected public app to register /api/v1/ws route, but it did not")
	}
	if !hasRootWsRoute {
		t.Errorf("expected public app to register /ws route, but it did not")
	}

	// Check if private app has system route registered
	hasPrivateSystemRoute := false
	for _, route := range privateApp.GetRoutes() {
		if route.Path == "/api/v1/system" {
			hasPrivateSystemRoute = true
			break
		}
	}
	if !hasPrivateSystemRoute {
		t.Errorf("expected private app to register /api/v1/system route, but it did not")
	}
}

func TestCORSPreflight_TC_POL_008(t *testing.T) {
	app := fiber.New()
	middleware.RegisterPublicCORS(app)

	policyHdlr := handlers.NewPolicyHandler(&dummyPolicyService{})
	routes.RegisterPublic(app, routes.PublicDeps{
		AuthHandler:   func(c fiber.Ctx) error { return c.Next() },
		Subsystem:     subsysteroutes.Config{},
		PolicyHandler: policyHdlr,
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/policy/523e4567-e89b-12d3-a456-426614174000/overview", nil)
	req.Header.Set("Origin", "https://operator.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, X-Request-Id, X-Correlation-Id")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("expected 204 No Content or 200 OK for CORS preflight, got %d", resp.StatusCode)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Errorf("expected Access-Control-Allow-Headers to be set, but got empty")
	}
}

func TestQueryTokenAuthPropagation(t *testing.T) {
	app := fiber.New()
	var capturedAuthHeader string

	routes.RegisterPublic(app, routes.PublicDeps{
		AuthHandler: func(c fiber.Ctx) error {
			capturedAuthHeader = c.Get("Authorization")
			return c.SendStatus(http.StatusOK)
		},
		Subsystem: subsysteroutes.Config{},
	})

	// Send request with ?token=xyz without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system?token=my-secret-token", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected app.Test error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if capturedAuthHeader != "Bearer my-secret-token" {
		t.Errorf("expected 'Bearer my-secret-token', got %q", capturedAuthHeader)
	}
}

