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

type FieldService interface {
	Create(ctx context.Context, field *models.Field) (*models.Field, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Field, error)
	ListByComposite(ctx context.Context, compositeID uuid.UUID, p repository.Pagination) (*repository.Page[models.Field], error)
	Update(ctx context.Context, field *models.Field) (*models.Field, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type fieldService struct {
	repo repository.FieldRepository
}

func NewFieldService(repo repository.FieldRepository) FieldService {
	return &fieldService{repo: repo}
}

func (s *fieldService) Create(ctx context.Context, field *models.Field) (*models.Field, error) {
	result, err := s.repo.Create(ctx, field)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return result, nil
}

func (s *fieldService) GetByID(ctx context.Context, id uuid.UUID) (*models.Field, error) {
	field, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return field, nil
}

func (s *fieldService) ListByComposite(ctx context.Context, compositeID uuid.UUID, p repository.Pagination) (*repository.Page[models.Field], error) {
	return s.repo.ListByComposite(ctx, compositeID, p)
}

func (s *fieldService) Update(ctx context.Context, field *models.Field) (*models.Field, error) {
	if _, err := s.GetByID(ctx, field.ID); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, field)
}

func (s *fieldService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id, deletedBy)
}
