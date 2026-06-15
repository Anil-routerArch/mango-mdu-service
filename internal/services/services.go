package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/routerarchitects/mango-mdu-service/internal/db"
	"github.com/routerarchitects/mango-mdu-service/internal/models"
	"github.com/routerarchitects/ra-common-mods/apperror"
)

// ItemService defines the contract for item management business logic.
type ItemService interface {
	Create(ctx context.Context, req models.CreateItemRequest) (*models.SampleItem, error)
	GetByID(ctx context.Context, id string) (*models.SampleItem, error)
	List(ctx context.Context) ([]models.SampleItem, error)
	Update(ctx context.Context, id string, req models.UpdateItemRequest) (*models.SampleItem, error)
	Delete(ctx context.Context, id string) error
}

type itemService struct {
	db *db.Database
}

// NewItemService creates a new concrete instance of ItemService.
func NewItemService(database *db.Database) ItemService {
	return &itemService{db: database}
}

func (s *itemService) Create(ctx context.Context, req models.CreateItemRequest) (*models.SampleItem, error) {
	query := `
		INSERT INTO sample_items (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at
	`

	var item models.SampleItem
	err := s.db.Pool.QueryRow(ctx, query, req.Name, req.Description).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to insert sample item into database", err)
	}

	return &item, nil
}

func (s *itemService) GetByID(ctx context.Context, id string) (*models.SampleItem, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM sample_items
		WHERE id = $1
	`

	var item models.SampleItem
	err := s.db.Pool.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.New(apperror.CodeNotFound, fmt.Sprintf("sample item with ID %s not found", id))
		}
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to query database for sample item", err)
	}

	return &item, nil
}

func (s *itemService) List(ctx context.Context) ([]models.SampleItem, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM sample_items
		ORDER BY created_at DESC
	`

	rows, err := s.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to list sample items from database", err)
	}
	defer rows.Close()

	var items []models.SampleItem
	for rows.Next() {
		var item models.SampleItem
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, apperror.Wrap(apperror.CodeInternal, "failed to scan row data for sample item", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "cursor error during list items iteration", err)
	}

	return items, nil
}

func (s *itemService) Update(ctx context.Context, id string, req models.UpdateItemRequest) (*models.SampleItem, error) {
	// First check if the item exists
	item, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Dynamic updates
	name := item.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := item.Description
	if req.Description != nil {
		description = *req.Description
	}

	query := `
		UPDATE sample_items
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING id, name, description, created_at, updated_at
	`

	var updatedItem models.SampleItem
	err = s.db.Pool.QueryRow(ctx, query, name, description, id).Scan(
		&updatedItem.ID,
		&updatedItem.Name,
		&updatedItem.Description,
		&updatedItem.CreatedAt,
		&updatedItem.UpdatedAt,
	)
	if err != nil {
		return nil, apperror.Wrap(apperror.CodeInternal, "failed to update sample item in database", err)
	}

	return &updatedItem, nil
}

func (s *itemService) Delete(ctx context.Context, id string) error {
	// First check if the item exists
	_, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	query := `DELETE FROM sample_items WHERE id = $1`
	_, err = s.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return apperror.Wrap(apperror.CodeInternal, "failed to delete sample item from database", err)
	}

	return nil
}
