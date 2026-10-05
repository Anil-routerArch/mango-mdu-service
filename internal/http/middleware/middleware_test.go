package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/http/middleware"
)

func TestRegisterPublicCORS(t *testing.T) {
	app := fiber.New()
	middleware.RegisterPublicCORS(app)

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, X-Request-Id, X-Correlation-Id")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		t.Errorf("expected 204 or 200 for CORS preflight, got %d", resp.StatusCode)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowHeaders, "X-Request-Id") {
		t.Errorf("expected Access-Control-Allow-Headers to contain X-Request-Id, got %q", allowHeaders)
	}
	if !strings.Contains(allowHeaders, "X-Correlation-Id") {
		t.Errorf("expected Access-Control-Allow-Headers to contain X-Correlation-Id, got %q", allowHeaders)
	}
}
