package owprov

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/ra-common-mods/apperror"
)

const (
	userAgent        = "mango-mdu-service/1.0"
	pageSize         = 500
	maxPages         = 100
	maxErrorBytes    = 2048
	maxResponseBytes = 10 << 20 // 10 MB
)

// Client defines the contract for communicating with OWPROV.
type Client interface {
	GetPolicy(ctx context.Context, id, token, reqID, corrID string) (*models.ManagementPolicy, error)
	GetRolesByPolicy(ctx context.Context, policyID, token, reqID, corrID string) ([]models.ManagementRole, error)
	GetEntities(ctx context.Context, token, reqID, corrID string) ([]models.Entity, error)
	GetVenues(ctx context.Context, token, reqID, corrID string) ([]models.Venue, error)
	GetInventory(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error)
	GetInventoryCount(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error)
	GetInventorySerials(ctx context.Context, token, reqID, corrID string) ([]string, error)
	GetInventoryItem(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error)
	DeleteInventoryItem(ctx context.Context, serial, token, reqID, corrID string) error
}

type client struct {
	urlResolver func() string
	httpClient  *http.Client
	logger      *slog.Logger
}

// Config holds configuration for creating an OWPROV client.
type Config struct {
	URLResolver func() string
	Timeout     time.Duration
	TLSConfig   *tls.Config
	Logger      *slog.Logger
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

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &client{
		urlResolver: cfg.URLResolver,
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
	return "", models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "owprov service endpoint not discovered or available")
}

// GetPolicy retrieves a single management policy by ID from OWPROV.
func (c *client) GetPolicy(ctx context.Context, id, token, reqID, corrID string) (*models.ManagementPolicy, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/api/v1/managementPolicy/%s", baseURL, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create policy request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV request failed",
				"service", "owprov",
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, models.NewApiError(http.StatusNotFound, "Not Found", "Management policy not found")
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
		_ = body
		if c.logger != nil {
			c.logger.Error("downstream OWPROV returned non-200 status",
				"service", "owprov",
				"status", resp.StatusCode,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
	}

	var policy models.ManagementPolicy
	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	err = json.NewDecoder(limitedReader).Decode(&policy)
	resp.Body.Close()
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse policy response", err)
	}

	return &policy, nil
}

