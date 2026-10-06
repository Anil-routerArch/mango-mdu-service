package handlers

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/mango-mdu-service/internal/services"
)

// PolicyHandler handles HTTP requests for policy overview.
type PolicyHandler struct {
	svc services.PolicyService
}

// NewPolicyHandler constructs a new PolicyHandler.
func NewPolicyHandler(svc services.PolicyService) *PolicyHandler {
	return &PolicyHandler{svc: svc}
}

// GetOverview handles GET /api/v1/policy/:id/overview.
func (h *PolicyHandler) GetOverview(c fiber.Ctx) error {
	id := c.Params("id")

	token := c.Get("Authorization")
	reqID := c.Get("X-Request-Id")
	corrID := c.Get("X-Correlation-Id")

	overview, err := h.svc.GetPolicyOverview(c.Context(), id, token, reqID, corrID)
	if err != nil {
		var apiErr models.ApiError
		if errors.As(err, &apiErr) {
			return c.Status(apiErr.ErrorCode).JSON(apiErr)
		}

		// Fallback generic 500 error
		return c.Status(http.StatusInternalServerError).JSON(models.NewApiError(
			http.StatusInternalServerError,
			"Internal Server Error",
			err.Error(),
		))
	}

	return c.Status(http.StatusOK).JSON(overview)
}
