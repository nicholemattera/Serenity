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

// CompositeDetail is a Composite with its Fields eagerly loaded.
type CompositeDetail struct {
	models.Composite
	Fields []models.Field `json:"fields"`
}

type CompositeService interface {
	Create(ctx context.Context, composite *models.Composite) (*models.Composite, error)
	GetByID(ctx context.Context, id uuid.UUID, enrich bool) (*CompositeDetail, error)
	GetBySlug(ctx context.Context, slug string, enrich bool) (*CompositeDetail, error)
	List(ctx context.Context, p *repository.Pagination, enrich bool) (*repository.Page[CompositeDetail], error)
	Update(ctx context.Context, composite *models.Composite, enrich bool) (*CompositeDetail, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type compositeService struct {
	compositeRepo repository.CompositeRepository
	fieldSvc      FieldService
}

func NewCompositeService(compositeRepo repository.CompositeRepository, fieldSvc FieldService) CompositeService {
	return &compositeService{compositeRepo: compositeRepo, fieldSvc: fieldSvc}
}

func (s *compositeService) enrich(ctx context.Context, composite *models.Composite) (*CompositeDetail, error) {
	fields, err := s.fieldSvc.ListByComposite(ctx, composite.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load fields: %w", err)
	}
	detail := &CompositeDetail{Composite: *composite}
	if fields.Data != nil {
		detail.Fields = fields.Data
	} else {
		detail.Fields = []models.Field{}
	}
	return detail, nil
}

func (s *compositeService) Create(ctx context.Context, composite *models.Composite) (*models.Composite, error) {
	composite, err := s.compositeRepo.Create(ctx, composite)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return composite, nil
}

func (s *compositeService) GetByID(ctx context.Context, id uuid.UUID, enrich bool) (*CompositeDetail, error) {
	composite, err := s.compositeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !enrich {
		return &CompositeDetail{Composite: *composite}, nil
	}

	return s.enrich(ctx, composite)
}

func (s *compositeService) GetBySlug(ctx context.Context, slug string, enrich bool) (*CompositeDetail, error) {
	composite, err := s.compositeRepo.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !enrich {
		return &CompositeDetail{Composite: *composite}, nil
	}

	return s.enrich(ctx, composite)
}

func (s *compositeService) enrichList(ctx context.Context, composites []models.Composite) ([]CompositeDetail, error) {
	details := make([]CompositeDetail, len(composites))
	if len(composites) == 0 {
		return details, nil
	}
	ids := make([]uuid.UUID, len(composites))
	for i, c := range composites {
		ids[i] = c.ID
	}
	fieldsByComposite, err := s.fieldSvc.ListByComposites(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load fields: %w", err)
	}
	for i, c := range composites {
		fields := fieldsByComposite[c.ID]
		if fields == nil {
			fields = []models.Field{}
		}
		details[i] = CompositeDetail{Composite: c, Fields: fields}
	}
	return details, nil
}

func (s *compositeService) List(ctx context.Context, p *repository.Pagination, enrich bool) (*repository.Page[CompositeDetail], error) {
	composites, err := s.compositeRepo.List(ctx, p)
	if err != nil {
		return nil, err
	}

	var details []CompositeDetail
	if enrich {
		details, err = s.enrichList(ctx, composites.Data)
		if err != nil {
			return nil, err
		}
	} else {
		details = make([]CompositeDetail, len(composites.Data))
		for i, c := range composites.Data {
			details[i] = CompositeDetail{Composite: c}
		}
	}

	return &repository.Page[CompositeDetail]{Data: details, Total: composites.Total, Limit: composites.Limit, Offset: composites.Offset}, nil
}

func (s *compositeService) Update(ctx context.Context, composite *models.Composite, enrich bool) (*CompositeDetail, error) {
	if _, err := s.compositeRepo.GetByID(ctx, composite.ID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	result, err := s.compositeRepo.Update(ctx, composite)
	if err != nil {
		return nil, err
	}

	if !enrich {
		return &CompositeDetail{Composite: *composite}, nil
	}

	return s.enrich(ctx, result)
}

func (s *compositeService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.compositeRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return s.compositeRepo.Delete(ctx, id, deletedBy)
}
