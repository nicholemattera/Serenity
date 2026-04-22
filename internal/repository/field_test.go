package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/repository"
)

func TestFieldRepository_ListByComposites(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	compositeRepo := repository.NewCompositeRepository(db)
	fieldRepo := repository.NewFieldRepository(db)

	t.Run("empty input returns empty map", func(t *testing.T) {
		result, err := fieldRepo.ListByComposites(ctx, []uuid.UUID{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty map, got %d entries", len(result))
		}
	})

	t.Run("returns fields grouped by composite ID", func(t *testing.T) {
		compositeA := seedComposite(t, compositeRepo)
		compositeB := seedComposite(t, compositeRepo)

		seedField(t, fieldRepo, compositeA.ID)
		seedField(t, fieldRepo, compositeA.ID)
		seedField(t, fieldRepo, compositeB.ID)

		result, err := fieldRepo.ListByComposites(ctx, []uuid.UUID{compositeA.ID, compositeB.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result[compositeA.ID]) != 2 {
			t.Errorf("expected 2 fields for compositeA, got %d", len(result[compositeA.ID]))
		}
		if len(result[compositeB.ID]) != 1 {
			t.Errorf("expected 1 field for compositeB, got %d", len(result[compositeB.ID]))
		}

		for _, f := range result[compositeA.ID] {
			if f.CompositeID != compositeA.ID {
				t.Errorf("field belongs to wrong composite: got %v", f.CompositeID)
			}
		}
	})

	t.Run("composite with no fields is absent from result map", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)

		result, err := fieldRepo.ListByComposites(ctx, []uuid.UUID{composite.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result[composite.ID]; ok {
			t.Error("expected composite with no fields to be absent from result map")
		}
	})

	t.Run("deleted fields are excluded", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		field := seedField(t, fieldRepo, composite.ID)
		actor := uuid.New()

		if err := fieldRepo.Delete(ctx, field.ID, actor); err != nil {
			t.Fatalf("delete: %v", err)
		}

		result, err := fieldRepo.ListByComposites(ctx, []uuid.UUID{composite.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result[composite.ID]) != 0 {
			t.Errorf("expected deleted field to be excluded, got %d entries", len(result[composite.ID]))
		}
	})

	t.Run("fields ordered by position within composite", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)

		// Create fields and then update their positions explicitly via the Update method.
		// seedField creates with position=0; we update to set distinct ordering.
		f1 := seedField(t, fieldRepo, composite.ID)
		f2 := seedField(t, fieldRepo, composite.ID)
		f3 := seedField(t, fieldRepo, composite.ID)

		f1.Position = 1
		f2.Position = 2
		f3.Position = 3
		_, _ = fieldRepo.Update(ctx, f1)
		_, _ = fieldRepo.Update(ctx, f2)
		_, _ = fieldRepo.Update(ctx, f3)

		result, err := fieldRepo.ListByComposites(ctx, []uuid.UUID{composite.ID})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fields := result[composite.ID]
		if len(fields) != 3 {
			t.Fatalf("expected 3 fields, got %d", len(fields))
		}
		if fields[0].ID != f1.ID || fields[1].ID != f2.ID || fields[2].ID != f3.ID {
			t.Errorf("fields not in position order: got %v %v %v", fields[0].ID, fields[1].ID, fields[2].ID)
		}
	})
}
