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

type RoleService interface {
	Create(ctx context.Context, role *models.Role) (*models.Role, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	List(ctx context.Context, p *repository.Pagination) (*repository.Page[models.Role], error)
	Update(ctx context.Context, role *models.Role) (*models.Role, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type roleService struct {
	repo repository.RoleRepository
}

func NewRoleService(repo repository.RoleRepository) RoleService {
	return &roleService{repo: repo}
}

func (s *roleService) Create(ctx context.Context, role *models.Role) (*models.Role, error) {
	result, err := s.repo.Create(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return result, nil
}

func (s *roleService) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	role, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (s *roleService) List(ctx context.Context, p *repository.Pagination) (*repository.Page[models.Role], error) {
	return s.repo.List(ctx, p)
}

func (s *roleService) Update(ctx context.Context, role *models.Role) (*models.Role, error) {
	if _, err := s.GetByID(ctx, role.ID); err != nil {
		return nil, err
	}
	result, err := s.repo.Update(ctx, role)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *roleService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id, deletedBy)
}
