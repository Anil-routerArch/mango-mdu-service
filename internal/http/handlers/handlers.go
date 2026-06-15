package handlers

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/mango-mdu-service/internal/services"
	"github.com/routerarchitects/ra-common-mods/apperror"
)

var validate = validator.New()

// ItemHandler is temporary scaffold placeholder HTTP code kept only to preserve
// a runnable starter service until real MDU handlers replace the sample item CRUD.
type ItemHandler struct {
	svc services.ItemService
}

func NewItemHandler(service services.ItemService) *ItemHandler {
	return &ItemHandler{svc: service}
}

// CreateItem handles POST /api/v1/items.
// Temporary scaffold placeholder endpoint only.
func (h *ItemHandler) CreateItem(c fiber.Ctx) error {
	var req models.CreateItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return HandleError(c, apperror.Wrap(apperror.CodeInvalidInput, "invalid request JSON body", err))
	}

	if err := validate.StructCtx(c.Context(), &req); err != nil {
		return HandleError(c, apperror.Wrap(apperror.CodeInvalidInput, err.Error(), err))
	}

	item, err := h.svc.Create(c.Context(), req)
	if err != nil {
		return HandleError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(models.ItemResponse{Data: *item})
}

// GetItem handles GET /api/v1/items/:id.
// Temporary scaffold placeholder endpoint only.
func (h *ItemHandler) GetItem(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return HandleError(c, apperror.New(apperror.CodeInvalidInput, "missing item ID parameter"))
	}

	item, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return HandleError(c, err)
	}

	return c.JSON(models.ItemResponse{Data: *item})
}

// ListItems handles GET /api/v1/items.
// Temporary scaffold placeholder endpoint only.
func (h *ItemHandler) ListItems(c fiber.Ctx) error {
	items, err := h.svc.List(c.Context())
	if err != nil {
		return HandleError(c, err)
	}

	// Guarantee return of empty list rather than null in JSON
	if items == nil {
		items = []models.SampleItem{}
	}

	return c.JSON(models.ItemListResponse{Data: items})
}

// UpdateItem handles PUT /api/v1/items/:id.
// Temporary scaffold placeholder endpoint only.
func (h *ItemHandler) UpdateItem(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return HandleError(c, apperror.New(apperror.CodeInvalidInput, "missing item ID parameter"))
	}

	var req models.UpdateItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return HandleError(c, apperror.Wrap(apperror.CodeInvalidInput, "invalid request JSON body", err))
	}

	if err := validate.StructCtx(c.Context(), &req); err != nil {
		return HandleError(c, apperror.Wrap(apperror.CodeInvalidInput, err.Error(), err))
	}

	item, err := h.svc.Update(c.Context(), id, req)
	if err != nil {
		return HandleError(c, err)
	}

	return c.JSON(models.ItemResponse{Data: *item})
}

// DeleteItem handles DELETE /api/v1/items/:id.
// Temporary scaffold placeholder endpoint only.
func (h *ItemHandler) DeleteItem(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return HandleError(c, apperror.New(apperror.CodeInvalidInput, "missing item ID parameter"))
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		return HandleError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
}

// HandleError parses custom application errors into semantic HTTP responses.
func HandleError(c fiber.Ctx, err error) error {
	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		status := http.StatusInternalServerError

		switch appErr.Code() {
		case apperror.CodeNotFound:
			status = http.StatusNotFound
		case apperror.CodeInvalidInput:
			status = http.StatusBadRequest
		case apperror.CodeUnauthorized:
			status = http.StatusUnauthorized
		case apperror.CodeForbidden:
			status = http.StatusForbidden
		case apperror.CodeInternal:
			status = http.StatusInternalServerError
		}

		return c.Status(status).JSON(fiber.Map{
			"error":   appErr.Message(),
			"code":    appErr.Code(),
			"details": appErr.Error(),
		})
	}

	// Fallback for raw Go errors
	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
		"error": "an unexpected internal error occurred",
		"code":  apperror.CodeInternal,
	})
}
