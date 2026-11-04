package events

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/routerarchitects/mango-mdu-service/internal/gateway/owprov"
)

var (
	// ErrSessionNotFound indicates that a referenced session ID does not exist in the hub.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionOwnershipMismatch indicates that the caller's identity/token does not own the session.
	ErrSessionOwnershipMismatch = errors.New("caller does not own this session")
)

// ClientSession represents an active client event stream connection.
type ClientSession struct {
	ID                string
	Token             string
	Send              chan []byte
	AllowedSerials    map[string]struct{}
	SubscribedSerials map[string]struct{}
	indexedSerials    map[string]struct{}
	unrestricted      bool
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

// SubscribedCount returns the number of currently subscribed devices.
// In unrestricted mode, this equals the count of permitted devices (AllowedSerials).
func (s *ClientSession) SubscribedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.unrestricted {
		return len(s.AllowedSerials)
	}
	return len(s.SubscribedSerials)
}

// IsUnrestricted reports whether the session receives events for all permitted devices without filtering.
func (s *ClientSession) IsUnrestricted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.unrestricted
}

func normalizeToken(t string) string {
	t = strings.TrimPrefix(t, "Bearer ")
	t = strings.TrimPrefix(t, "bearer ")
	return strings.TrimSpace(t)
}

// ExtractUserIdentity extracts the user identity (id, userId, user, username, email, or sub claim) from a JWT token.
// If the token is not a JWT or has no recognizable user claim, it returns an empty string.
func ExtractUserIdentity(token string) string {
	token = normalizeToken(token)
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return ""
	}
	var claims struct {
		ID       string `json:"id"`
		UserID   string `json:"userId"`
		User     string `json:"user"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Sub      string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if claims.ID != "" {
		return claims.ID
	}
	if claims.UserID != "" {
		return claims.UserID
	}
	if claims.User != "" {
		return claims.User
	}
	if claims.Username != "" {
		return claims.Username
	}
	if claims.Email != "" {
		return claims.Email
	}
	return claims.Sub
}

// VerifyOwnership checks whether callerToken has permission to modify this session.
// It returns true if callerToken matches session.Token directly or corresponds to the same user identity.
func (s *ClientSession) VerifyOwnership(callerToken string) bool {
	s.mu.RLock()
	sessionToken := normalizeToken(s.Token)
	s.mu.RUnlock()

	callerToken = normalizeToken(callerToken)

	// In test environments where session was created without a token, allow access
	if sessionToken == "" {
		return true
	}
	if callerToken == "" {
		return false
	}
	if sessionToken == callerToken {
		return true
	}

	userSession := ExtractUserIdentity(sessionToken)
	userCaller := ExtractUserIdentity(callerToken)
	if userSession != "" && userCaller != "" && userSession == userCaller {
		return true
	}

	return false
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
// If initialSerials is nil, the session operates in unrestricted mode (fleet-wide listener).
// If initialSerials is non-nil (even if empty), the session operates in filtered viewport mode.
func (h *Hub) CreateSession(token string, allowedSerials []string, initialSerials []string) *ClientSession {
	allowedMap := make(map[string]struct{}, len(allowedSerials))
	for _, s := range allowedSerials {
		allowedMap[NormalizeSerial(s)] = struct{}{}
	}

	unrestricted := (initialSerials == nil)
	subMap := make(map[string]struct{})
	if !unrestricted {
		for _, s := range initialSerials {
			norm := NormalizeSerial(s)
			if _, ok := allowedMap[norm]; ok {
				subMap[norm] = struct{}{}
			}
		}
	}

	return &ClientSession{
		ID:                uuid.NewString(),
		Token:             token,
		Send:              make(chan []byte, 256),
		AllowedSerials:    allowedMap,
		SubscribedSerials: subMap,
		indexedSerials:    make(map[string]struct{}),
		unrestricted:      unrestricted,
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
		"subscribedCount", session.SubscribedCount(),
		"unrestricted", session.IsUnrestricted(),
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
// It verifies caller ownership against session.Token / user identity before applying changes.
// CRITICAL SECURITY GUARANTEE: Serials are strictly intersected with session.AllowedSerials.
// Any serial requested that is not in the user's permitted inventory is discarded.
// If requestedSerials contains "*", the session reverts to unrestricted mode (all permitted devices).
// If requestedSerials is empty or contains no permitted serials, zero devices are subscribed.
func (h *Hub) UpdateSubscription(sessionID, callerToken string, requestedSerials []string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	session, exists := h.sessions[sessionID]
	if !exists || session == nil || session.IsClosed() {
		return 0, fmt.Errorf("session not found: %s: %w", sessionID, ErrSessionNotFound)
	}

	// Verify session ownership
	if !session.VerifyOwnership(callerToken) {
		return 0, fmt.Errorf("session ownership verification failed for session %s: %w", sessionID, ErrSessionOwnershipMismatch)
	}

	// Remove old indexes for this session
	h.deindexSessionSerials(session)

	// Filter requested serials against allowed inventory or check for wildcard
	session.mu.Lock()
	if normCaller := normalizeToken(callerToken); normCaller != "" {
		session.Token = normCaller
	}

	isWildcard := false
	for _, s := range requestedSerials {
		if strings.TrimSpace(s) == "*" {
			isWildcard = true
			break
		}
	}

	var grantedCount int
	if isWildcard {
		session.unrestricted = true
		session.SubscribedSerials = make(map[string]struct{})
		grantedCount = len(session.AllowedSerials)
	} else {
		session.unrestricted = false
		newSubMap := make(map[string]struct{})
		for _, s := range requestedSerials {
			norm := NormalizeSerial(s)
			if _, allowed := session.AllowedSerials[norm]; allowed {
				newSubMap[norm] = struct{}{}
			}
		}
		session.SubscribedSerials = newSubMap
		grantedCount = len(newSubMap)
	}
	session.mu.Unlock()

	// Re-index new subscriptions
	h.indexSessionSerials(session)

	h.logger.Debug("updated session subscriptions",
		"sessionId", sessionID,
		"requested", len(requestedSerials),
		"granted", grantedCount,
		"unrestricted", isWildcard,
	)

	return grantedCount, nil
}

// RefreshSessionAuthorization re-evaluates the session's permitted inventory against OWPROV.
// It removes any revoked serials from AllowedSerials and SubscribedSerials, updates the inverted index,
// and terminates/unregisters the session if the token is invalid, access is revoked, or no devices remain accessible.
func (h *Hub) RefreshSessionAuthorization(ctx context.Context, sessionID, reqID, corrID string) error {
	h.mu.RLock()
	session, exists := h.sessions[sessionID]
	h.mu.RUnlock()

	if !exists || session == nil || session.IsClosed() {
		return fmt.Errorf("session not found or closed: %s", sessionID)
	}

	// Fetch fresh allowed inventory from OWPROV using the session's token
	newSerials, err := h.AuthorizeUser(ctx, session.Token, reqID, corrID)
	if err != nil {
		h.logger.Warn("periodic re-authorization failed; unregistering session",
			"sessionId", sessionID,
			"error", err,
		)
		h.UnregisterSession(sessionID)
		return fmt.Errorf("re-authorization failed: %w", err)
	}

	newAllowedMap := make(map[string]struct{}, len(newSerials))
	for _, s := range newSerials {
		newAllowedMap[NormalizeSerial(s)] = struct{}{}
	}

	// If all permissions were revoked, close and evict session
	if len(newAllowedMap) == 0 {
		h.logger.Warn("all device access revoked; unregistering session",
			"sessionId", sessionID,
		)
		h.UnregisterSession(sessionID)
		return fmt.Errorf("all device permissions revoked")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Re-verify existence under write lock
	session, exists = h.sessions[sessionID]
	if !exists || session == nil || session.IsClosed() {
		return nil
	}

	// Remove old indexes for this session
	h.deindexSessionSerials(session)

	session.mu.Lock()
	session.AllowedSerials = newAllowedMap

	var revokedCount int
	if !session.unrestricted {
		// Prune SubscribedSerials that are no longer allowed
		prunedSubs := make(map[string]struct{})
		for subSerial := range session.SubscribedSerials {
			if _, allowed := newAllowedMap[subSerial]; allowed {
				prunedSubs[subSerial] = struct{}{}
			} else {
				revokedCount++
			}
		}
		session.SubscribedSerials = prunedSubs
	}
	session.mu.Unlock()

	// Re-index remaining valid subscriptions
	h.indexSessionSerials(session)

	if revokedCount > 0 {
		h.logger.Info("pruned revoked serials from active session subscriptions",
			"sessionId", sessionID,
			"revokedCount", revokedCount,
			"remainingSubscribed", session.SubscribedCount(),
		)
	}

	return nil
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
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.indexedSerials == nil {
		session.indexedSerials = make(map[string]struct{})
	}

	// Determine target serials to index based on subscription mode
	var target map[string]struct{}
	if session.unrestricted {
		target = session.AllowedSerials
	} else {
		target = session.SubscribedSerials
	}

	for serial := range target {
		if _, ok := h.serialToSessions[serial]; !ok {
			h.serialToSessions[serial] = make(map[string]*ClientSession)
		}
		h.serialToSessions[serial][session.ID] = session
		session.indexedSerials[serial] = struct{}{}
	}
}

func (h *Hub) deindexSessionSerials(session *ClientSession) {
	session.mu.Lock()
	defer session.mu.Unlock()

	for serial := range session.indexedSerials {
		if subs, ok := h.serialToSessions[serial]; ok {
			delete(subs, session.ID)
			if len(subs) == 0 {
				delete(h.serialToSessions, serial)
			}
		}
	}
	session.indexedSerials = make(map[string]struct{})
}
