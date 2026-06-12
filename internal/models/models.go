package models

import (
	"time"
)

// SampleItem represents the domain model and database entity.
type SampleItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateItemRequest defines the incoming payload to create a new item.
type CreateItemRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=255"`
	Description string `json:"description" validate:"max=1000"`
}

// UpdateItemRequest defines the incoming payload to update an existing item.
type UpdateItemRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
}

// ItemResponse wraps a single item inside the standard response envelope.
type ItemResponse struct {
	Data SampleItem `json:"data"`
}

// ItemListResponse wraps multiple items inside the standard response envelope.
type ItemListResponse struct {
	Data []SampleItem `json:"data"`
}
