package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

type FieldValueService interface {
	Set(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.FieldValue, error)
	ListByEntity(ctx context.Context, entityID uuid.UUID, p repository.Pagination) (*repository.Page[models.FieldValue], error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type fieldValueService struct {
	repo repository.FieldValueRepository
}

func NewFieldValueService(repo repository.FieldValueRepository) FieldValueService {
	return &fieldValueService{repo: repo}
}

// Set creates or updates the field value for the given entity+field pair.
func (s *fieldValueService) Set(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
	existing, err := s.repo.GetByEntityAndField(ctx, fv.EntityID, fv.FieldID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		existing.Value = fv.Value
		existing.UpdatedBy = fv.UpdatedBy
		return s.repo.Update(ctx, existing)
	}
	result, err := s.repo.Create(ctx, fv)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return result, nil
}

func (s *fieldValueService) GetByID(ctx context.Context, id uuid.UUID) (*models.FieldValue, error) {
	fv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return fv, nil
}

func (s *fieldValueService) ListByEntity(ctx context.Context, entityID uuid.UUID, p repository.Pagination) (*repository.Page[models.FieldValue], error) {
	return s.repo.ListByEntity(ctx, entityID, p)
}

func (s *fieldValueService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id, deletedBy)
}
