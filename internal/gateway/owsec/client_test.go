package owsec_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owsec"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

func TestOWSecClient_GetUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-sec-token" {
			t.Errorf("missing or invalid auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "mango-mdu-service/1.0" {
			t.Errorf("missing user-agent: %s", r.Header.Get("User-Agent"))
		}

		resp := models.SecUserListResponse{
			Users: []models.SecUser{
				{ID: "usr-1", Name: "Anita Sharma", Email: "anita@example.com", UserRole: "noc"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owsec.NewClient(owsec.Config{URLResolver: func() string { return ts.URL }})
	users, err := client.GetUsers(context.Background(), "test-sec-token", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 || users[0].Name != "Anita Sharma" {
		t.Errorf("unexpected users: %+v", users)
	}
}

func TestOWSecClient_Unreachable(t *testing.T) {
	client := owsec.NewClient(owsec.Config{URLResolver: func() string { return "http://127.0.0.1:1" }}) // closed port
	_, err := client.GetUsers(context.Background(), "token", "req", "corr")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(models.ApiError)
	if !ok || apiErr.ErrorCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 ApiError, got %+v", err)
	}
}

func TestOWSecClient_DualAuthentication(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-INTERNAL-NAME") != "https://localhost:17005" {
			t.Errorf("missing or incorrect X-INTERNAL-NAME: %s", r.Header.Get("X-INTERNAL-NAME"))
		}
		if r.Header.Get("X-API-KEY") != "test-mdu-api-key" {
			t.Errorf("missing or incorrect X-API-KEY: %s", r.Header.Get("X-API-KEY"))
		}
		if r.Header.Get("Authorization") != "Bearer test-user-token" {
			t.Errorf("missing or incorrect Authorization: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Request-Id") != "req-dual-1" {
			t.Errorf("missing or incorrect X-Request-Id: %s", r.Header.Get("X-Request-Id"))
		}
		if r.Header.Get("X-Correlation-Id") != "corr-dual-1" {
			t.Errorf("missing or incorrect X-Correlation-Id: %s", r.Header.Get("X-Correlation-Id"))
		}

		resp := models.SecUserListResponse{
			Users: []models.SecUser{
				{ID: "usr-2", Name: "Bob Smith", Email: "bob@example.com", UserRole: "admin"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := owsec.NewClient(owsec.Config{
		URLResolver:  func() string { return ts.URL },
		InternalName: "https://localhost:17005",
		InternalKey:  "test-mdu-api-key",
	})
	users, err := client.GetUsers(context.Background(), "test-user-token", "req-dual-1", "corr-dual-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 || users[0].Name != "Bob Smith" {
		t.Errorf("unexpected users: %+v", users)
	}
}

