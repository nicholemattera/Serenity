package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

const defaultPermissionCacheTTL = 45 * time.Second

type PermissionService interface {
	Create(ctx context.Context, permission *models.Permission) (*models.Permission, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	ListByRole(ctx context.Context, roleID uuid.UUID, p *repository.Pagination) (*repository.Page[models.Permission], error)
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

type permissionCacheEntry struct {
	permission *models.Permission
	noRows     bool
	expiresAt  time.Time
}

type permissionService struct {
	repo           repository.PermissionRepository
	mu             sync.RWMutex
	compositeCache map[string]*permissionCacheEntry
	resourceCache  map[string]*permissionCacheEntry
	ttl            time.Duration
}

func NewPermissionService(repo repository.PermissionRepository) PermissionService {
	return &permissionService{
		repo:           repo,
		compositeCache: make(map[string]*permissionCacheEntry),
		resourceCache:  make(map[string]*permissionCacheEntry),
		ttl:            defaultPermissionCacheTTL,
	}
}

func (s *permissionService) cachedByRoleAndComposite(ctx context.Context, roleID, compositeID uuid.UUID) (*models.Permission, error) {
	key := roleID.String() + ":" + compositeID.String()

	s.mu.RLock()
	entry, ok := s.compositeCache[key]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		if entry.noRows {
			return nil, pgx.ErrNoRows
		}
		return entry.permission, nil
	}

	p, err := s.repo.GetByRoleAndComposite(ctx, roleID, compositeID)

	s.mu.Lock()
	if errors.Is(err, pgx.ErrNoRows) {
		s.compositeCache[key] = &permissionCacheEntry{noRows: true, expiresAt: time.Now().Add(s.ttl)}
	} else if err == nil {
		s.compositeCache[key] = &permissionCacheEntry{permission: p, expiresAt: time.Now().Add(s.ttl)}
	}
	s.mu.Unlock()

	return p, err
}

func (s *permissionService) cachedByRoleAndResource(ctx context.Context, roleID uuid.UUID, resourceType models.ResourceType) (*models.Permission, error) {
	key := roleID.String() + ":" + string(resourceType)

	s.mu.RLock()
	entry, ok := s.resourceCache[key]
	s.mu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		if entry.noRows {
			return nil, pgx.ErrNoRows
		}
		return entry.permission, nil
	}

	p, err := s.repo.GetByRoleAndResource(ctx, roleID, resourceType)

	s.mu.Lock()
	if errors.Is(err, pgx.ErrNoRows) {
		s.resourceCache[key] = &permissionCacheEntry{noRows: true, expiresAt: time.Now().Add(s.ttl)}
	} else if err == nil {
		s.resourceCache[key] = &permissionCacheEntry{permission: p, expiresAt: time.Now().Add(s.ttl)}
	}
	s.mu.Unlock()

	return p, err
}

func (s *permissionService) evict(p *models.Permission) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.CompositeID != nil {
		delete(s.compositeCache, p.RoleID.String()+":"+p.CompositeID.String())
	}
	if p.ResourceType != nil {
		delete(s.resourceCache, p.RoleID.String()+":"+string(*p.ResourceType))
	}
}

func (s *permissionService) Create(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	result, err := s.repo.Create(ctx, permission)
	if err != nil {
		return nil, err
	}
	s.evict(result)
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

func (s *permissionService) ListByRole(ctx context.Context, roleID uuid.UUID, p *repository.Pagination) (*repository.Page[models.Permission], error) {
	return s.repo.ListByRole(ctx, roleID, p)
}

func (s *permissionService) Update(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	existing, err := s.GetByID(ctx, permission.ID)
	if err != nil {
		return nil, err
	}
	result, err := s.repo.Update(ctx, permission)
	if err != nil {
		return nil, err
	}
	s.evict(existing)
	return result, nil
}

func (s *permissionService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	existing, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id, deletedBy); err != nil {
		return err
	}
	s.evict(existing)
	return nil
}

func (s *permissionService) CanRead(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultRead, nil
	}
	p, err := s.cachedByRoleAndComposite(ctx, *roleID, composite.ID)
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
	p, err := s.cachedByRoleAndComposite(ctx, *roleID, composite.ID)
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
	p, err := s.cachedByRoleAndResource(ctx, *roleID, resourceType)
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
	p, err := s.cachedByRoleAndResource(ctx, *roleID, resourceType)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return p.CanWrite, nil
}
