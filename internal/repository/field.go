package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type FieldRepository interface {
	Create(ctx context.Context, field *models.Field) (*models.Field, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Field, error)
	GetBySlug(ctx context.Context, compositeID uuid.UUID, slug string) (*models.Field, error)
	ListByComposite(ctx context.Context, compositeID uuid.UUID, p *Pagination) (*Page[models.Field], error)
	ListByComposites(ctx context.Context, compositeIDs []uuid.UUID) (map[uuid.UUID][]models.Field, error)
	Update(ctx context.Context, field *models.Field) (*models.Field, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type fieldRepository struct {
	db *pgxpool.Pool
}

func NewFieldRepository(db *pgxpool.Pool) FieldRepository {
	return &fieldRepository{db: db}
}

const fieldColumns = `id, composite_id, name, slug, type, required, position, default_value, metadata,
	created_at, updated_at, deleted_at, created_by, updated_by, deleted_by`

func scanField(s interface{ Scan(...any) error }, f *models.Field) error {
	return s.Scan(
		&f.ID, &f.CompositeID, &f.Name, &f.Slug, &f.Type, &f.Required, &f.Position, &f.DefaultValue, &f.Metadata,
		&f.CreatedAt, &f.UpdatedAt, &f.DeletedAt,
		&f.CreatedBy, &f.UpdatedBy, &f.DeletedBy,
	)
}

func (r *fieldRepository) Create(ctx context.Context, field *models.Field) (*models.Field, error) {
	field.ID = uuid.New()
	now := time.Now()
	field.CreatedAt = now
	field.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO fields (id, composite_id, name, slug, type, required, position, default_value, metadata, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, field.ID, field.CompositeID, field.Name, field.Slug, field.Type,
		field.Required, field.Position, field.DefaultValue, field.Metadata,
		field.CreatedAt, field.UpdatedAt, field.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create field: %w", err)
	}

	return field, nil
}

func (r *fieldRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Field, error) {
	field := &models.Field{}
	err := scanField(r.db.QueryRow(ctx, `
		SELECT `+fieldColumns+`
		FROM fields
		WHERE id = $1 AND deleted_at IS NULL
	`, id), field)
	if err != nil {
		return nil, fmt.Errorf("failed to get field: %w", err)
	}

	return field, nil
}

func (r *fieldRepository) GetBySlug(ctx context.Context, compositeID uuid.UUID, slug string) (*models.Field, error) {
	field := &models.Field{}
	err := scanField(r.db.QueryRow(ctx, `
		SELECT `+fieldColumns+`
		FROM fields
		WHERE composite_id = $1 AND slug = $2 AND deleted_at IS NULL
	`, compositeID, slug), field)
	if err != nil {
		return nil, fmt.Errorf("failed to get field: %w", err)
	}
	return field, nil
}

func (r *fieldRepository) ListByComposite(ctx context.Context, compositeID uuid.UUID, p *Pagination) (*Page[models.Field], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM fields WHERE composite_id = $1 AND deleted_at IS NULL`, compositeID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count fields: %w", err)
	}

	query := `SELECT ` + fieldColumns + ` FROM fields WHERE composite_id = $1 AND deleted_at IS NULL ORDER BY position ASC`
	args := []any{compositeID}
	if p != nil {
		query += ` LIMIT $2 OFFSET $3`
		args = append(args, p.Limit, p.Offset)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list fields: %w", err)
	}
	defer rows.Close()

	var fields []models.Field
	for rows.Next() {
		var field models.Field
		if err := scanField(rows, &field); err != nil {
			return nil, fmt.Errorf("failed to scan field: %w", err)
		}
		fields = append(fields, field)
	}

	page := &Page[models.Field]{Data: fields, Total: total}
	if p != nil {
		page.Limit = p.Limit
		page.Offset = p.Offset
	}
	return page, nil
}

func (r *fieldRepository) ListByComposites(ctx context.Context, compositeIDs []uuid.UUID) (map[uuid.UUID][]models.Field, error) {
	if len(compositeIDs) == 0 {
		return map[uuid.UUID][]models.Field{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT `+fieldColumns+`
		FROM fields
		WHERE composite_id = ANY($1) AND deleted_at IS NULL
		ORDER BY composite_id, position ASC
	`, compositeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list fields by composites: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]models.Field)
	for rows.Next() {
		var f models.Field
		if err := scanField(rows, &f); err != nil {
			return nil, fmt.Errorf("failed to scan field: %w", err)
		}
		result[f.CompositeID] = append(result[f.CompositeID], f)
	}
	return result, nil
}

func (r *fieldRepository) Update(ctx context.Context, field *models.Field) (*models.Field, error) {
	field.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		UPDATE fields
		SET name = $1, slug = $2, type = $3, required = $4, position = $5,
		    default_value = $6, metadata = $7, updated_at = $8, updated_by = $9
		WHERE id = $10 AND deleted_at IS NULL
	`, field.Name, field.Slug, field.Type, field.Required, field.Position,
		field.DefaultValue, field.Metadata, field.UpdatedAt, field.UpdatedBy, field.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update field: %w", err)
	}

	return field, nil
}

func (r *fieldRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE fields SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete field: %w", err)
	}

	return nil
}
