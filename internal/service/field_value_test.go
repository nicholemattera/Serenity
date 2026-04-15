package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

func TestFieldValueService_Set(t *testing.T) {
	ctx := context.Background()
	entityID := uuid.New()
	fieldID := uuid.New()

	existingFV := &models.FieldValue{
		ID:       uuid.New(),
		EntityID: entityID,
		FieldID:  fieldID,
		Value:    "old value",
	}

	t.Run("creates field value when none exists", func(t *testing.T) {
		created := false
		svc := service.NewFieldValueService(&mockFieldValueRepo{
			getByEntityAndField: func(_ context.Context, _, _ uuid.UUID) (*models.FieldValue, error) {
				return nil, pgx.ErrNoRows
			},
			create: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				created = true
				return fv, nil
			},
			update: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				t.Error("update should not be called when no existing value")
				return nil, nil
			},
		})

		_, err := svc.Set(ctx, &models.FieldValue{EntityID: entityID, FieldID: fieldID, Value: "new value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !created {
			t.Error("expected Create to be called")
		}
	})

	t.Run("updates field value when one already exists", func(t *testing.T) {
		updated := false
		var updatedValue string
		svc := service.NewFieldValueService(&mockFieldValueRepo{
			getByEntityAndField: func(_ context.Context, _, _ uuid.UUID) (*models.FieldValue, error) {
				return existingFV, nil
			},
			create: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				t.Error("create should not be called when existing value present")
				return nil, nil
			},
			update: func(_ context.Context, fv *models.FieldValue) (*models.FieldValue, error) {
				updated = true
				updatedValue = fv.Value
				return fv, nil
			},
		})

		_, err := svc.Set(ctx, &models.FieldValue{EntityID: entityID, FieldID: fieldID, Value: "updated value"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updated {
			t.Error("expected Update to be called")
		}
		if updatedValue != "updated value" {
			t.Errorf("expected updated value %q, got %q", "updated value", updatedValue)
		}
	})
}
