package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

func TestPermissionService_CanReadResource(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()

	tests := []struct {
		name       string
		roleID     *uuid.UUID
		repoResult *models.Permission
		repoErr    error
		expected   bool
	}{
		{
			name:     "unauthenticated is always denied",
			roleID:   nil,
			expected: false,
		},
		{
			name:       "role with can_read=true is allowed",
			roleID:     &roleID,
			repoResult: &models.Permission{CanRead: true},
			expected:   true,
		},
		{
			name:       "role with can_read=false is denied",
			roleID:     &roleID,
			repoResult: &models.Permission{CanRead: false},
			expected:   false,
		},
		{
			name:     "no permission record defaults to denied",
			roleID:   &roleID,
			repoErr:  pgx.ErrNoRows,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewPermissionService(&mockPermissionRepo{
				getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
					return nil, pgx.ErrNoRows
				},
				getByRoleAndResource: func(_ context.Context, _ uuid.UUID, _ models.ResourceType) (*models.Permission, error) {
					return tt.repoResult, tt.repoErr
				},
			})

			got, err := svc.CanReadResource(ctx, models.ResourceTypeComposite, tt.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestPermissionService_CanWriteResource(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()

	tests := []struct {
		name       string
		roleID     *uuid.UUID
		repoResult *models.Permission
		repoErr    error
		expected   bool
	}{
		{
			name:     "unauthenticated is always denied",
			roleID:   nil,
			expected: false,
		},
		{
			name:       "role with can_write=true is allowed",
			roleID:     &roleID,
			repoResult: &models.Permission{CanWrite: true},
			expected:   true,
		},
		{
			name:       "role with can_write=false is denied",
			roleID:     &roleID,
			repoResult: &models.Permission{CanWrite: false},
			expected:   false,
		},
		{
			name:     "no permission record defaults to denied",
			roleID:   &roleID,
			repoErr:  pgx.ErrNoRows,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewPermissionService(&mockPermissionRepo{
				getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
					return nil, pgx.ErrNoRows
				},
				getByRoleAndResource: func(_ context.Context, _ uuid.UUID, _ models.ResourceType) (*models.Permission, error) {
					return tt.repoResult, tt.repoErr
				},
			})

			got, err := svc.CanWriteResource(ctx, models.ResourceTypeComposite, tt.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestPermissionService_CacheHit_Composite(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()
	compositeID := uuid.New()
	composite := &models.Composite{ID: compositeID, DefaultRead: false}

	calls := 0
	svc := service.NewPermissionService(&mockPermissionRepo{
		getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
			calls++
			return &models.Permission{CanRead: true, CanWrite: false}, nil
		},
	})

	for i := 0; i < 3; i++ {
		got, err := svc.CanRead(ctx, composite, &roleID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Errorf("expected true")
		}
	}

	if calls != 1 {
		t.Errorf("expected 1 repo call, got %d", calls)
	}
}

func TestPermissionService_CacheHit_Resource(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()

	calls := 0
	svc := service.NewPermissionService(&mockPermissionRepo{
		getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
			return nil, pgx.ErrNoRows
		},
		getByRoleAndResource: func(_ context.Context, _ uuid.UUID, _ models.ResourceType) (*models.Permission, error) {
			calls++
			return &models.Permission{CanRead: true}, nil
		},
	})

	for i := 0; i < 3; i++ {
		got, err := svc.CanReadResource(ctx, models.ResourceTypeComposite, &roleID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Errorf("expected true")
		}
	}

	if calls != 1 {
		t.Errorf("expected 1 repo call, got %d", calls)
	}
}

func TestPermissionService_CacheNoRowsHit(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()

	calls := 0
	svc := service.NewPermissionService(&mockPermissionRepo{
		getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
			return nil, pgx.ErrNoRows
		},
		getByRoleAndResource: func(_ context.Context, _ uuid.UUID, _ models.ResourceType) (*models.Permission, error) {
			calls++
			return nil, pgx.ErrNoRows
		},
	})

	for i := 0; i < 3; i++ {
		got, err := svc.CanReadResource(ctx, models.ResourceTypeUser, &roleID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Errorf("expected false")
		}
	}

	if calls != 1 {
		t.Errorf("expected 1 repo call for no-rows, got %d", calls)
	}
}

func TestPermissionService_CacheEvictedOnCreate(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()
	compositeID := uuid.New()
	composite := &models.Composite{ID: compositeID, DefaultRead: false}

	calls := 0
	svc := service.NewPermissionService(&mockPermissionRepo{
		getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
			calls++
			return &models.Permission{CanRead: true}, nil
		},
	})

	// Populate cache.
	if _, err := svc.CanRead(ctx, composite, &roleID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create evicts the key.
	if _, err := svc.Create(ctx, &models.Permission{RoleID: roleID, CompositeID: &compositeID, CanRead: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Next read must hit repo again.
	if _, err := svc.CanRead(ctx, composite, &roleID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 2 {
		t.Errorf("expected 2 repo calls (cache evicted), got %d", calls)
	}
}

func TestPermissionService_CanRead(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()
	compositeID := uuid.New()

	tests := []struct {
		name       string
		composite  *models.Composite
		roleID     *uuid.UUID
		repoResult *models.Permission
		repoErr    error
		expected   bool
	}{
		{
			name:      "unauthenticated, default_read=false",
			composite: &models.Composite{ID: compositeID, DefaultRead: false},
			roleID:    nil,
			expected:  false,
		},
		{
			name:      "unauthenticated, default_read=true",
			composite: &models.Composite{ID: compositeID, DefaultRead: true},
			roleID:    nil,
			expected:  true,
		},
		{
			name:       "authenticated, role permission can_read=true",
			composite:  &models.Composite{ID: compositeID, DefaultRead: false},
			roleID:     &roleID,
			repoResult: &models.Permission{CanRead: true},
			expected:   true,
		},
		{
			name:       "authenticated, role permission can_read=false",
			composite:  &models.Composite{ID: compositeID, DefaultRead: true},
			roleID:     &roleID,
			repoResult: &models.Permission{CanRead: false},
			expected:   false,
		},
		{
			name:      "authenticated, no permission record, falls back to default_read=true",
			composite: &models.Composite{ID: compositeID, DefaultRead: true},
			roleID:    &roleID,
			repoErr:   pgx.ErrNoRows,
			expected:  true,
		},
		{
			name:      "authenticated, no permission record, falls back to default_read=false",
			composite: &models.Composite{ID: compositeID, DefaultRead: false},
			roleID:    &roleID,
			repoErr:   pgx.ErrNoRows,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewPermissionService(&mockPermissionRepo{
				getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
					return tt.repoResult, tt.repoErr
				},
			})

			got, err := svc.CanRead(ctx, tt.composite, tt.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestPermissionService_CanWrite(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()
	compositeID := uuid.New()

	tests := []struct {
		name       string
		composite  *models.Composite
		roleID     *uuid.UUID
		repoResult *models.Permission
		repoErr    error
		expected   bool
	}{
		{
			name:      "unauthenticated, default_write=false",
			composite: &models.Composite{ID: compositeID, DefaultWrite: false},
			roleID:    nil,
			expected:  false,
		},
		{
			name:      "unauthenticated, default_write=true",
			composite: &models.Composite{ID: compositeID, DefaultWrite: true},
			roleID:    nil,
			expected:  true,
		},
		{
			name:       "authenticated, role permission can_write=true",
			composite:  &models.Composite{ID: compositeID, DefaultWrite: false},
			roleID:     &roleID,
			repoResult: &models.Permission{CanWrite: true},
			expected:   true,
		},
		{
			name:       "authenticated, role permission can_write=false",
			composite:  &models.Composite{ID: compositeID, DefaultWrite: true},
			roleID:     &roleID,
			repoResult: &models.Permission{CanWrite: false},
			expected:   false,
		},
		{
			name:      "authenticated, no permission record, falls back to default_write=true",
			composite: &models.Composite{ID: compositeID, DefaultWrite: true},
			roleID:    &roleID,
			repoErr:   pgx.ErrNoRows,
			expected:  true,
		},
		{
			name:      "authenticated, no permission record, falls back to default_write=false",
			composite: &models.Composite{ID: compositeID, DefaultWrite: false},
			roleID:    &roleID,
			repoErr:   pgx.ErrNoRows,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewPermissionService(&mockPermissionRepo{
				getByRoleAndComposite: func(_ context.Context, _, _ uuid.UUID) (*models.Permission, error) {
					return tt.repoResult, tt.repoErr
				},
			})

			got, err := svc.CanWrite(ctx, tt.composite, tt.roleID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}
