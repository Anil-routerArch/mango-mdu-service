package events_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/events"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
)

type mockProvClientForHub struct {
	getInventorySerialsFn func(ctx context.Context, token, reqID, corrID string) ([]string, error)
}

func (m *mockProvClientForHub) GetPolicy(ctx context.Context, id, token, reqID, corrID string) (*models.ManagementPolicy, error) {
	return nil, nil
}
func (m *mockProvClientForHub) GetRolesByPolicy(ctx context.Context, policyID, token, reqID, corrID string) ([]models.ManagementRole, error) {
	return nil, nil
}
func (m *mockProvClientForHub) GetEntities(ctx context.Context, token, reqID, corrID string) ([]models.Entity, error) {
	return nil, nil
}
func (m *mockProvClientForHub) GetVenues(ctx context.Context, token, reqID, corrID string) ([]models.Venue, error) {
	return nil, nil
}
func (m *mockProvClientForHub) GetInventory(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
	return nil, nil
}
func (m *mockProvClientForHub) GetInventoryCount(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
	return 0, nil
}
func (m *mockProvClientForHub) GetInventorySerials(ctx context.Context, token, reqID, corrID string) ([]string, error) {
	if m.getInventorySerialsFn != nil {
		return m.getInventorySerialsFn(ctx, token, reqID, corrID)
	}
	return nil, nil
}
func (m *mockProvClientForHub) GetInventoryItem(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
	return nil, nil
}
func (m *mockProvClientForHub) DeleteInventoryItem(ctx context.Context, serial, token, reqID, corrID string) error {
	return nil
}

func TestHub_AuthorizeUser(t *testing.T) {
	mockProv := &mockProvClientForHub{
		getInventorySerialsFn: func(ctx context.Context, token, reqID, corrID string) ([]string, error) {
			if token != "valid-jwt" {
				return nil, models.NewApiError(401, "Unauthorized", "invalid token")
			}
			return []string{"24:F5:A2:01:B1:A0", "AA-BB-CC-DD-EE-FF"}, nil
		},
	}

	hub := events.NewHub(mockProv, nil)
	serials, err := hub.AuthorizeUser(context.Background(), "valid-jwt", "req-1", "corr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(serials) != 2 {
		t.Fatalf("expected 2 serials, got %d", len(serials))
	}
	if serials[0] != "24f5a201b1a0" || serials[1] != "aabbccddeeff" {
		t.Errorf("unexpected normalized serials: %v", serials)
	}
}

