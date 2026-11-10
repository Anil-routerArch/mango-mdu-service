package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/events"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"golang.org/x/net/websocket"
)

const (
	// defaultWSWriteTimeout defines the maximum duration allowed for an outbound WebSocket write before timing out.
	defaultWSWriteTimeout = 10 * time.Second
)

// TokenValidator defines the contract for validating tokens against auth services.
type TokenValidator interface {
	ValidateToken(ctx context.Context, rawToken string) error
}

// DeviceEventsHandler manages real-time device event streaming and client subscriptions.
type DeviceEventsHandler struct {
	hub            *events.Hub
	tokenValidator TokenValidator
	writeTimeout   time.Duration
	logger         *slog.Logger
}

// NewDeviceEventsHandler creates a new DeviceEventsHandler instance.
func NewDeviceEventsHandler(hub *events.Hub, tokenValidator TokenValidator, logger ...*slog.Logger) *DeviceEventsHandler {
	var l *slog.Logger
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	} else {
		l = slog.Default()
	}
	return &DeviceEventsHandler{
		hub:            hub,
		tokenValidator: tokenValidator,
		writeTimeout:   defaultWSWriteTimeout,
		logger:         l,
	}
}

// SetWriteTimeout configures the timeout for outbound WebSocket writes.
func (h *DeviceEventsHandler) SetWriteTimeout(timeout time.Duration) {
	if timeout > 0 {
		h.writeTimeout = timeout
	}
}

// WriteTimeout returns the configured outbound WebSocket write timeout.
func (h *DeviceEventsHandler) WriteTimeout() time.Duration {
	if h.writeTimeout <= 0 {
		return defaultWSWriteTimeout
	}
	return h.writeTimeout
}

// wsWriter provides thread-safe writing to a WebSocket connection with a strict write deadline.
type wsWriter struct {
	ws      *websocket.Conn
	timeout time.Duration
	mu      sync.Mutex
}

func newWSWriter(ws *websocket.Conn, timeout time.Duration) *wsWriter {
	if timeout <= 0 {
		timeout = defaultWSWriteTimeout
	}
	return &wsWriter{
		ws:      ws,
		timeout: timeout,
	}
}

// Write sets the write deadline on the underlying connection and sends the message frame.
func (w *wsWriter) Write(msg string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ws.SetWriteDeadline(time.Now().Add(w.timeout)); err != nil {
		return err
	}
	return websocket.Message.Send(w.ws, msg)
}

// HandleWebSocket handles GET /api/v1/ws and /ws (matching OpenWiFi production pattern).
// Upgrades connection without upfront HTTP auth headers, then requires first-frame authentication ("token:<token>").
func (h *DeviceEventsHandler) HandleWebSocket(c fiber.Ctx) error {
	wsServer := websocket.Server{
		Handshake: func(cfg *websocket.Config, req *http.Request) error {
			// Accept cross-origin WebSocket connections matching public CORS policy
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			defer ws.Close()
			h.ServeWebSocket(ws, c.Get("X-Request-Id"), c.Get("X-Correlation-Id"))
		}),
	}

	fastHandler := fasthttpadaptor.NewFastHTTPHandler(wsServer)
	fastHandler(c.RequestCtx())
	return nil
}

