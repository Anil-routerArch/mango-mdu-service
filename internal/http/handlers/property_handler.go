package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/mango-mdu-service/internal/services"
)

type PropertyHandler struct {
	svc    services.PropertyService
	logger *slog.Logger
}

func NewPropertyHandler(svc services.PropertyService, logger ...*slog.Logger) *PropertyHandler {
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}
	return &PropertyHandler{svc: svc, logger: l}
}

func propertyRequestMetadata(c fiber.Ctx) (string, string, string) {
	return c.Get("Authorization"), c.Get("X-Request-Id"), c.Get("X-Correlation-Id")
}

func (h *PropertyHandler) respond(c fiber.Ctx, data any, err error) error {
	if err == nil {
		return c.Status(http.StatusOK).JSON(data)
	}
	var apiErr models.ApiError
	if errors.As(err, &apiErr) {
		return c.Status(apiErr.ErrorCode).JSON(apiErr)
	}
	h.logger.Error("property request failed", "error", err, "request_id", c.Get("X-Request-Id"), "correlation_id", c.Get("X-Correlation-Id"))
	return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(http.StatusInternalServerError, "Internal Server Error", "Failed to retrieve property data"))
}

func (h *PropertyHandler) GetTree(c fiber.Ctx) error {
	token, reqID, corrID := propertyRequestMetadata(c)
	data, err := h.svc.GetTree(c.Context(), token, reqID, corrID)
	return h.respond(c, data, err)
}

func (h *PropertyHandler) GetMap(c fiber.Ctx) error {
	token, reqID, corrID := propertyRequestMetadata(c)
	data, err := h.svc.GetMap(c.Context(), token, reqID, corrID)
	return h.respond(c, data, err)
}

func (h *PropertyHandler) GetPortfolio(c fiber.Ctx) error {
	token, reqID, corrID := propertyRequestMetadata(c)
	data, err := h.svc.GetPortfolio(c.Context(), token, reqID, corrID)
	return h.respond(c, data, err)
}

func (h *PropertyHandler) getOverview(c fiber.Ctx, scopeType string) error {
	token, reqID, corrID := propertyRequestMetadata(c)
	data, err := h.svc.GetOverview(c.Context(), scopeType, c.Params("id"), token, reqID, corrID)
	return h.respond(c, data, err)
}

func (h *PropertyHandler) GetPropertyOverview(c fiber.Ctx) error { return h.getOverview(c, "property") }
func (h *PropertyHandler) GetVenueOverview(c fiber.Ctx) error    { return h.getOverview(c, "venue") }

func intQuery(c fiber.Ctx, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}

func (h *PropertyHandler) getDevices(c fiber.Ctx, scopeType string) error {
	token, reqID, corrID := propertyRequestMetadata(c)
	query := models.ScopedDevicesQuery{
		PageIndex: intQuery(c, "pageIndex", 0), PageSize: intQuery(c, "pageSize", 25),
		Search: c.Query("search"), VenueID: c.Query("venue"), Availability: c.Query("availability"),
		DeviceType: c.Query("type"), Attention: c.Query("attention"),
	}
	data, err := h.svc.GetDevices(c.Context(), scopeType, c.Params("id"), token, reqID, corrID, query)
	return h.respond(c, data, err)
}

func (h *PropertyHandler) GetPropertyDevices(c fiber.Ctx) error { return h.getDevices(c, "property") }
func (h *PropertyHandler) GetVenueDevices(c fiber.Ctx) error    { return h.getDevices(c, "venue") }
