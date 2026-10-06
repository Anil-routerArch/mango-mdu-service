package owprov

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/ra-common-mods/apperror"
)

const (
	userAgent = "mango-mdu-service/1.0"
	pageSize  = 500
)

// Client defines the contract for communicating with OWPROV.
type Client interface {
	GetPolicy(ctx context.Context, id, token, reqID, corrID string) (*models.ManagementPolicy, error)
	GetRolesByPolicy(ctx context.Context, policyID, token, reqID, corrID string) ([]models.ManagementRole, error)
	GetEntities(ctx context.Context, token, reqID, corrID string) ([]models.Entity, error)
	GetVenues(ctx context.Context, token, reqID, corrID string) ([]models.Venue, error)
}

type client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for creating an OWPROV client.
type Config struct {
	BaseURL   string
	Timeout   time.Duration
	TLSConfig *tls.Config
}

// NewClient creates a new OWPROV client instance.
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
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// GetPolicy retrieves a single management policy by ID from OWPROV.
func (c *client) GetPolicy(ctx context.Context, id, token, reqID, corrID string) (*models.ManagementPolicy, error) {
	endpoint := fmt.Sprintf("%s/api/v1/managementPolicy/%s", c.baseURL, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create policy request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", fmt.Sprintf("downstream OWPROV unreachable: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, models.NewApiError(http.StatusNotFound, "Not Found", "Management policy not found")
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("OWPROV returned status %d: %s", resp.StatusCode, string(body)))
	}

	var policy models.ManagementPolicy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse policy response", err)
	}

	return &policy, nil
}

// GetRolesByPolicy fetches all management roles associated with policyID using pagination.
func (c *client) GetRolesByPolicy(ctx context.Context, policyID, token, reqID, corrID string) ([]models.ManagementRole, error) {
	var allRoles []models.ManagementRole
	offset := 0

	for {
		endpoint := fmt.Sprintf("%s/api/v1/managementRole?policyId=%s&limit=%d&offset=%d",
			c.baseURL, url.QueryEscape(policyID), pageSize, offset)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to create roles request", err)
		}

		c.setHeaders(req, token, reqID, corrID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", fmt.Sprintf("downstream OWPROV unreachable: %v", err))
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("OWPROV returned status %d: %s", resp.StatusCode, string(body)))
		}

		var roleResp models.ManagementRoleListResponse
		err = json.NewDecoder(resp.Body).Decode(&roleResp)
		resp.Body.Close()
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse roles response", err)
		}

		allRoles = append(allRoles, roleResp.Roles...)

		if len(roleResp.Roles) < pageSize {
			break
		}
		offset += pageSize
	}

	return allRoles, nil
}

// GetEntities fetches properties/entities from OWPROV.
func (c *client) GetEntities(ctx context.Context, token, reqID, corrID string) ([]models.Entity, error) {
	endpoint := fmt.Sprintf("%s/api/v1/entity", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create entity request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", fmt.Sprintf("downstream OWPROV unreachable: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("OWPROV returned status %d: %s", resp.StatusCode, string(body)))
	}

	var entityResp models.EntityListResponse
	if err := json.NewDecoder(resp.Body).Decode(&entityResp); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse entities response", err)
	}

	return entityResp.Entities, nil
}

// GetVenues fetches venues from OWPROV.
func (c *client) GetVenues(ctx context.Context, token, reqID, corrID string) ([]models.Venue, error) {
	endpoint := fmt.Sprintf("%s/api/v1/venue", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create venue request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", fmt.Sprintf("downstream OWPROV unreachable: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("OWPROV returned status %d: %s", resp.StatusCode, string(body)))
	}

	var venueResp models.VenueListResponse
	if err := json.NewDecoder(resp.Body).Decode(&venueResp); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse venues response", err)
	}

	return venueResp.Venues, nil
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