// GetRolesByPolicy fetches all management roles associated with policyID using pagination.
func (c *client) GetRolesByPolicy(ctx context.Context, policyID, token, reqID, corrID string) ([]models.ManagementRole, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	var allRoles []models.ManagementRole
	offset := 0
	pages := 0

	for {
		pages++
		endpoint := fmt.Sprintf("%s/api/v1/managementRole?policyId=%s&limit=%d&offset=%d",
			baseURL, url.QueryEscape(policyID), pageSize, offset)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to create roles request", err)
		}

		c.setHeaders(req, token, reqID, corrID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("downstream OWPROV request failed",
					"service", "owprov",
					"error", err,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
			_ = body
			resp.Body.Close()
			if c.logger != nil {
				c.logger.Error("downstream OWPROV returned non-200 status",
					"service", "owprov",
					"status", resp.StatusCode,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
		}

		var roleResp models.ManagementRoleListResponse
		limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
		err = json.NewDecoder(limitedReader).Decode(&roleResp)
		resp.Body.Close()
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse roles response", err)
		}

		allRoles = append(allRoles, roleResp.Roles...)

		if len(roleResp.Roles) < pageSize || pages >= maxPages {
			if len(roleResp.Roles) == pageSize && pages >= maxPages && c.logger != nil {
				c.logger.Warn("pagination reached maximum allowed pages", "service", "owprov", "endpoint", "managementRole", "max_pages", maxPages, "total_records", len(allRoles))
			}
			break
		}
		offset += pageSize
	}

	return allRoles, nil
}

// GetEntities fetches properties/entities from OWPROV using pagination.
func (c *client) GetEntities(ctx context.Context, token, reqID, corrID string) ([]models.Entity, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	var allEntities []models.Entity
	offset := 0
	pages := 0

	for {
		pages++
		endpoint := fmt.Sprintf("%s/api/v1/entity?limit=%d&offset=%d", baseURL, pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to create entity request", err)
		}

		c.setHeaders(req, token, reqID, corrID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("downstream OWPROV request failed",
					"service", "owprov",
					"error", err,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
			_ = body
			resp.Body.Close()
			if c.logger != nil {
				c.logger.Error("downstream OWPROV returned non-200 status",
					"service", "owprov",
					"status", resp.StatusCode,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
		}

		var entityResp models.EntityListResponse
		limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
		err = json.NewDecoder(limitedReader).Decode(&entityResp)
		resp.Body.Close()
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse entities response", err)
		}

		allEntities = append(allEntities, entityResp.Entities...)

		if len(entityResp.Entities) < pageSize || pages >= maxPages {
			if len(entityResp.Entities) == pageSize && pages >= maxPages && c.logger != nil {
				c.logger.Warn("pagination reached maximum allowed pages", "service", "owprov", "endpoint", "entity", "max_pages", maxPages, "total_records", len(allEntities))
			}
			break
		}
		offset += pageSize
	}

	return allEntities, nil
}

// GetVenues fetches venues from OWPROV using pagination.
func (c *client) GetVenues(ctx context.Context, token, reqID, corrID string) ([]models.Venue, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	var allVenues []models.Venue
	offset := 0
	pages := 0

	for {
		pages++
		endpoint := fmt.Sprintf("%s/api/v1/venue?limit=%d&offset=%d", baseURL, pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to create venue request", err)
		}

		c.setHeaders(req, token, reqID, corrID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("downstream OWPROV request failed",
					"service", "owprov",
					"error", err,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
			_ = body
			resp.Body.Close()
			if c.logger != nil {
				c.logger.Error("downstream OWPROV returned non-200 status",
					"service", "owprov",
					"status", resp.StatusCode,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
		}

		var venueResp models.VenueListResponse
		limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
		err = json.NewDecoder(limitedReader).Decode(&venueResp)
		resp.Body.Close()
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse venues response", err)
		}

		allVenues = append(allVenues, venueResp.Venues...)

		if len(venueResp.Venues) < pageSize || pages >= maxPages {
			if len(venueResp.Venues) == pageSize && pages >= maxPages && c.logger != nil {
				c.logger.Warn("pagination reached maximum allowed pages", "service", "owprov", "endpoint", "venue", "max_pages", maxPages, "total_records", len(allVenues))
			}
			break
		}
		offset += pageSize
	}

	return allVenues, nil
}

// GetInventory retrieves a page of inventory records from OWPROV with extended information.
func (c *client) GetInventory(ctx context.Context, token, reqID, corrID string, limit, offset int, unassigned bool) ([]models.InventoryTag, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	endpoint := fmt.Sprintf("%s/api/v1/inventory?withExtendedInfo=true&limit=%d&offset=%d", baseURL, limit, offset)
	if unassigned {
		endpoint += "&unassigned=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create inventory request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV inventory request failed",
				"service", "owprov",
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV returned non-200 status for inventory",
				"service", "owprov",
				"status", resp.StatusCode,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
	}

	var tagResp models.InventoryTagListResponse
	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(limitedReader).Decode(&tagResp); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse inventory response", err)
	}

	return tagResp.Tags(), nil
}

// GetInventoryCount retrieves the total count of inventory records accessible to the caller.
func (c *client) GetInventoryCount(ctx context.Context, token, reqID, corrID string, unassigned bool) (int, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return 0, err
	}

	endpoint := fmt.Sprintf("%s/api/v1/inventory?countOnly=true", baseURL)
	if unassigned {
		endpoint += "&unassigned=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, apperror.Wrap(apperror.CodeInternal, "failed to create inventory count request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV inventory count request failed",
				"service", "owprov",
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return 0, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return 0, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV returned non-200 status for inventory count",
				"service", "owprov",
				"status", resp.StatusCode,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return 0, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
	}

	var countResp models.CountOnlyResponse
	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(limitedReader).Decode(&countResp); err != nil {
		return 0, apperror.Wrap(apperror.CodeInternal, "failed to parse inventory count response", err)
	}

	return countResp.Count, nil
}

func (c *client) GetInventorySerials(ctx context.Context, token, reqID, corrID string) ([]string, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	var allSerials []string
	offset := 0
	pages := 0

	for {
		pages++
		endpoint := fmt.Sprintf("%s/api/v1/inventory?serialOnly=true&limit=%d&offset=%d", baseURL, pageSize, offset)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to create inventory serials request", err)
		}

		c.setHeaders(req, token, reqID, corrID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("downstream OWPROV inventory serials request failed",
					"service", "owprov",
					"error", err,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(http.StatusServiceUnavailable, "Service Unavailable", "downstream OWPROV service unreachable")
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if c.logger != nil {
				c.logger.Error("downstream OWPROV returned non-200 status for inventory serials",
					"service", "owprov",
					"status", resp.StatusCode,
					"endpoint", req.URL.Path,
					"request_id", reqID,
					"correlation_id", corrID,
				)
			}
			return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
		}

		var serialsResp models.InventorySerialsResponse
		limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
		err = json.NewDecoder(limitedReader).Decode(&serialsResp)
		resp.Body.Close()
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse inventory serials response", err)
		}

		allSerials = append(allSerials, serialsResp.SerialNumbers...)

		if len(serialsResp.SerialNumbers) < pageSize || pages >= maxPages {
			if len(serialsResp.SerialNumbers) == pageSize && pages >= maxPages && c.logger != nil {
				c.logger.Warn("pagination reached maximum allowed pages", "service", "owprov", "endpoint", "inventory", "max_pages", maxPages, "total_records", len(allSerials))
			}
			break
		}
		offset += pageSize
	}

	return allSerials, nil
}

func normalizeSerial(s string) string {
	clean := strings.ToLower(strings.TrimSpace(s))
	clean = strings.ReplaceAll(clean, ":", "")
	clean = strings.ReplaceAll(clean, "-", "")
	return clean
}

// GetInventoryItem retrieves a single inventory record from OWPROV by serial number.
func (c *client) GetInventoryItem(ctx context.Context, serial, token, reqID, corrID string) (*models.InventoryTag, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}

	clean := normalizeSerial(serial)

	endpoint := fmt.Sprintf("%s/api/v1/inventory/%s?withExtendedInfo=true", baseURL, url.PathEscape(clean))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to create inventory item request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV inventory item request failed",
				"service", "owprov",
				"serial", clean,
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(http.StatusBadGateway, "Bad Gateway", "failed to reach downstream owprov service")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil, models.NewApiError(http.StatusForbidden, "Forbidden", "Device is not assigned to user or access denied")
	}

	if resp.StatusCode != http.StatusOK {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV returned non-200 status for inventory item",
				"service", "owprov",
				"serial", clean,
				"status", resp.StatusCode,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return nil, models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
	}

	var tag models.InventoryTag
	limitedReader := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(limitedReader).Decode(&tag); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to parse inventory item response", err)
	}

	return &tag, nil
}

// DeleteInventoryItem removes an inventory item from OWPROV.
func (c *client) DeleteInventoryItem(ctx context.Context, serial, token, reqID, corrID string) error {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return err
	}

	clean := normalizeSerial(serial)
	if clean == "" {
		return models.NewApiError(http.StatusBadRequest, "Bad Request", "serial number is required")
	}

	endpoint := fmt.Sprintf("%s/api/v1/inventory/%s", baseURL, url.PathEscape(clean))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return apperror.Wrap(apperror.CodeInternal, "failed to create inventory delete request", err)
	}

	c.setHeaders(req, token, reqID, corrID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("downstream OWPROV delete inventory item request failed",
				"service", "owprov",
				"serial", clean,
				"error", err,
				"endpoint", req.URL.Path,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}
		return models.NewApiError(http.StatusBadGateway, "Bad Gateway", "failed to reach downstream owprov service")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return models.NewApiError(http.StatusUnauthorized, "Unauthorized", "Invalid or expired token")
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return models.NewApiError(resp.StatusCode, "Downstream Error", fmt.Sprintf("downstream OWPROV returned status %d", resp.StatusCode))
	}

	return nil
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
