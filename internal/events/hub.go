package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owprov"
)

// ClientSession represents an active client event stream connection.
type ClientSession struct {
	ID                string
	Token             string
	Send              chan []byte
	AllowedSerials    map[string]struct{}
	SubscribedSerials map[string]struct{}
	mu                sync.RWMutex
	closed            bool
	done              chan struct{}
}

// Close gracefully closes the session and its send channel.
func (s *ClientSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.done)
		close(s.Send)
	}
}

// IsClosed checks if the session is closed.
func (s *ClientSession) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// Done returns the completion notification channel.
func (s *ClientSession) Done() <-chan struct{} {
	return s.done
}

// Hub manages active client sessions and implements the inverted index for O(1) multi-tenant event routing.
type Hub struct {
	mu               sync.RWMutex
	sessions         map[string]*ClientSession
	serialToSessions map[string]map[string]*ClientSession
	provClient       owprov.Client
	logger           *slog.Logger
}

// NewHub initializes an event dispatch Hub.
func NewHub(provClient owprov.Client, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		sessions:         make(map[string]*ClientSession),
		serialToSessions: make(map[string]map[string]*ClientSession),
		provClient:       provClient,
		logger:           logger,
	}
}

// AuthorizeUser queries OWPROV using the caller's JWT token to retrieve all device serials
// the user is authorized to access under tenant/venue/entity RBAC.
func (h *Hub) AuthorizeUser(ctx context.Context, token, reqID, corrID string) ([]string, error) {
	if h.provClient == nil {
		return nil, fmt.Errorf("provClient is uninitialized")
	}

	serials, err := h.provClient.GetInventorySerials(ctx, token, reqID, corrID)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize user inventory: %w", err)
	}

	normalized := make([]string, 0, len(serials))
	for _, s := range serials {
		norm := NormalizeSerial(s)
		if norm != "" {
			normalized = append(normalized, norm)
		}
	}

	return normalized, nil
}

// CreateSession creates and configures a new ClientSession.
func (h *Hub) CreateSession(token string, allowedSerials []string, initialSerials []string) *ClientSession {
	allowedMap := make(map[string]struct{}, len(allowedSerials))
	for _, s := range allowedSerials {
		allowedMap[NormalizeSerial(s)] = struct{}{}
	}

	subMap := make(map[string]struct{})
	for _, s := range initialSerials {
		norm := NormalizeSerial(s)
		if _, ok := allowedMap[norm]; ok {
			subMap[norm] = struct{}{}
		}
	}

	return &ClientSession{
		ID:                uuid.NewString(),
		Token:             token,
		Send:              make(chan []byte, 256),
		AllowedSerials:    allowedMap,
		SubscribedSerials: subMap,
		done:              make(chan struct{}),
	}
}

// RegisterSession registers a client session and updates the inverted index.
func (h *Hub) RegisterSession(session *ClientSession) {
	if session == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.sessions[session.ID] = session
	h.indexSessionSerials(session)

	h.logger.Debug("registered client session",
		"sessionId", session.ID,
		"allowedCount", len(session.AllowedSerials),
		"subscribedCount", len(session.SubscribedSerials),
		"totalSessions", len(h.sessions),
	)
}

// UnregisterSession removes a client session and cleans up its index entries.
func (h *Hub) UnregisterSession(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, exists := h.sessions[sessionID]
	if !exists {
		return
	}

	// Remove from inverted index
	h.deindexSessionSerials(session)
	delete(h.sessions, sessionID)

	session.Close()

	h.logger.Debug("unregistered client session",
		"sessionId", sessionID,
		"remainingSessions", len(h.sessions),
	)
}

// UpdateSubscription modifies the active viewport serials for an existing session.
// CRITICAL SECURITY GUARANTEE: Serials are strictly intersected with session.AllowedSerials.
// Any serial requested that is not in the user's permitted inventory is discarded.
func (h *Hub) UpdateSubscription(sessionID string, requestedSerials []string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, exists := h.sessions[sessionID]
	if !exists {
		return 0, fmt.Errorf("session not found: %s", sessionID)
	}

	// Remove old indexes for this session
	h.deindexSessionSerials(session)

	// Filter requested serials against allowed inventory
	session.mu.Lock()
	newSubMap := make(map[string]struct{})
	for _, s := range requestedSerials {
		norm := NormalizeSerial(s)
		if _, allowed := session.AllowedSerials[norm]; allowed {
			newSubMap[norm] = struct{}{}
		}
	}
	session.SubscribedSerials = newSubMap
	session.mu.Unlock()

	// Re-index new subscriptions
	h.indexSessionSerials(session)

	h.logger.Debug("updated session subscriptions",
		"sessionId", sessionID,
		"requested", len(requestedSerials),
		"granted", len(newSubMap),
	)

	return len(newSubMap), nil
}

// Broadcast distributes a device event to all authorized client sessions in O(1) time.
// Sessions not subscribed or not authorized to view the device receive nothing.
func (h *Hub) Broadcast(event *DeviceEvent) int {
	if event == nil {
		return 0
	}

	serial := NormalizeSerial(event.SerialNumber)
	if serial == "" {
		return 0
	}

	payload, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to marshal device event for broadcast", "error", err, "serial", serial)
		return 0
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	subscribers, ok := h.serialToSessions[serial]
	if !ok || len(subscribers) == 0 {
		return 0
	}

	delivered := 0
	for _, session := range subscribers {
		if session.IsClosed() {
			continue
		}

		select {
		case session.Send <- payload:
			delivered++
		default:
			// Buffer full: drop message for slow client to prevent lagging or head-of-line blocking
			h.logger.Warn("client buffer full, dropped device event",
				"sessionId", session.ID,
				"serial", serial,
			)
		}
	}

	return delivered
}

// GetActiveSessionCount returns the number of currently connected sessions.
func (h *Hub) GetActiveSessionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.sessions)
}

// GetSubscribersForSerial returns the number of sessions currently subscribed to a serial.
func (h *Hub) GetSubscribersForSerial(serial string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	subs, ok := h.serialToSessions[NormalizeSerial(serial)]
	if !ok {
		return 0
	}
	return len(subs)
}

func (h *Hub) indexSessionSerials(session *ClientSession) {
	session.mu.RLock()
	defer session.mu.RUnlock()

	// If viewport subscriptions are specified, index those
	if len(session.SubscribedSerials) > 0 {
		for serial := range session.SubscribedSerials {
			if _, ok := h.serialToSessions[serial]; !ok {
				h.serialToSessions[serial] = make(map[string]*ClientSession)
			}
			h.serialToSessions[serial][session.ID] = session
		}
		return
	}

	// Otherwise, index all allowed serials
	for serial := range session.AllowedSerials {
		if _, ok := h.serialToSessions[serial]; !ok {
			h.serialToSessions[serial] = make(map[string]*ClientSession)
		}
		h.serialToSessions[serial][session.ID] = session
	}
}

func (h *Hub) deindexSessionSerials(session *ClientSession) {
	session.mu.RLock()
	defer session.mu.RUnlock()

	targetSerials := session.SubscribedSerials
	if len(targetSerials) == 0 {
		targetSerials = session.AllowedSerials
	}

	for serial := range targetSerials {
		if subs, ok := h.serialToSessions[serial]; ok {
			delete(subs, session.ID)
			if len(subs) == 0 {
				delete(h.serialToSessions, serial)
			}
		}
	}
}
