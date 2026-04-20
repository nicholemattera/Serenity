package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type FieldValueRepository interface {
	Create(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.FieldValue, error)
	GetByEntityAndField(ctx context.Context, entityID, fieldID uuid.UUID) (*models.FieldValue, error)
	ListByEntity(ctx context.Context, entityID uuid.UUID, p *Pagination) (*Page[models.FieldValue], error)
	Update(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type fieldValueRepository struct {
	db *pgxpool.Pool
}

func NewFieldValueRepository(db *pgxpool.Pool) FieldValueRepository {
	return &fieldValueRepository{db: db}
}

const fieldValueColumns = `id, entity_id, field_id, value,
	created_at, updated_at, deleted_at, created_by, updated_by, deleted_by`

func scanFieldValue(s interface{ Scan(...any) error }, fv *models.FieldValue) error {
	return s.Scan(
		&fv.ID, &fv.EntityID, &fv.FieldID, &fv.Value,
		&fv.CreatedAt, &fv.UpdatedAt, &fv.DeletedAt,
		&fv.CreatedBy, &fv.UpdatedBy, &fv.DeletedBy,
	)
}

func (r *fieldValueRepository) Create(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
	fv.ID = uuid.New()
	now := time.Now()
	fv.CreatedAt = now
	fv.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO field_values (id, entity_id, field_id, value, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, fv.ID, fv.EntityID, fv.FieldID, fv.Value, fv.CreatedAt, fv.UpdatedAt, fv.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create field value: %w", err)
	}

	return fv, nil
}

func (r *fieldValueRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FieldValue, error) {
	fv := &models.FieldValue{}
	err := scanFieldValue(r.db.QueryRow(ctx, `
		SELECT `+fieldValueColumns+`
		FROM field_values
		WHERE id = $1 AND deleted_at IS NULL
	`, id), fv)
	if err != nil {
		return nil, fmt.Errorf("failed to get field value: %w", err)
	}

	return fv, nil
}

func (r *fieldValueRepository) GetByEntityAndField(ctx context.Context, entityID, fieldID uuid.UUID) (*models.FieldValue, error) {
	fv := &models.FieldValue{}
	err := scanFieldValue(r.db.QueryRow(ctx, `
		SELECT `+fieldValueColumns+`
		FROM field_values
		WHERE entity_id = $1 AND field_id = $2 AND deleted_at IS NULL
	`, entityID, fieldID), fv)
	if err != nil {
		return nil, fmt.Errorf("failed to get field value: %w", err)
	}

	return fv, nil
}

func (r *fieldValueRepository) ListByEntity(ctx context.Context, entityID uuid.UUID, p *Pagination) (*Page[models.FieldValue], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM field_values WHERE entity_id = $1 AND deleted_at IS NULL`, entityID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count field values: %w", err)
	}

	query, args := paginateQuery(`SELECT `+fieldValueColumns+` FROM field_values WHERE entity_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC`, []any{entityID}, p)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list field values: %w", err)
	}
	defer rows.Close()

	var values []models.FieldValue
	for rows.Next() {
		var fv models.FieldValue
		if err := scanFieldValue(rows, &fv); err != nil {
			return nil, fmt.Errorf("failed to scan field value: %w", err)
		}
		values = append(values, fv)
	}

	return pageResult(values, total, p), nil
}

func (r *fieldValueRepository) Update(ctx context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
	fv.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		UPDATE field_values
		SET value = $1, updated_at = $2, updated_by = $3
		WHERE id = $4 AND deleted_at IS NULL
	`, fv.Value, fv.UpdatedAt, fv.UpdatedBy, fv.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update field value: %w", err)
	}

	return fv, nil
}

func (r *fieldValueRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE field_values SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete field value: %w", err)
	}

	return nil
}
