package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/events"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"golang.org/x/net/websocket"
)

// TokenValidator defines the contract for validating tokens against auth services.
type TokenValidator interface {
	ValidateToken(ctx context.Context, rawToken string) error
}

// DeviceEventsHandler manages real-time device event streaming and client subscriptions.
type DeviceEventsHandler struct {
	hub            *events.Hub
	tokenValidator TokenValidator
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
		logger:         l,
	}
}

// SubscriptionUpdateRequest represents the payload for POST /api/v1/devices/events/subscriptions.
type SubscriptionUpdateRequest struct {
	SessionID string   `json:"sessionId"`
	Serials   []string `json:"serials"`
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
			h.serveWebSocket(ws, c.Get("X-Request-Id"), c.Get("X-Correlation-Id"))
		}),
	}

	fastHandler := fasthttpadaptor.NewFastHTTPHandler(wsServer)
	fastHandler(c.RequestCtx())
	return nil
}

func (h *DeviceEventsHandler) serveWebSocket(ws *websocket.Conn, reqID, corrID string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		_ = websocket.Message.Send(ws, `{"error":"Invalid token format. Closing connection."}`)
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
			_ = websocket.Message.Send(ws, `{"error":"Invalid token. Closing connection."}`)
			return
		}
	}

	// 3. Authorize user with OWPROV to resolve permitted device inventory under RBAC
	authCtx, authCancel := context.WithTimeout(ctx, 10*time.Second)
	allowedSerials, err := h.hub.AuthorizeUser(authCtx, rawToken, reqID, corrID)
	authCancel()
	if err != nil {
		h.logger.Warn("websocket inventory authorization failed with owprov", "error", err)
		_ = websocket.Message.Send(ws, `{"error":"Failed to authorize inventory. Closing connection."}`)
		return
	}

	// Reset read deadline for normal streaming operation
	_ = ws.SetReadDeadline(time.Time{})

	// 4. Create and register client session in Hub
	session := h.hub.CreateSession(rawToken, allowedSerials, nil)
	h.hub.RegisterSession(session)
	defer h.hub.UnregisterSession(session.ID)

	h.logger.Info("websocket client authenticated successfully",
		"sessionId", session.ID,
		"allowedSerialsCount", len(allowedSerials),
	)

	// Send Welcome message (matching OWPROV welcome message convention)
	welcomeMsg, _ := json.Marshal(map[string]interface{}{
		"success":      "Welcome! Bienvenue! Bienvenidos!",
		"sessionId":    session.ID,
		"allowedCount": len(allowedSerials),
	})
	if err := websocket.Message.Send(ws, string(welcomeMsg)); err != nil {
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
						_ = websocket.Message.Send(ws, `{"type":"error","error":"Session token expired or revoked. Terminating stream.","code":4401}`)
						session.Close()
						return
					}
				}
				if err := h.hub.RefreshSessionAuthorization(refreshCtx, session.ID, reqID, corrID); err != nil {
					refreshCancel()
					h.logger.Warn("websocket session authorization refresh failed", "sessionId", session.ID, "error", err)
					_ = websocket.Message.Send(ws, `{"type":"error","error":"Access revoked or session terminated.","code":4403}`)
					session.Close()
					return
				}
				refreshCancel()
			case <-ticker.C:
				if err := websocket.Message.Send(ws, `{"ping":true}`); err != nil {
					return
				}
			case msg, ok := <-session.Send:
				if !ok {
					return
				}
				if err := websocket.Message.Send(ws, string(msg)); err != nil {
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
		if err := json.Unmarshal([]byte(trimmed), &cmd); err == nil && len(cmd.Serials) > 0 {
			granted, _ := h.hub.UpdateSubscription(session.ID, session.Token, cmd.Serials)
			ack, _ := json.Marshal(map[string]interface{}{
				"type":            "subscription_ack",
				"subscribedCount": granted,
			})
			_ = websocket.Message.Send(ws, string(ack))
		}
	}

	cancel()
	<-writerDone
}

