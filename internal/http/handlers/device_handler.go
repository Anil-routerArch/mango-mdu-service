package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/mango-mdu-service/internal/services"
)

// DeviceHandler handles HTTP requests for fleet devices.
type DeviceHandler struct {
	svc    services.DeviceService
	logger *slog.Logger
}

// NewDeviceHandler constructs a new DeviceHandler.
func NewDeviceHandler(svc services.DeviceService, logger ...*slog.Logger) *DeviceHandler {
	var l *slog.Logger
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	} else {
		l = slog.Default()
	}
	return &DeviceHandler{svc: svc, logger: l}
}

// GetDevices handles GET /api/v1/devices.
func (h *DeviceHandler) GetDevices(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized,
			"Unauthorized",
			"Authorization header is required",
		))
	}

	pageIndex := 0
	if p := c.Query("pageIndex"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val >= 0 {
			pageIndex = val
		}
	} else if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val >= 0 {
			pageIndex = val
		}
	}

	pageSize := 10
	if ps := c.Query("pageSize"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 500 {
			pageSize = val
		}
	} else if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 500 {
			pageSize = val
		}
	}

	unassignedOnly := false
	if u := c.Query("unassignedOnly"); u != "" {
		unassignedOnly = u == "true" || u == "1"
	} else if u := c.Query("unassigned"); u != "" {
		unassignedOnly = u == "true" || u == "1"
	}

	resp, err := h.svc.GetDevices(c.Context(), token, reqID, corrID, pageIndex, pageSize, unassignedOnly)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}

		if h.logger != nil {
			h.logger.Error("failed to get devices",
				"error", err,
				"request_id", reqID,
				"correlation_id", corrID,
			)
		}

		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError,
			"Internal Server Error",
			"Failed to retrieve devices",
		))
	}

	return c.Status(http.StatusOK).JSON(resp)
}

func extractQueryParams(c fiber.Ctx) url.Values {
	values := url.Values{}
	c.Request().URI().QueryArgs().VisitAll(func(key, val []byte) {
		values.Add(string(key), string(val))
	})
	return values
}

// GetDevice handles GET /api/v1/devices/:serial.
func (h *DeviceHandler) GetDevice(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	data, err := h.svc.GetDevice(c.Context(), serial, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// UpdateDevice handles PUT /api/v1/devices/:serial.
func (h *DeviceHandler) UpdateDevice(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	body := c.Body()
	data, err := h.svc.UpdateDevice(c.Context(), serial, body, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// DeleteDevice handles DELETE /api/v1/devices/:serial.
func (h *DeviceHandler) DeleteDevice(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	if err := h.svc.DeleteDevice(c.Context(), serial, token, reqID, corrID); err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"success": true})
}

// GetDeviceCommands handles GET /api/v1/devices/:serial/commands.
func (h *DeviceHandler) GetDeviceCommands(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	queryParams := extractQueryParams(c)
	data, err := h.svc.GetDeviceCommands(c.Context(), serial, token, reqID, corrID, queryParams)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// DeleteDeviceCommand handles DELETE /api/v1/devices/:serial/commands/:uuid.
func (h *DeviceHandler) DeleteDeviceCommand(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")
	commandUUID := c.Params("commandUUID")
	if commandUUID == "" {
		commandUUID = c.Params("uuid")
	}

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	err := h.svc.DeleteDeviceCommand(c.Context(), serial, commandUUID, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{"success": true})
}

// ExecuteDeviceCommand handles POST /api/v1/devices/:serial/:command.
func (h *DeviceHandler) ExecuteDeviceCommand(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")
	command := c.Params("command")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	body := c.Body()
	data, err := h.svc.ExecuteDeviceCommand(c.Context(), serial, command, body, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// GetDeviceHealthChecks handles GET /api/v1/devices/:serial/healthchecks.
func (h *DeviceHandler) GetDeviceHealthChecks(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	queryParams := extractQueryParams(c)
	data, err := h.svc.GetDeviceHealthChecks(c.Context(), serial, token, reqID, corrID, queryParams)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// GetDeviceLogs handles GET /api/v1/devices/:serial/logs.
func (h *DeviceHandler) GetDeviceLogs(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	queryParams := extractQueryParams(c)
	data, err := h.svc.GetDeviceLogs(c.Context(), serial, token, reqID, corrID, queryParams)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// GetDeviceStatistics handles GET /api/v1/devices/:serial/statistics.
func (h *DeviceHandler) GetDeviceStatistics(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	queryParams := extractQueryParams(c)
	data, err := h.svc.GetDeviceStatistics(c.Context(), serial, token, reqID, corrID, queryParams)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// GetDeviceStatus handles GET /api/v1/devices/:serial/status.
func (h *DeviceHandler) GetDeviceStatus(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	data, err := h.svc.GetDeviceStatus(c.Context(), serial, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// GetDeviceCapabilities handles GET /api/v1/devices/:serial/capabilities.
func (h *DeviceHandler) GetDeviceCapabilities(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	data, err := h.svc.GetDeviceCapabilities(c.Context(), serial, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}

// GetDeviceRtty handles GET /api/v1/devices/:serial/rtty.
func (h *DeviceHandler) GetDeviceRtty(c fiber.Ctx) error {
	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")
	serial := c.Params("serial")

	if token == "" {
		return c.Status(http.StatusUnauthorized).JSON(models.NewApiError(
			http.StatusUnauthorized, "Unauthorized", "Authorization header is required",
		))
	}

	data, err := h.svc.GetDeviceRtty(c.Context(), serial, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError, "Internal Server Error", err.Error(),
		))
	}

	c.Set("Content-Type", "application/json")
	return c.Status(http.StatusOK).Send(data)
}
