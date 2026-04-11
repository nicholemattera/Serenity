package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type CompositeRepository interface {
	Create(ctx context.Context, composite *models.Composite) (*models.Composite, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Composite, error)
	GetBySlug(ctx context.Context, slug string) (*models.Composite, error)
	List(ctx context.Context, p Pagination) (*Page[models.Composite], error)
	Update(ctx context.Context, composite *models.Composite) (*models.Composite, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type compositeRepository struct {
	db *pgxpool.Pool
}

func NewCompositeRepository(db *pgxpool.Pool) CompositeRepository {
	return &compositeRepository{db: db}
}

const compositeColumns = `id, name, slug, default_read, default_write,
	created_at, updated_at, deleted_at, created_by, updated_by, deleted_by`

func scanComposite(s interface{ Scan(...any) error }, c *models.Composite) error {
	return s.Scan(
		&c.ID, &c.Name, &c.Slug, &c.DefaultRead, &c.DefaultWrite,
		&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
		&c.CreatedBy, &c.UpdatedBy, &c.DeletedBy,
	)
}

func (r *compositeRepository) Create(ctx context.Context, composite *models.Composite) (*models.Composite, error) {
	composite.ID = uuid.New()
	now := time.Now()
	composite.CreatedAt = now
	composite.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO composites (id, name, slug, default_read, default_write, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, composite.ID, composite.Name, composite.Slug, composite.DefaultRead, composite.DefaultWrite,
		composite.CreatedAt, composite.UpdatedAt, composite.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create composite: %w", err)
	}

	return composite, nil
}

func (r *compositeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Composite, error) {
	composite := &models.Composite{}
	err := scanComposite(r.db.QueryRow(ctx, `
		SELECT `+compositeColumns+`
		FROM composites
		WHERE id = $1 AND deleted_at IS NULL
	`, id), composite)
	if err != nil {
		return nil, fmt.Errorf("failed to get composite: %w", err)
	}

	return composite, nil
}

func (r *compositeRepository) GetBySlug(ctx context.Context, slug string) (*models.Composite, error) {
	composite := &models.Composite{}
	err := scanComposite(r.db.QueryRow(ctx, `
		SELECT `+compositeColumns+`
		FROM composites
		WHERE slug = $1 AND deleted_at IS NULL
	`, slug), composite)
	if err != nil {
		return nil, fmt.Errorf("failed to get composite by slug: %w", err)
	}

	return composite, nil
}

func (r *compositeRepository) List(ctx context.Context, p Pagination) (*Page[models.Composite], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM composites WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count composites: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT `+compositeColumns+`
		FROM composites
		WHERE deleted_at IS NULL
		ORDER BY name ASC
		LIMIT $1 OFFSET $2
	`, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list composites: %w", err)
	}
	defer rows.Close()

	var composites []models.Composite
	for rows.Next() {
		var composite models.Composite
		if err := scanComposite(rows, &composite); err != nil {
			return nil, fmt.Errorf("failed to scan composite: %w", err)
		}
		composites = append(composites, composite)
	}

	return &Page[models.Composite]{Data: composites, Total: total, Limit: p.Limit, Offset: p.Offset}, nil
}

func (r *compositeRepository) Update(ctx context.Context, composite *models.Composite) (*models.Composite, error) {
	composite.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		UPDATE composites
		SET name = $1, slug = $2, default_read = $3, default_write = $4, updated_at = $5, updated_by = $6
		WHERE id = $7 AND deleted_at IS NULL
	`, composite.Name, composite.Slug, composite.DefaultRead, composite.DefaultWrite,
		composite.UpdatedAt, composite.UpdatedBy, composite.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update composite: %w", err)
	}

	return composite, nil
}

func (r *compositeRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE composites SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete composite: %w", err)
	}

	return nil
}
