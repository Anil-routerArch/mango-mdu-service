package events_test

import (
	"testing"

	"github.com/routerarchitects/mango-mdu-service/internal/events"
)

func TestParseConnectionEvent_Connect(t *testing.T) {
	raw := `{
		"system": {"id": 1, "host": "owgw.example.com"},
		"payload": {
			"serialNumber": "24:F5:A2:01:B1:A0",
			"uuid": 1001,
			"firmware": "OpenWrt 21.02-SNAPSHOT",
			"connectionIp": "192.168.1.100",
			"timestamp": 1729850000
		}
	}`

	event, err := events.ParseKafkaMessage("connection", []byte("24f5a201b1a0"), []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Type != "connection" {
		t.Errorf("expected type 'connection', got %q", event.Type)
	}
	if event.SerialNumber != "24f5a201b1a0" {
		t.Errorf("expected normalized serial '24f5a201b1a0', got %q", event.SerialNumber)
	}
	if event.Connected == nil || !*event.Connected {
		t.Errorf("expected connected=true")
	}
	if event.Firmware != "OpenWrt 21.02-SNAPSHOT" {
		t.Errorf("expected firmware 'OpenWrt 21.02-SNAPSHOT', got %q", event.Firmware)
	}
	if event.ConnectionIP != "192.168.1.100" {
		t.Errorf("expected connectionIp '192.168.1.100', got %q", event.ConnectionIP)
	}
	if event.Timestamp != 1729850000 {
		t.Errorf("expected timestamp 1729850000, got %d", event.Timestamp)
	}
}

func TestParseConnectionEvent_Disconnect(t *testing.T) {
	raw := `{
		"system": {"id": 1, "host": "owgw.example.com"},
		"payload": {
			"disconnection": {
				"serialNumber": "24-F5-A2-01-B1-A0",
				"timestamp": 1729851000,
				"uuid": 1001
			}
		}
	}`

	event, err := events.ParseKafkaMessage("connection", []byte("24f5a201b1a0"), []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Type != "connection" {
		t.Errorf("expected type 'connection', got %q", event.Type)
	}
	if event.SerialNumber != "24f5a201b1a0" {
		t.Errorf("expected normalized serial '24f5a201b1a0', got %q", event.SerialNumber)
	}
	if event.Connected == nil || *event.Connected {
		t.Errorf("expected connected=false")
	}
	if event.Timestamp != 1729851000 {
		t.Errorf("expected timestamp 1729851000, got %d", event.Timestamp)
	}
}

func TestParseStateEvent_Success(t *testing.T) {
	raw := `{
		"system": {"id": 1, "host": "owgw.example.com"},
		"payload": {
			"serial": "24f5a201b1a0",
			"state": {
				"unit": {
					"memory": {
						"free": 64000000,
						"total": 256000000
					},
					"load": [0.25, 0.15, 0.05]
				},
				"uptime": 86400,
				"timestamp": 1729850500
			}
		}
	}`

	event, err := events.ParseKafkaMessage("state", []byte("24f5a201b1a0"), []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Type != "state" {
		t.Errorf("expected type 'state', got %q", event.Type)
	}
	if event.SerialNumber != "24f5a201b1a0" {
		t.Errorf("expected serial '24f5a201b1a0', got %q", event.SerialNumber)
	}
	if event.MemoryFree != 64000000 || event.MemoryTotal != 256000000 {
		t.Errorf("unexpected memory: free=%d, total=%d", event.MemoryFree, event.MemoryTotal)
	}
	// Used = 192000000 / 256000000 = 75%
	if event.MemoryPct != 75.0 {
		t.Errorf("expected memory pct 75.0, got %f", event.MemoryPct)
	}
	if len(event.Load) != 3 || event.Load[0] != 0.25 {
		t.Errorf("unexpected load: %v", event.Load)
	}
}

func TestParseHealthcheckEvent_Success(t *testing.T) {
	raw := `{
		"system": {"id": 1, "host": "owgw.example.com"},
		"payload": {
			"serial": "24f5a201b1a0",
			"sanity": 95,
			"recorded": 1729850800
		}
	}`

	event, err := events.ParseKafkaMessage("healthcheck", []byte("24f5a201b1a0"), []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Type != "healthcheck" {
		t.Errorf("expected type 'healthcheck', got %q", event.Type)
	}
	if event.SerialNumber != "24f5a201b1a0" {
		t.Errorf("expected serial '24f5a201b1a0', got %q", event.SerialNumber)
	}
	if event.Sanity == nil || *event.Sanity != 95 {
		t.Errorf("expected sanity 95, got %v", event.Sanity)
	}
	if event.Timestamp != 1729850800 {
		t.Errorf("expected timestamp 1729850800, got %d", event.Timestamp)
	}
}

func TestParseKafkaMessage_SerialFallbackFromKey(t *testing.T) {
	raw := `{
		"payload": {
			"state": {
				"unit": {
					"memory": {"free": 100, "total": 200}
				}
			}
		}
	}`

	event, err := events.ParseKafkaMessage("state", []byte("aa:bb:cc:dd:ee:ff"), []byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.SerialNumber != "aabbccddeeff" {
		t.Errorf("expected serial to fallback to key 'aabbccddeeff', got %q", event.SerialNumber)
	}
}

func TestParseKafkaMessage_InvalidInputs(t *testing.T) {
	// Empty payload
	_, err := events.ParseKafkaMessage("connection", []byte("key"), nil)
	if err == nil {
		t.Errorf("expected error for empty payload")
	}

	// Unsupported topic
	_, err = events.ParseKafkaMessage("unsupported_topic", []byte("key"), []byte(`{"data":1}`))
	if err == nil {
		t.Errorf("expected error for unsupported topic")
	}

	// Malformed JSON
	_, err = events.ParseKafkaMessage("connection", []byte("key"), []byte(`{invalid-json`))
	if err == nil {
		t.Errorf("expected error for malformed json")
	}
}
