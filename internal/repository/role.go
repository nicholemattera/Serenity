package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) (*models.Role, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	List(ctx context.Context, p *Pagination) (*Page[models.Role], error)
	Update(ctx context.Context, role *models.Role) (*models.Role, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type roleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Create(ctx context.Context, role *models.Role) (*models.Role, error) {
	role.ID = uuid.New()
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO roles (id, name, hierarchy_level, session_timeout, allow_registration, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, role.ID, role.Name, role.HierarchyLevel, role.SessionTimeout, role.AllowRegistration, role.CreatedAt, role.UpdatedAt, role.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

func (r *roleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	role := &models.Role{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, hierarchy_level, session_timeout, allow_registration, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
		FROM roles
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&role.ID, &role.Name, &role.HierarchyLevel, &role.SessionTimeout, &role.AllowRegistration,
		&role.CreatedAt, &role.UpdatedAt, &role.DeletedAt,
		&role.CreatedBy, &role.UpdatedBy, &role.DeletedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return role, nil
}

func (r *roleRepository) List(ctx context.Context, p *Pagination) (*Page[models.Role], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM roles WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count roles: %w", err)
	}

	query, args := paginateQuery(`SELECT id, name, hierarchy_level, session_timeout, allow_registration, created_at, updated_at, deleted_at, created_by, updated_by, deleted_by FROM roles WHERE deleted_at IS NULL ORDER BY hierarchy_level ASC`, nil, p)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(
			&role.ID, &role.Name, &role.HierarchyLevel, &role.SessionTimeout, &role.AllowRegistration,
			&role.CreatedAt, &role.UpdatedAt, &role.DeletedAt,
			&role.CreatedBy, &role.UpdatedBy, &role.DeletedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return pageResult(roles, total, p), nil
}

func (r *roleRepository) Update(ctx context.Context, role *models.Role) (*models.Role, error) {
	role.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE roles
		SET name = $1, hierarchy_level = $2, session_timeout = $3, allow_registration = $4, updated_at = $5, updated_by = $6
		WHERE id = $7 AND deleted_at IS NULL
	`, role.Name, role.HierarchyLevel, role.SessionTimeout, role.AllowRegistration, role.UpdatedAt, role.UpdatedBy, role.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	} else if result.RowsAffected() == 0 {
		return nil, ErrNoRowsAffected
	}

	return role, nil
}

func (r *roleRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE roles SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	} else if result.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}

	return nil
}