// ServeWebSocket serves an upgraded WebSocket connection through its complete lifecycle:
// in-band token authentication, RBAC authorization, event streaming, write deadlines, and viewport subscription.
func (h *DeviceEventsHandler) ServeWebSocket(ws *websocket.Conn, reqID, corrID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	writer := newWSWriter(ws, h.WriteTimeout())

	// 1. First frame must be "token:<token>" within 10 seconds
	_ = ws.SetReadDeadline(time.Now().Add(10 * time.Second))
	var authFrame string
	if err := websocket.Message.Receive(ws, &authFrame); err != nil {
		h.logger.Debug("websocket closed before authentication", "error", err)
		return
	}

	tokens := strings.SplitN(authFrame, ":", 2)
	if len(tokens) != 2 || strings.TrimSpace(tokens[0]) != "token" || strings.TrimSpace(tokens[1]) == "" {
		h.logger.Warn("invalid websocket auth frame format")
		_ = writer.Write(`{"error":"Invalid token format. Closing connection."}`)
		return
	}
	rawToken := strings.TrimSpace(tokens[1])
	rawToken = strings.TrimPrefix(rawToken, "Bearer ")
	rawToken = strings.TrimPrefix(rawToken, "bearer ")
	rawToken = strings.TrimSpace(rawToken)

	// 2. Validate token with OWSEC if validator configured
	if h.tokenValidator != nil {
		valCtx, valCancel := context.WithTimeout(ctx, 5*time.Second)
		err := h.tokenValidator.ValidateToken(valCtx, rawToken)
		valCancel()
		if err != nil {
			h.logger.Warn("websocket token validation failed with owsec", "error", err)
			_ = writer.Write(`{"error":"Invalid token. Closing connection."}`)
			return
		}
	}

	// 3. Authorize user with OWPROV to resolve permitted device inventory under RBAC
	authCtx, authCancel := context.WithTimeout(ctx, 10*time.Second)
	allowedSerials, err := h.hub.AuthorizeUser(authCtx, rawToken, reqID, corrID)
	authCancel()
	if err != nil {
		h.logger.Warn("websocket inventory authorization failed with owprov", "error", err)
		_ = writer.Write(`{"error":"Failed to authorize inventory. Closing connection."}`)
		return
	}

	// Reset read deadline for normal streaming operation
	_ = ws.SetReadDeadline(time.Time{})

	// 4. Create and register client session in Hub
	session := h.hub.CreateSession(rawToken, allowedSerials, nil)
	h.hub.RegisterSession(session)
	defer func() {
		session.Close()
		_ = ws.Close()
		h.hub.UnregisterSession(session.ID)
	}()

	h.logger.Info("websocket client authenticated successfully",
		"sessionId", session.ID,
		"allowedSerialsCount", len(allowedSerials),
	)

	// Send Welcome message (matching OWPROV welcome message convention)
	welcomeMsg, _ := json.Marshal(map[string]interface{}{
		"success":         "Welcome! Bienvenue! Bienvenidos!",
		"sessionId":       session.ID,
		"allowedCount":    len(allowedSerials),
		"subscribedCount": session.SubscribedCount(),
	})
	if err := writer.Write(string(welcomeMsg)); err != nil {
		h.logger.Warn("failed to send websocket welcome message", "sessionId", session.ID, "error", err)
		return
	}

	// 5. Goroutine for sending outbound device events from Hub to WebSocket with periodic auth re-validation
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		authTicker := time.NewTicker(60 * time.Second)
		defer authTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-session.Done():
				return
			case <-authTicker.C:
				refreshCtx, refreshCancel := context.WithTimeout(ctx, 10*time.Second)
				if h.tokenValidator != nil {
					if err := h.tokenValidator.ValidateToken(refreshCtx, rawToken); err != nil {
						refreshCancel()
						h.logger.Warn("websocket session token expired or revoked", "sessionId", session.ID, "error", err)
						_ = writer.Write(`{"type":"error","error":"Session token expired or revoked. Terminating stream.","code":4401}`)
						session.Close()
						_ = ws.Close()
						return
					}
				}
				if err := h.hub.RefreshSessionAuthorization(refreshCtx, session.ID, reqID, corrID); err != nil {
					refreshCancel()
					h.logger.Warn("websocket session authorization refresh failed", "sessionId", session.ID, "error", err)
					_ = writer.Write(`{"type":"error","error":"Access revoked or session terminated.","code":4403}`)
					session.Close()
					_ = ws.Close()
					return
				}
				refreshCancel()
			case <-ticker.C:
				if err := writer.Write(`{"ping":true}`); err != nil {
					h.logger.Warn("websocket ping write failed or timed out", "sessionId", session.ID, "error", err)
					session.Close()
					_ = ws.Close()
					return
				}
			case msg, ok := <-session.Send:
				if !ok {
					return
				}
				if err := writer.Write(string(msg)); err != nil {
					h.logger.Warn("websocket event write failed or timed out", "sessionId", session.ID, "error", err)
					session.Close()
					_ = ws.Close()
					return
				}
			}
		}
	}()

	// 6. Inbound loop for receiving client commands (e.g. viewport serial updates)
	for {
		var incoming string
		if err := websocket.Message.Receive(ws, &incoming); err != nil {
			break
		}

		trimmed := strings.TrimSpace(incoming)
		if trimmed == "" || trimmed == "pong" {
			continue
		}

		var cmd struct {
			Serials []string `json:"serials"`
		}
		if err := json.Unmarshal([]byte(trimmed), &cmd); err == nil && cmd.Serials != nil {
			granted, _ := h.hub.UpdateSubscription(session.ID, session.Token, cmd.Serials)
			ack, _ := json.Marshal(map[string]interface{}{
				"type":            "subscription_ack",
				"subscribedCount": granted,
			})
			if err := writer.Write(string(ack)); err != nil {
				h.logger.Warn("websocket subscription ack write failed or timed out", "sessionId", session.ID, "error", err)
				session.Close()
				_ = ws.Close()
				break
			}
		}
	}

	cancel()
	session.Close()
	_ = ws.Close()
	<-writerDone
}
