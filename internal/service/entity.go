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

// EntityDetail is an Entity with its FieldValues eagerly loaded.
type EntityDetail struct {
	models.Entity
	FieldValues []models.FieldValue `json:"field_values"`
}

type EntityService interface {
	Create(ctx context.Context, entity *models.Entity, parentID *uuid.UUID, afterID *uuid.UUID) (*models.Entity, error)
	GetByID(ctx context.Context, id uuid.UUID, enrich bool) (*EntityDetail, error)
	GetBySlug(ctx context.Context, compositeID uuid.UUID, slug string, enrich bool) (*EntityDetail, error)
	ListByComposite(ctx context.Context, compositeID uuid.UUID, p *repository.Pagination, enrich bool) (*repository.Page[EntityDetail], error)
	ListChildren(ctx context.Context, parentID uuid.UUID, p *repository.Pagination, enrich bool) (*repository.Page[EntityDetail], error)
	Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, afterID *uuid.UUID) error
	MoveRoot(ctx context.Context, id uuid.UUID, afterID *uuid.UUID) error
	Update(ctx context.Context, entity *models.Entity, enrich bool) (*EntityDetail, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type entityService struct {
	entityRepo    repository.EntityRepository
	fieldValueSvc FieldValueService
}

func NewEntityService(entityRepo repository.EntityRepository, fieldValueSvc FieldValueService) EntityService {
	return &entityService{entityRepo: entityRepo, fieldValueSvc: fieldValueSvc}
}

func (s *entityService) enrich(ctx context.Context, entity *models.Entity) (*EntityDetail, error) {
	fvs, err := s.fieldValueSvc.ListByEntity(ctx, entity.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load field values: %w", err)
	}
	detail := &EntityDetail{Entity: *entity}
	if fvs.Data != nil {
		detail.FieldValues = fvs.Data
	} else {
		detail.FieldValues = []models.FieldValue{}
	}
	return detail, nil
}

func (s *entityService) Create(ctx context.Context, entity *models.Entity, parentID *uuid.UUID, afterID *uuid.UUID) (*models.Entity, error) {
	result, err := s.entityRepo.Create(ctx, entity, parentID, afterID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return result, nil
}

func (s *entityService) GetByID(ctx context.Context, id uuid.UUID, enrich bool) (*EntityDetail, error) {
	entity, err := s.entityRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !enrich {
		return &EntityDetail{Entity: *entity}, nil
	}

	return s.enrich(ctx, entity)
}

func (s *entityService) GetBySlug(ctx context.Context, compositeID uuid.UUID, slug string, enrich bool) (*EntityDetail, error) {
	entity, err := s.entityRepo.GetBySlug(ctx, compositeID, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !enrich {
		return &EntityDetail{Entity: *entity}, nil
	}

	return s.enrich(ctx, entity)
}

func (s *entityService) ListByComposite(ctx context.Context, compositeID uuid.UUID, p *repository.Pagination, enrich bool) (*repository.Page[EntityDetail], error) {
	entities, err := s.entityRepo.ListByComposite(ctx, compositeID, p)
	if err != nil {
		return nil, err
	}

	details := make([]EntityDetail, len(entities.Data))
	for i, e := range entities.Data {
		if !enrich {
			details[i] = EntityDetail{Entity: e}
			continue
		}

		detail, err := s.enrich(ctx, &e)
		if err != nil {
			details[i] = EntityDetail{Entity: e}
			continue
		}

		details[i] = *detail
	}

	return &repository.Page[EntityDetail]{Data: details, Total: entities.Total, Limit: entities.Limit, Offset: entities.Offset}, nil
}

func (s *entityService) ListChildren(ctx context.Context, parentID uuid.UUID, p *repository.Pagination, enrich bool) (*repository.Page[EntityDetail], error) {
	entities, err := s.entityRepo.ListChildren(ctx, parentID, p)
	if err != nil {
		return nil, err
	}

	details := make([]EntityDetail, len(entities.Data))
	for i, e := range entities.Data {
		if !enrich {
			details[i] = EntityDetail{Entity: e}
			continue
		}

		detail, err := s.enrich(ctx, &e)
		if err != nil {
			details[i] = EntityDetail{Entity: e}
			continue
		}

		details[i] = *detail
	}

	return &repository.Page[EntityDetail]{Data: details, Total: entities.Total, Limit: entities.Limit, Offset: entities.Offset}, nil
}

func (s *entityService) Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, afterID *uuid.UUID) error {
	if _, err := s.GetByID(ctx, id, false); err != nil {
		return err
	}
	return s.entityRepo.Move(ctx, id, parentID, afterID)
}

func (s *entityService) MoveRoot(ctx context.Context, id uuid.UUID, afterID *uuid.UUID) error {
	if _, err := s.GetByID(ctx, id, false); err != nil {
		return err
	}
	return s.entityRepo.MoveRoot(ctx, id, afterID)
}

func (s *entityService) Update(ctx context.Context, entity *models.Entity, enrich bool) (*EntityDetail, error) {
	if _, err := s.entityRepo.GetByID(ctx, entity.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result, err := s.entityRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	if !enrich {
		return &EntityDetail{Entity: *result}, nil
	}

	return s.enrich(ctx, result)
}

func (s *entityService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.GetByID(ctx, id, false); err != nil {
		return err
	}
	return s.entityRepo.Delete(ctx, id, deletedBy)
}
