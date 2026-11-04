package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/events"
	"github.com/routerarchitects/mango-mdu-service/internal/http/handlers"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

type mockProvForEventsHandler struct {
	getSerialsFn func(ctx context.Context, token, reqID, corrID string) ([]string, error)
}

func (m *mockProvForEventsHandler) GetPolicy(ctx context.Context, id, token, reqID, corrID string) (*models.ManagementPolicy, error) {
	return nil, nil
}
func (m *mockProvForEventsHandler) GetRolesByPolicy(ctx context.Context, policyID, token, reqID, corrID string) ([]models.ManagementRole, error) {
	return nil, nil
}
func (m *mockProvForEventsHandler) GetEntities(ctx context.Context, token, reqID, corrID string) ([]models.Entity, error) {
	return nil, nil
}
func (m *mockProvForEventsHandler) GetVenues(ctx context.Context, token, reqID, corrID string) ([]models.Venue, error) {
	return nil, nil
}
func (m *mockProvForEventsHandler) GetInventory(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
	return nil, nil
}
func (m *mockProvForEventsHandler) GetInventoryCount(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
	return 0, nil
}
func (m *mockProvForEventsHandler) GetInventorySerials(ctx context.Context, token, reqID, corrID string) ([]string, error) {
	if m.getSerialsFn != nil {
		return m.getSerialsFn(ctx, token, reqID, corrID)
	}
	return nil, nil
}
func (m *mockProvForEventsHandler) GetInventoryItem(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
	return nil, nil
}
func (m *mockProvForEventsHandler) DeleteInventoryItem(ctx context.Context, serial, token, reqID, corrID string) error {
	return nil
}

func setupEventsApp(hub *events.Hub, validator ...handlers.TokenValidator) (*fiber.App, *handlers.DeviceEventsHandler) {
	app := fiber.New()
	var val handlers.TokenValidator
	if len(validator) > 0 {
		val = validator[0]
	}
	handler := handlers.NewDeviceEventsHandler(hub, val)
	app.Get("/api/v1/ws", handler.HandleWebSocket)
	app.Get("/api/v1/devices/events", handler.StreamEvents)
	app.Post("/api/v1/devices/events/subscriptions", handler.UpdateSubscriptions)
	return app, handler
}

func TestStreamEvents_MissingToken(t *testing.T) {
	hub := events.NewHub(nil, nil)
	app, _ := setupEventsApp(hub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/events", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestStreamEvents_AuthFailure(t *testing.T) {
	mockProv := &mockProvForEventsHandler{
		getSerialsFn: func(ctx context.Context, token, reqID, corrID string) ([]string, error) {
			return nil, models.NewApiError(401, "Unauthorized", "bad token")
		},
	}
	hub := events.NewHub(mockProv, nil)
	app, _ := setupEventsApp(hub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/events", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

type mockTokenValidator struct {
	validateFn func(ctx context.Context, rawToken string) error
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, rawToken string) error {
	if m.validateFn != nil {
		return m.validateFn(ctx, rawToken)
	}
	return nil
}

func TestUpdateSubscriptions_MissingToken(t *testing.T) {
	hub := events.NewHub(nil, nil)
	app, _ := setupEventsApp(hub)

	body, _ := json.Marshal(map[string]interface{}{
		"sessionId": "some-session",
		"serials":   []string{"dev1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/events/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized when missing token, got %d", resp.StatusCode)
	}
}

func TestUpdateSubscriptions_InvalidBody(t *testing.T) {
	hub := events.NewHub(nil, nil)
	app, _ := setupEventsApp(hub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/events/subscriptions", bytes.NewBufferString(`{invalid`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestUpdateSubscriptions_SessionNotFound(t *testing.T) {
	hub := events.NewHub(nil, nil)
	app, _ := setupEventsApp(hub)

	body, _ := json.Marshal(map[string]interface{}{
		"sessionId": "non-existent-session",
		"serials":   []string{"dev1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/events/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", resp.StatusCode)
	}
}

func TestUpdateSubscriptions_OwnershipMismatch_Forbidden(t *testing.T) {
	hub := events.NewHub(nil, nil)
	session := hub.CreateSession("alice-token", []string{"dev1", "dev2"}, nil)
	hub.RegisterSession(session)
	defer hub.UnregisterSession(session.ID)

	app, _ := setupEventsApp(hub)

	// Attacker Bob attempts to update Alice's session
	body, _ := json.Marshal(map[string]interface{}{
		"sessionId": session.ID,
		"serials":   []string{"dev1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/events/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer bob-attacker-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected status 403 Forbidden for ownership mismatch, got %d", resp.StatusCode)
	}
}

func TestUpdateSubscriptions_TokenValidator_Error(t *testing.T) {
	hub := events.NewHub(nil, nil)
	validator := &mockTokenValidator{
		validateFn: func(ctx context.Context, rawToken string) error {
			return errors.New("token expired or revoked")
		},
	}
	app, _ := setupEventsApp(hub, validator)

	body, _ := json.Marshal(map[string]interface{}{
		"sessionId": "any-session",
		"serials":   []string{"dev1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/events/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer expired-token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 Unauthorized on failed token validation, got %d", resp.StatusCode)
	}
}

func TestUpdateSubscriptions_Success(t *testing.T) {
	hub := events.NewHub(nil, nil)
	session := hub.CreateSession("token", []string{"dev1", "dev2"}, nil)
	hub.RegisterSession(session)
	defer hub.UnregisterSession(session.ID)

	app, _ := setupEventsApp(hub)

	body, _ := json.Marshal(map[string]interface{}{
		"sessionId": session.ID,
		"serials":   []string{"dev1"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/events/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result)
	}
	if result["subscribedCount"] != float64(1) {
		t.Errorf("expected subscribedCount=1, got %v", result["subscribedCount"])
	}
}

func TestWebSocketHandler_Registered(t *testing.T) {
	hub := events.NewHub(nil, nil)
	app, handler := setupEventsApp(hub)
	if handler == nil {
		t.Fatal("expected handler not to be nil")
	}

	routes := app.GetRoutes()
	hasWs := false
	for _, r := range routes {
		if r.Path == "/api/v1/ws" && r.Method == "GET" {
			hasWs = true
			break
		}
	}
	if !hasWs {
		t.Errorf("expected /api/v1/ws GET route to be registered on app")
	}
}
