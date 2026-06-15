package models

import (
	"time"
)

// SampleItem is a temporary scaffold placeholder model carried over from the
// starter template. It does not represent the planned MDU domain.
type SampleItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateItemRequest is a temporary scaffold placeholder request model.
type CreateItemRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

// UpdateItemRequest is a temporary scaffold placeholder request model.
type UpdateItemRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// ItemResponse is a temporary scaffold placeholder response model.
type ItemResponse struct {
	Data SampleItem `json:"data"`
}

// ItemListResponse is a temporary scaffold placeholder response model.
type ItemListResponse struct {
	Data []SampleItem `json:"data"`
}
