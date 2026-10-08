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