// TestHub_MultiTenantIsolation verifies that:
// 1. Client A (Tenant A) receives updates ONLY for Tenant A devices.
// 2. Client B (Tenant B) receives updates ONLY for Tenant B devices.
// 3. Client B CANNOT subscribe to Tenant A devices (privilege escalation prevention).
// 4. Events for unknown devices are received by NO clients.
func TestHub_MultiTenantIsolation(t *testing.T) {
	hub := events.NewHub(nil, nil)

	// User A has access to alpha1, alpha2
	sessionA := hub.CreateSession("token-A", []string{"alpha1", "alpha2"}, nil)
	hub.RegisterSession(sessionA)
	defer hub.UnregisterSession(sessionA.ID)

	// User B has access to beta1, beta2
	sessionB := hub.CreateSession("token-B", []string{"beta1", "beta2"}, nil)
	hub.RegisterSession(sessionB)
	defer hub.UnregisterSession(sessionB.ID)

	if hub.GetActiveSessionCount() != 2 {
		t.Fatalf("expected 2 active sessions, got %d", hub.GetActiveSessionCount())
	}

	// Verify initial index
	if hub.GetSubscribersForSerial("alpha1") != 1 {
		t.Errorf("expected 1 subscriber for alpha1, got %d", hub.GetSubscribersForSerial("alpha1"))
	}
	if hub.GetSubscribersForSerial("beta1") != 1 {
		t.Errorf("expected 1 subscriber for beta1, got %d", hub.GetSubscribersForSerial("beta1"))
	}

	// 1. Broadcast event for alpha1 (Tenant A device)
	connTrue := true
	hub.Broadcast(&events.DeviceEvent{
		Type:         "connection",
		SerialNumber: "alpha1",
		Connected:    &connTrue,
		Timestamp:    100,
	})

	// User A must receive alpha1
	select {
	case msg := <-sessionA.Send:
		var ev events.DeviceEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if ev.SerialNumber != "alpha1" {
			t.Errorf("expected serial alpha1, got %s", ev.SerialNumber)
		}
	case <-time.After(50 * time.Millisecond):
		t.Errorf("sessionA timed out waiting for alpha1 event")
	}

	// User B must receive NOTHING
	select {
	case msg := <-sessionB.Send:
		t.Fatalf("SECURITY VIOLATION: User B received unauthorized event for alpha1: %s", string(msg))
	default:
		// expected
	}

	// 2. Broadcast event for beta1 (Tenant B device)
	hub.Broadcast(&events.DeviceEvent{
		Type:         "healthcheck",
		SerialNumber: "beta1",
		Timestamp:    200,
	})

	// User B must receive beta1
	select {
	case msg := <-sessionB.Send:
		var ev events.DeviceEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if ev.SerialNumber != "beta1" {
			t.Errorf("expected serial beta1, got %s", ev.SerialNumber)
		}
	case <-time.After(50 * time.Millisecond):
		t.Errorf("sessionB timed out waiting for beta1 event")
	}

	// User A must receive NOTHING
	select {
	case msg := <-sessionA.Send:
		t.Fatalf("SECURITY VIOLATION: User A received unauthorized event for beta1: %s", string(msg))
	default:
		// expected
	}

	// 3. User B attempts PRIVILEGE ESCALATION: explicitly requests subscription to alpha1!
	granted, err := hub.UpdateSubscription(sessionB.ID, []string{"alpha1", "beta1"})
	if err != nil {
		t.Fatalf("unexpected error updating subscription: %v", err)
	}
	// alpha1 must have been rejected because it's not in User B's allowedSerials!
	if granted != 1 {
		t.Errorf("expected only 1 granted subscription (beta1), got %d", granted)
	}

	// Broadcast alpha1 again
	hub.Broadcast(&events.DeviceEvent{
		Type:         "state",
		SerialNumber: "alpha1",
		Timestamp:    300,
	})

	// Drain User A
	<-sessionA.Send

	// Verify User B STILL received NOTHING
	select {
	case msg := <-sessionB.Send:
		t.Fatalf("SECURITY VIOLATION: User B successfully escalated privileges and received alpha1 event: %s", string(msg))
	default:
		// expected
	}

	// 4. Broadcast for unowned device gamma1
	hub.Broadcast(&events.DeviceEvent{
		Type:         "connection",
		SerialNumber: "gamma1",
		Timestamp:    400,
	})

	select {
	case msg := <-sessionA.Send:
		t.Fatalf("unexpected event on session A: %s", string(msg))
	default:
	}

	select {
	case msg := <-sessionB.Send:
		t.Fatalf("unexpected event on session B: %s", string(msg))
	default:
	}
}

func TestHub_UnregisterSession(t *testing.T) {
	hub := events.NewHub(nil, nil)

	session := hub.CreateSession("token-1", []string{"dev1"}, nil)
	hub.RegisterSession(session)

	if hub.GetActiveSessionCount() != 1 {
		t.Errorf("expected 1 session, got %d", hub.GetActiveSessionCount())
	}
	if hub.GetSubscribersForSerial("dev1") != 1 {
		t.Errorf("expected 1 subscriber for dev1, got %d", hub.GetSubscribersForSerial("dev1"))
	}

	hub.UnregisterSession(session.ID)

	if hub.GetActiveSessionCount() != 0 {
		t.Errorf("expected 0 sessions, got %d", hub.GetActiveSessionCount())
	}
	if hub.GetSubscribersForSerial("dev1") != 0 {
		t.Errorf("expected 0 subscribers for dev1, got %d", hub.GetSubscribersForSerial("dev1"))
	}
	if !session.IsClosed() {
		t.Errorf("expected session to be closed")
	}
}
