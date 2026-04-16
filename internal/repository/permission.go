package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type PermissionRepository interface {
	Create(ctx context.Context, permission *models.Permission) (*models.Permission, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	GetByRoleAndComposite(ctx context.Context, roleID, compositeID uuid.UUID) (*models.Permission, error)
	GetByRoleAndResource(ctx context.Context, roleID uuid.UUID, resourceType models.ResourceType) (*models.Permission, error)
	ListByRole(ctx context.Context, roleID uuid.UUID, p Pagination) (*Page[models.Permission], error)
	Update(ctx context.Context, permission *models.Permission) (*models.Permission, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type permissionRepository struct {
	db *pgxpool.Pool
}

func NewPermissionRepository(db *pgxpool.Pool) PermissionRepository {
	return &permissionRepository{db: db}
}

const permissionColumns = `id, role_id, composite_id, resource_type, can_read, can_write,
	created_at, updated_at, deleted_at, created_by, updated_by, deleted_by`

func scanPermission(s interface{ Scan(...any) error }, p *models.Permission) error {
	return s.Scan(
		&p.ID, &p.RoleID, &p.CompositeID, &p.ResourceType, &p.CanRead, &p.CanWrite,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		&p.CreatedBy, &p.UpdatedBy, &p.DeletedBy,
	)
}

func (r *permissionRepository) Create(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	permission.ID = uuid.New()
	now := time.Now()
	permission.CreatedAt = now
	permission.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO permissions (id, role_id, composite_id, resource_type, can_read, can_write, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, permission.ID, permission.RoleID, permission.CompositeID, permission.ResourceType,
		permission.CanRead, permission.CanWrite,
		permission.CreatedAt, permission.UpdatedAt, permission.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create permission: %w", err)
	}

	return permission, nil
}

func (r *permissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	permission := &models.Permission{}
	err := scanPermission(r.db.QueryRow(ctx, `
		SELECT `+permissionColumns+`
		FROM permissions
		WHERE id = $1 AND deleted_at IS NULL
	`, id), permission)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return permission, nil
}

func (r *permissionRepository) GetByRoleAndComposite(ctx context.Context, roleID, compositeID uuid.UUID) (*models.Permission, error) {
	permission := &models.Permission{}
	err := scanPermission(r.db.QueryRow(ctx, `
		SELECT `+permissionColumns+`
		FROM permissions
		WHERE role_id = $1 AND composite_id = $2 AND deleted_at IS NULL
	`, roleID, compositeID), permission)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return permission, nil
}

func (r *permissionRepository) GetByRoleAndResource(ctx context.Context, roleID uuid.UUID, resourceType models.ResourceType) (*models.Permission, error) {
	permission := &models.Permission{}
	err := scanPermission(r.db.QueryRow(ctx, `
		SELECT `+permissionColumns+`
		FROM permissions
		WHERE role_id = $1 AND resource_type = $2 AND deleted_at IS NULL
	`, roleID, resourceType), permission)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return permission, nil
}

func (r *permissionRepository) ListByRole(ctx context.Context, roleID uuid.UUID, p Pagination) (*Page[models.Permission], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM permissions WHERE role_id = $1 AND deleted_at IS NULL`, roleID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count permissions: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+permissionColumns+`
		FROM permissions
		WHERE role_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, roleID, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		if err := scanPermission(rows, &permission); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return &Page[models.Permission]{Data: permissions, Total: total, Limit: p.Limit, Offset: p.Offset}, nil
}

func (r *permissionRepository) Update(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	permission.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		UPDATE permissions
		SET can_read = $1, can_write = $2, updated_at = $3, updated_by = $4
		WHERE id = $5 AND deleted_at IS NULL
	`, permission.CanRead, permission.CanWrite, permission.UpdatedAt, permission.UpdatedBy, permission.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update permission: %w", err)
	}

	return permission, nil
}

func (r *permissionRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE permissions SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete permission: %w", err)
	}

	return nil
}