// StreamEvents handles GET /api/v1/devices/events.
// It establishes a Server-Sent Events (SSE) connection scoped to the caller's authorized devices.
func (h *DeviceEventsHandler) StreamEvents(c fiber.Ctx) error {
	// Extract token from Authorization header or ?token= query parameter (for standard EventSource)
	token := c.Get("Authorization")
	if token == "" {
		token = c.Query("token")
	}
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized,
			"Unauthorized",
			"authorization token is required via Authorization header or token query parameter",
		))
	}

	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")

	// Authorize user with OWPROV to obtain allowed serials under RBAC
	allowedSerials, err := h.hub.AuthorizeUser(c.Context(), token, reqID, corrID)
	if err != nil {
		h.logger.Warn("user authorization failed for device events stream", "error", err)
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized,
			"Unauthorized",
			"failed to authorize access to device inventory",
		))
	}

	// Extract optional initial viewport serial filter
	var initialSerials []string
	if rawSerials := c.Query("serials"); rawSerials != "" {
		parts := strings.Split(rawSerials, ",")
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				initialSerials = append(initialSerials, s)
			}
		}
	}

	session := h.hub.CreateSession(token, allowedSerials, initialSerials)
	h.hub.RegisterSession(session)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	return c.SendStreamWriter(func(w *bufio.Writer) {
		defer h.hub.UnregisterSession(session.ID)

		// Send initial ready handshake frame
		handshake, _ := json.Marshal(map[string]interface{}{
			"sessionId":       session.ID,
			"allowedCount":    len(session.AllowedSerials),
			"subscribedCount": len(session.SubscribedSerials),
		})
		_, _ = fmt.Fprintf(w, "event: ready\ndata: %s\n\n", handshake)
		if err := w.Flush(); err != nil {
			return
		}

		// Keepalive ticker every 15 seconds
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		// Periodic authorization refresh and token-expiration check every 60 seconds
		authTicker := time.NewTicker(60 * time.Second)
		defer authTicker.Stop()

		for {
			select {
			case <-session.Done():
				return
			case <-authTicker.C:
				refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 10*time.Second)
				if h.tokenValidator != nil {
					if err := h.tokenValidator.ValidateToken(refreshCtx, token); err != nil {
						refreshCancel()
						h.logger.Warn("SSE stream token expired or revoked", "sessionId", session.ID, "error", err)
						_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", `{"error":"Session token expired or revoked. Terminating stream.","code":4401}`)
						_ = w.Flush()
						session.Close()
						return
					}
				}
				if err := h.hub.RefreshSessionAuthorization(refreshCtx, session.ID, reqID, corrID); err != nil {
					refreshCancel()
					h.logger.Warn("SSE stream authorization refresh failed", "sessionId", session.ID, "error", err)
					_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", `{"error":"Access revoked or session terminated.","code":4403}`)
					_ = w.Flush()
					session.Close()
					return
				}
				refreshCancel()
			case <-ticker.C:
				_, _ = fmt.Fprintf(w, ": keepalive %d\n\n", time.Now().Unix())
				if err := w.Flush(); err != nil {
					return
				}
			case msg, ok := <-session.Send:
				if !ok {
					return
				}
				_, _ = fmt.Fprintf(w, "event: device_event\ndata: %s\n\n", msg)
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
}

// UpdateSubscriptions handles POST /api/v1/devices/events/subscriptions.
// Clients update their active viewport serials as they paginate or filter their fleet table.
// It verifies the caller's authorization token and confirms session ownership before applying updates.
func (h *DeviceEventsHandler) UpdateSubscriptions(c fiber.Ctx) error {
	callerToken := c.Get("Authorization")
	if callerToken == "" {
		callerToken = c.Query("token")
	}
	callerToken = strings.TrimPrefix(callerToken, "Bearer ")
	callerToken = strings.TrimPrefix(callerToken, "bearer ")
	callerToken = strings.TrimSpace(callerToken)

	if callerToken == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized,
			"Unauthorized",
			"authorization token is required via Authorization header or token query parameter",
		))
	}

	if h.tokenValidator != nil {
		valCtx, valCancel := context.WithTimeout(c.Context(), 5*time.Second)
		err := h.tokenValidator.ValidateToken(valCtx, callerToken)
		valCancel()
		if err != nil {
			h.logger.Warn("subscription update token validation failed", "error", err)
			return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
				http.StatusUnauthorized,
				"Unauthorized",
				"invalid or expired authorization token",
			))
		}
	}

	var req SubscriptionUpdateRequest
	if err := c.Bind().Body(&req); err != nil || req.SessionID == "" {
		return c.Status(http.StatusBadRequest).JSON(models.NewApiError(
			http.StatusBadRequest,
			"Bad Request",
			"valid sessionId and serials array are required",
		))
	}

	grantedCount, err := h.hub.UpdateSubscription(req.SessionID, callerToken, req.Serials)
	if err != nil {
		if errors.Is(err, events.ErrSessionOwnershipMismatch) {
			h.logger.Warn("unauthorized session subscription update attempt",
				"sessionId", req.SessionID,
				"error", err,
			)
			return c.Status(http.StatusForbidden).JSON(models.NewApiError(
				http.StatusForbidden,
				"Forbidden",
				"caller does not own this session",
			))
		}
		return c.Status(http.StatusNotFound).JSON(models.NewApiError(
			http.StatusNotFound,
			"Not Found",
			err.Error(),
		))
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"sessionId":       req.SessionID,
		"subscribedCount": grantedCount,
	})
}
