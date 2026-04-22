package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

func TestFieldValueRepository_Upsert(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	compositeRepo := repository.NewCompositeRepository(db)
	fieldRepo := repository.NewFieldRepository(db)
	entityRepo := repository.NewEntityRepository(db)
	fvRepo := repository.NewFieldValueRepository(db)
	actor := uuid.New()

	t.Run("inserts new record", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		field := seedField(t, fieldRepo, composite.ID)
		entity := seedEntity(t, entityRepo, composite.ID)

		fv, err := fvRepo.Upsert(ctx, &models.FieldValue{
			EntityID: entity.ID,
			FieldID:  field.ID,
			Value:    "hello",
			Audit:    models.Audit{CreatedBy: &actor, UpdatedBy: &actor},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fv.ID == uuid.Nil {
			t.Error("expected non-nil ID")
		}
		if fv.Value != "hello" {
			t.Errorf("expected value %q, got %q", "hello", fv.Value)
		}
		if fv.EntityID != entity.ID {
			t.Errorf("expected entity_id %v, got %v", entity.ID, fv.EntityID)
		}
		if fv.FieldID != field.ID {
			t.Errorf("expected field_id %v, got %v", field.ID, fv.FieldID)
		}
	})

	t.Run("updates value on conflict", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		field := seedField(t, fieldRepo, composite.ID)
		entity := seedEntity(t, entityRepo, composite.ID)

		first, err := fvRepo.Upsert(ctx, &models.FieldValue{
			EntityID: entity.ID,
			FieldID:  field.ID,
			Value:    "original",
			Audit:    models.Audit{CreatedBy: &actor, UpdatedBy: &actor},
		})
		if err != nil {
			t.Fatalf("first upsert: %v", err)
		}

		second, err := fvRepo.Upsert(ctx, &models.FieldValue{
			EntityID: entity.ID,
			FieldID:  field.ID,
			Value:    "updated",
			Audit:    models.Audit{CreatedBy: &actor, UpdatedBy: &actor},
		})
		if err != nil {
			t.Fatalf("second upsert: %v", err)
		}

		if second.ID != first.ID {
			t.Errorf("expected same row ID on conflict: first=%v second=%v", first.ID, second.ID)
		}
		if second.Value != "updated" {
			t.Errorf("expected value %q after update, got %q", "updated", second.Value)
		}
	})

	t.Run("deleted record does not conflict — inserts fresh row", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		field := seedField(t, fieldRepo, composite.ID)
		entity := seedEntity(t, entityRepo, composite.ID)

		first, err := fvRepo.Upsert(ctx, &models.FieldValue{
			EntityID: entity.ID,
			FieldID:  field.ID,
			Value:    "before delete",
			Audit:    models.Audit{CreatedBy: &actor, UpdatedBy: &actor},
		})
		if err != nil {
			t.Fatalf("first upsert: %v", err)
		}

		if err := fvRepo.Delete(ctx, first.ID, actor); err != nil {
			t.Fatalf("delete: %v", err)
		}

		// The partial unique index only applies WHERE deleted_at IS NULL,
		// so a new row should be inserted rather than conflicting.
		second, err := fvRepo.Upsert(ctx, &models.FieldValue{
			EntityID: entity.ID,
			FieldID:  field.ID,
			Value:    "after delete",
			Audit:    models.Audit{CreatedBy: &actor, UpdatedBy: &actor},
		})
		if err != nil {
			t.Fatalf("upsert after delete: %v", err)
		}
		if second.ID == first.ID {
			t.Error("expected a new row after the previous one was deleted")
		}
		if second.Value != "after delete" {
			t.Errorf("expected value %q, got %q", "after delete", second.Value)
		}
	})
}

func TestFieldValueRepository_ListByEntities(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	compositeRepo := repository.NewCompositeRepository(db)
	fieldRepo := repository.NewFieldRepository(db)
	entityRepo := repository.NewEntityRepository(db)
	fvRepo := repository.NewFieldValueRepository(db)
	actor := uuid.New()

	t.Run("empty input returns empty map", func(t *testing.T) {
		result, err := fvRepo.ListByEntities(ctx, []uuid.UUID{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("returns field values grouped by entity ID", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		field1 := seedField(t, fieldRepo, composite.ID)
		field2 := seedField(t, fieldRepo, composite.ID)
		entityA := seedEntity(t, entityRepo, composite.ID)
		entityB := seedEntity(t, entityRepo, composite.ID)

		_, _ = fvRepo.Upsert(ctx, &models.FieldValue{EntityID: entityA.ID, FieldID: field1.ID, Value: "a1", Audit: models.Audit{CreatedBy: &actor, UpdatedBy: &actor}})
		_, _ = fvRepo.Upsert(ctx, &models.FieldValue{EntityID: entityA.ID, FieldID: field2.ID, Value: "a2", Audit: models.Audit{CreatedBy: &actor, UpdatedBy: &actor}})
		_, _ = fvRepo.Upsert(ctx, &models.FieldValue{EntityID: entityB.ID, FieldID: field1.ID, Value: "b1", Audit: models.Audit{CreatedBy: &actor, UpdatedBy: &actor}})

		result, err := fvRepo.ListByEntities(ctx, []uuid.UUID{entityA.ID, entityB.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result[entityA.ID]) != 2 {
			t.Errorf("expected 2 values for entityA, got %d", len(result[entityA.ID]))
		}
		if len(result[entityB.ID]) != 1 {
			t.Errorf("expected 1 value for entityB, got %d", len(result[entityB.ID]))
		}
	})

	t.Run("entity with no values is absent from result map", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		entity := seedEntity(t, entityRepo, composite.ID)

		result, err := fvRepo.ListByEntities(ctx, []uuid.UUID{entity.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result[entity.ID]; ok {
			t.Error("expected entity with no values to be absent from result map")
		}
	})

	t.Run("deleted values are excluded", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		field := seedField(t, fieldRepo, composite.ID)
		entity := seedEntity(t, entityRepo, composite.ID)

		fv, _ := fvRepo.Upsert(ctx, &models.FieldValue{EntityID: entity.ID, FieldID: field.ID, Value: "to delete", Audit: models.Audit{CreatedBy: &actor, UpdatedBy: &actor}})
		_ = fvRepo.Delete(ctx, fv.ID, actor)

		result, err := fvRepo.ListByEntities(ctx, []uuid.UUID{entity.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[entity.ID]) != 0 {
			t.Errorf("expected deleted value to be excluded, got %d entries", len(result[entity.ID]))
		}
	})
}
