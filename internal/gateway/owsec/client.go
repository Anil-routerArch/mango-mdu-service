package owsec

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/ra-common-mods/apperror"
)

const (
	userAgent = "mango-mdu-service/1.0"
	pageSize  = 500
)

// Client defines the contract for communicating with OWSEC.
type Client interface {
	GetUsers(ctx context.Context, token, reqID, corrID string) ([]models.SecUser, error)
}

type client struct {
	urlResolver  func() string
	keyResolver  func() string
	internalName string
	internalKey  string
	httpClient   *http.Client
	logger       *slog.Logger
}

// Config holds configuration for creating an OWSEC client.
type Config struct {
	URLResolver  func() string
	KeyResolver  func() string
	InternalName string
	InternalKey  string
	Timeout      time.Duration
	TLSConfig    *tls.Config
	Logger       *slog.Logger
}

// NewClient creates a new OWSEC client instance.
func NewClient(cfg Config) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.TLSConfig != nil {
		transport.TLSClientConfig = cfg.TLSConfig
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &client{
		urlResolver:  cfg.URLResolver,
		keyResolver:  cfg.KeyResolver,
		internalName: cfg.InternalName,
		internalKey:  cfg.InternalKey,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		logger: logger,
	}
}

func (c *client) getBaseURL() (string, error) {
	if c.urlResolver != nil {
		if resolved := c.urlResolver(); resolved != "" {
			return strings.TrimRight(resolved, "/"), nil
		}
	}
	return "", models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owsec service endpoint not discovered or available")
}

// GetUsers retrieves all accessible users from OWSEC using pagination.
func (c *client) GetUsers(ctx context.Context, token, reqID, corrID string) ([]models.SecUser, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	var allUsers []models.SecUser
	offset := 0

	for {
		endpoint := fmt.Sprintf("%s/api/v1/users?limit=%d&offset=%d", baseURL, pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to create users request", err)
		}

		c.setHeaders(req, token, reqID, corrID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("downstream OWSEC request failed",
					"error", err,
					"path", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWSEC service unreachable")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if c.logger != nil {
				c.logger.Error("downstream OWSEC returned non-200 status",
					"status", resp.StatusCode,
					"body", string(body),
					"path", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWSEC returned status %d", resp.StatusCode))
		}

		var usersResp models.SecUserListResponse
		err = json.NewDecoder(resp.Body).Decode(&usersResp)
		resp.Body.Close()
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse users response", err)
		}

		allUsers = append(allUsers, usersResp.Users...)

		if len(usersResp.Users) < pageSize {
			break
		}
		offset += pageSize
	}

	return allUsers, nil
}

func (c *client) setHeaders(req *http.Request, token, reqID, corrID string) {
	if c.internalName != "" {
		req.Header.Set("X-INTERNAL-NAME", c.internalName)
	}
	var key string
	if c.keyResolver != nil {
		key = c.keyResolver()
	} else {
		key = c.internalKey
	}
	if key != "" {
		req.Header.Set("X-API-KEY", key)
	}
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			req.Header.Set("Authorization", "Bearer "+token)
		} else {
			req.Header.Set("Authorization", token)
		}
	}
	req.Header.Set("User-Agent", userAgent)
	if reqID != "" {
		req.Header.Set("X-Request-Id", reqID)
	}
	if corrID != "" {
		req.Header.Set("X-Correlation-Id", corrID)
	}
}
