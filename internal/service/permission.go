package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

type PermissionService interface {
	Create(ctx context.Context, permission *models.Permission) (*models.Permission, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	ListByRole(ctx context.Context, roleID uuid.UUID, p repository.Pagination) (*repository.Page[models.Permission], error)
	Update(ctx context.Context, permission *models.Permission) (*models.Permission, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	// CanRead returns true if the given role (nil = unauthenticated) may read entities in the composite.
	CanRead(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error)
	// CanWrite returns true if the given role (nil = unauthenticated) may write entities in the composite.
	CanWrite(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error)
	// CanReadResource returns true if the given role (nil = unauthenticated) may read the built-in resource type.
	CanReadResource(ctx context.Context, resourceType models.ResourceType, roleID *uuid.UUID) (bool, error)
	// CanWriteResource returns true if the given role (nil = unauthenticated) may write the built-in resource type.
	CanWriteResource(ctx context.Context, resourceType models.ResourceType, roleID *uuid.UUID) (bool, error)
}

type permissionService struct {
	repo repository.PermissionRepository
}

func NewPermissionService(repo repository.PermissionRepository) PermissionService {
	return &permissionService{repo: repo}
}

func (s *permissionService) Create(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	result, err := s.repo.Create(ctx, permission)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *permissionService) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return p, nil
}

func (s *permissionService) ListByRole(ctx context.Context, roleID uuid.UUID, p repository.Pagination) (*repository.Page[models.Permission], error) {
	return s.repo.ListByRole(ctx, roleID, p)
}

func (s *permissionService) Update(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	if _, err := s.GetByID(ctx, permission.ID); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, permission)
}

func (s *permissionService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id, deletedBy)
}

func (s *permissionService) CanRead(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultRead, nil
	}
	p, err := s.repo.GetByRoleAndComposite(ctx, *roleID, composite.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return composite.DefaultRead, nil
		}
		return false, err
	}
	return p.CanRead, nil
}

func (s *permissionService) CanWrite(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultWrite, nil
	}
	p, err := s.repo.GetByRoleAndComposite(ctx, *roleID, composite.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return composite.DefaultWrite, nil
		}
		return false, err
	}
	return p.CanWrite, nil
}

// CanReadResource returns true if the role has been explicitly granted read access to the
// built-in resource type. Unauthenticated users and roles without an explicit grant are denied.
func (s *permissionService) CanReadResource(ctx context.Context, resourceType models.ResourceType, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return false, nil
	}
	p, err := s.repo.GetByRoleAndResource(ctx, *roleID, resourceType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return p.CanRead, nil
}

// CanWriteResource returns true if the role has been explicitly granted write access to the
// built-in resource type. Unauthenticated users and roles without an explicit grant are denied.
func (s *permissionService) CanWriteResource(ctx context.Context, resourceType models.ResourceType, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return false, nil
	}
	p, err := s.repo.GetByRoleAndResource(ctx, *roleID, resourceType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return p.CanWrite, nil
}
