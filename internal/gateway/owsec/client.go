package owsec

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
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
	urlResolver func() string
	httpClient  *http.Client
}

// Config holds configuration for creating an OWSEC client.
type Config struct {
	URLResolver func() string
	Timeout     time.Duration
	TLSConfig   *tls.Config
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

	return &client{
		urlResolver: cfg.URLResolver,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
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
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", fmt.Sprintf("downstream OWSEC unreachable: %v", err))
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("OWSEC returned status %d: %s", resp.StatusCode, string(body)))
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
