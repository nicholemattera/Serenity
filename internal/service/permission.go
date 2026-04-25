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
	// CanReadEntity checks the entity resource-type permission first; falls through to composite-level if absent.
	CanReadEntity(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error)
	// CanWriteEntity checks the entity resource-type permission first; falls through to composite-level if absent.
	CanWriteEntity(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error)
	// CanReadFieldValue checks the field_value resource-type permission first; falls through to composite-level if absent.
	CanReadFieldValue(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error)
	// CanWriteFieldValue checks the field_value resource-type permission first; falls through to composite-level if absent.
	CanWriteFieldValue(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error)
}

type permissionCacheEntry struct {
	permission *models.Permission
	noRows     bool
	expiresAt  time.Time
}

type permissionService struct {
	repo           repository.PermissionRepository
	compositeMu    sync.RWMutex
	compositeCache map[string]*permissionCacheEntry
	resourceMu     sync.RWMutex
	resourceCache  map[string]*permissionCacheEntry
	ttl            time.Duration
	maxCacheSize   int
}

func NewPermissionService(repo repository.PermissionRepository, ttl time.Duration, maxCacheSize int) PermissionService {
	return &permissionService{
		repo:           repo,
		compositeCache: make(map[string]*permissionCacheEntry),
		resourceCache:  make(map[string]*permissionCacheEntry),
		ttl:            ttl,
		maxCacheSize:   maxCacheSize,
	}
}

// pruneCache must be called with the write lock held.
// It sweeps expired entries first; if the cache is still at capacity, it evicts one arbitrary entry.
func pruneCache(cache map[string]*permissionCacheEntry, maxSize int) {
	if len(cache) < maxSize {
		return
	}
	now := time.Now()
	for k, v := range cache {
		if now.After(v.expiresAt) {
			delete(cache, k)
		}
	}
	if len(cache) >= maxSize {
		for k := range cache {
			delete(cache, k)
			break
		}
	}
}

func (s *permissionService) cachedByRoleAndComposite(ctx context.Context, roleID, compositeID uuid.UUID) (*models.Permission, error) {
	key := roleID.String() + ":" + compositeID.String()

	s.compositeMu.RLock()
	entry, ok := s.compositeCache[key]
	s.compositeMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		if entry.noRows {
			return nil, pgx.ErrNoRows
		}
		return entry.permission, nil
	}

	p, err := s.repo.GetByRoleAndComposite(ctx, roleID, compositeID)

	s.compositeMu.Lock()
	if errors.Is(err, pgx.ErrNoRows) {
		pruneCache(s.compositeCache, s.maxCacheSize)
		s.compositeCache[key] = &permissionCacheEntry{noRows: true, expiresAt: time.Now().Add(s.ttl)}
	} else if err == nil {
		pruneCache(s.compositeCache, s.maxCacheSize)
		s.compositeCache[key] = &permissionCacheEntry{permission: p, expiresAt: time.Now().Add(s.ttl)}
	}
	s.compositeMu.Unlock()

	return p, err
}

func (s *permissionService) cachedByRoleAndResource(ctx context.Context, roleID uuid.UUID, resourceType models.ResourceType) (*models.Permission, error) {
	key := roleID.String() + ":" + string(resourceType)

	s.resourceMu.RLock()
	entry, ok := s.resourceCache[key]
	s.resourceMu.RUnlock()
	if ok && time.Now().Before(entry.expiresAt) {
		if entry.noRows {
			return nil, pgx.ErrNoRows
		}
		return entry.permission, nil
	}

	p, err := s.repo.GetByRoleAndResource(ctx, roleID, resourceType)

	s.resourceMu.Lock()
	if errors.Is(err, pgx.ErrNoRows) {
		pruneCache(s.resourceCache, s.maxCacheSize)
		s.resourceCache[key] = &permissionCacheEntry{noRows: true, expiresAt: time.Now().Add(s.ttl)}
	} else if err == nil {
		pruneCache(s.resourceCache, s.maxCacheSize)
		s.resourceCache[key] = &permissionCacheEntry{permission: p, expiresAt: time.Now().Add(s.ttl)}
	}
	s.resourceMu.Unlock()

	return p, err
}

func (s *permissionService) evict(p *models.Permission) {
	if p.CompositeID != nil {
		s.compositeMu.Lock()
		delete(s.compositeCache, p.RoleID.String()+":"+p.CompositeID.String())
		s.compositeMu.Unlock()
	}
	if p.ResourceType != nil {
		s.resourceMu.Lock()
		delete(s.resourceCache, p.RoleID.String()+":"+string(*p.ResourceType))
		s.resourceMu.Unlock()
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

func (s *permissionService) CanReadEntity(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultRead, nil
	}
	p, err := s.cachedByRoleAndResource(ctx, *roleID, models.ResourceTypeEntity)
	if err == nil {
		return p.CanRead, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return s.CanRead(ctx, composite, roleID)
}

func (s *permissionService) CanWriteEntity(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultWrite, nil
	}
	p, err := s.cachedByRoleAndResource(ctx, *roleID, models.ResourceTypeEntity)
	if err == nil {
		return p.CanWrite, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return s.CanWrite(ctx, composite, roleID)
}

func (s *permissionService) CanReadFieldValue(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultRead, nil
	}
	p, err := s.cachedByRoleAndResource(ctx, *roleID, models.ResourceTypeFieldValue)
	if err == nil {
		return p.CanRead, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return s.CanRead(ctx, composite, roleID)
}

func (s *permissionService) CanWriteFieldValue(ctx context.Context, composite *models.Composite, roleID *uuid.UUID) (bool, error) {
	if roleID == nil {
		return composite.DefaultWrite, nil
	}
	p, err := s.cachedByRoleAndResource(ctx, *roleID, models.ResourceTypeFieldValue)
	if err == nil {
		return p.CanWrite, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return s.CanWrite(ctx, composite, roleID)
}
