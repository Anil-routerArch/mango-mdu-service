package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Set required environment variables enforced by discovery/logger common packages
	t.Setenv("SERVICE_NAME", "mango-go-foundation-service")
	t.Setenv("SERVICE_TYPE", "mango-go-foundation-service")
	t.Setenv("SERVICE_VERSION", "dev")
	t.Setenv("SYSTEM_URI_PRIVATE", "https://localhost:17007")
	t.Setenv("SYSTEM_URI_PUBLIC", "https://localhost:8088")
	t.Setenv("DISCOVERY_TOPIC", "service_events")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading configuration defaults, got: %v", err)
	}

	if cfg.Server.HTTPPort != 8088 {
		t.Errorf("expected default HTTPPort to be 8088, got: %d", cfg.Server.HTTPPort)
	}

	if cfg.Server.PrivatePort != 17007 {
		t.Errorf("expected default PrivatePort to be 17007, got: %d", cfg.Server.PrivatePort)
	}

	if cfg.Database.Port != 5432 {
		t.Errorf("expected default Database Port to be 5432, got: %d", cfg.Database.Port)
	}

	if cfg.Database.StorageType != "postgresql" {
		t.Errorf("expected default StorageType to be 'postgresql', got: %s", cfg.Database.StorageType)
	}

	// Verify new feature flag defaults
	if !cfg.Discovery.Enabled {
		t.Errorf("expected default Discovery.Enabled to be true, got false")
	}

	if !cfg.RPC.Enabled {
		t.Errorf("expected default RPC.Enabled to be true, got false")
	}

	if !cfg.Auth.Enabled {
		t.Errorf("expected default Auth.Enabled to be true, got false")
	}
}
