package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

// seedComposite inserts a composite so entities have a valid composite_id FK.
func seedComposite(t *testing.T, repo repository.CompositeRepository) *models.Composite {
	t.Helper()
	c, err := repo.Create(context.Background(), &models.Composite{
		Name: "Test Composite",
		Slug: uuid.NewString(), // unique per test
	})
	if err != nil {
		t.Fatalf("seedComposite: %v", err)
	}
	return c
}

func newEntity(compositeID uuid.UUID, name string) *models.Entity {
	return &models.Entity{
		CompositeID: compositeID,
		Name:        name,
		Slug:        name,
	}
}

func TestEntityRepository_CreateRootNodes(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := repository.NewEntityRepository(db)
	composite := seedComposite(t, repository.NewCompositeRepository(db))

	t.Run("first root node gets lft=1 rgt=2 and root_position=1", func(t *testing.T) {
		e, err := repo.Create(ctx, newEntity(composite.ID, "root-1"), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if e.Left != 1 || e.Right != 2 {
			t.Errorf("expected lft=1 rgt=2, got lft=%d rgt=%d", e.Left, e.Right)
		}
		if e.TreeID != e.ID {
			t.Errorf("expected tree_id == id for root node")
		}
		if e.RootPosition == nil || *e.RootPosition != 1 {
			t.Errorf("expected root_position=1, got %v", e.RootPosition)
		}
	})

	t.Run("second root node appends after first with root_position=2", func(t *testing.T) {
		first, _ := repo.Create(ctx, newEntity(composite.ID, "root-a"), nil, nil)
		second, err := repo.Create(ctx, newEntity(composite.ID, "root-b"), nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if second.Left != 1 || second.Right != 2 {
			t.Errorf("second root should have its own lft=1 rgt=2, got lft=%d rgt=%d", second.Left, second.Right)
		}
		if second.TreeID == first.TreeID {
			t.Errorf("root nodes should have distinct tree_ids")
		}
		if second.RootPosition == nil || *second.RootPosition != *first.RootPosition+1 {
			t.Errorf("expected root_position=%d, got %v", *first.RootPosition+1, second.RootPosition)
		}
	})
}

func TestEntityRepository_CreateChildren(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := repository.NewEntityRepository(db)
	composite := seedComposite(t, repository.NewCompositeRepository(db))

	root, _ := repo.Create(ctx, newEntity(composite.ID, "root"), nil, nil)

	t.Run("first child of root", func(t *testing.T) {
		child, err := repo.Create(ctx, newEntity(composite.ID, "child-1"), &root.ID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if child.Left != 2 || child.Right != 3 {
			t.Errorf("expected lft=2 rgt=3, got lft=%d rgt=%d", child.Left, child.Right)
		}
		if child.TreeID != root.TreeID {
			t.Errorf("child should share root's tree_id")
		}
		if child.RootPosition != nil {
			t.Errorf("non-root node should have nil root_position")
		}

		// Root should have expanded to contain the child
		updated, _ := repo.GetByID(ctx, root.ID)
		if updated.Right != 4 {
			t.Errorf("root rgt should be 4 after child insert, got %d", updated.Right)
		}
	})

	t.Run("second child appended after first using afterID", func(t *testing.T) {
		child1, _ := repo.Create(ctx, newEntity(composite.ID, "sib-1"), &root.ID, nil)
		child2, err := repo.Create(ctx, newEntity(composite.ID, "sib-2"), &root.ID, &child1.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if child2.Left != child1.Right+1 {
			t.Errorf("sib-2 lft should follow sib-1 rgt, got lft=%d (sib-1 rgt=%d)", child2.Left, child1.Right)
		}
	})

	t.Run("insert child between existing siblings", func(t *testing.T) {
		r, _ := repo.Create(ctx, newEntity(composite.ID, "root-mid"), nil, nil)
		first, _ := repo.Create(ctx, newEntity(composite.ID, "first"), &r.ID, nil)
		third, _ := repo.Create(ctx, newEntity(composite.ID, "third"), &r.ID, &first.ID)
		second, err := repo.Create(ctx, newEntity(composite.ID, "second"), &r.ID, &first.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if second.Left != first.Right+1 {
			t.Errorf("second should be inserted after first, got lft=%d", second.Left)
		}
		// third should have been shifted right
		updatedThird, _ := repo.GetByID(ctx, third.ID)
		if updatedThird.Left != second.Right+1 {
			t.Errorf("third should have shifted right of second, got lft=%d", updatedThird.Left)
		}
	})
}

func TestEntityRepository_CreateValidation(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := repository.NewEntityRepository(db)
	compositeA := seedComposite(t, repository.NewCompositeRepository(db))
	compositeB := seedComposite(t, repository.NewCompositeRepository(db))

	rootA, _ := repo.Create(ctx, newEntity(compositeA.ID, "root-a"), nil, nil)
	rootB, _ := repo.Create(ctx, newEntity(compositeB.ID, "root-b"), nil, nil)

	t.Run("afterID from different tree is rejected", func(t *testing.T) {
		_, err := repo.Create(ctx, newEntity(compositeA.ID, "child"), &rootA.ID, &rootB.ID)
		if err == nil {
			t.Error("expected error when afterID belongs to a different tree")
		}
	})
}

func TestEntityRepository_Move(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := repository.NewEntityRepository(db)
	composite := seedComposite(t, repository.NewCompositeRepository(db))

	t.Run("move leaf forward among siblings", func(t *testing.T) {
		root, _ := repo.Create(ctx, newEntity(composite.ID, "root-mv-fwd"), nil, nil)
		a, _ := repo.Create(ctx, newEntity(composite.ID, "a"), &root.ID, nil)
		b, _ := repo.Create(ctx, newEntity(composite.ID, "b"), &root.ID, &a.ID)
		c, _ := repo.Create(ctx, newEntity(composite.ID, "c"), &root.ID, &b.ID)

		// Move a after c: order becomes b, c, a
		if err := repo.Move(ctx, a.ID, &root.ID, &c.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		children, err := repo.ListChildren(ctx, root.ID, &repository.Pagination{Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error listing children: %v", err)
		}
		if len(children.Data) != 3 {
			t.Fatalf("expected 3 children, got %d", len(children.Data))
		}
		order := []uuid.UUID{b.ID, c.ID, a.ID}
		for i, id := range order {
			if children.Data[i].ID != id {
				t.Errorf("position %d: expected %v, got %v", i, id, children.Data[i].ID)
			}
		}
	})

	t.Run("move leaf backward among siblings", func(t *testing.T) {
		root, _ := repo.Create(ctx, newEntity(composite.ID, "root-mv-bwd"), nil, nil)
		a, _ := repo.Create(ctx, newEntity(composite.ID, "bwd-a"), &root.ID, nil)
		b, _ := repo.Create(ctx, newEntity(composite.ID, "bwd-b"), &root.ID, &a.ID)
		c, _ := repo.Create(ctx, newEntity(composite.ID, "bwd-c"), &root.ID, &b.ID)

		// Move c to first: order becomes c, a, b
		if err := repo.Move(ctx, c.ID, &root.ID, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		children, err := repo.ListChildren(ctx, root.ID, &repository.Pagination{Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error listing children: %v", err)
		}
		order := []uuid.UUID{c.ID, a.ID, b.ID}
		for i, id := range order {
			if children.Data[i].ID != id {
				t.Errorf("position %d: expected %v, got %v", i, id, children.Data[i].ID)
			}
		}
	})

	t.Run("move subtree preserves internal structure", func(t *testing.T) {
		root, _ := repo.Create(ctx, newEntity(composite.ID, "root-subtree"), nil, nil)
		a, _ := repo.Create(ctx, newEntity(composite.ID, "sub-a"), &root.ID, nil)
		a1, _ := repo.Create(ctx, newEntity(composite.ID, "sub-a1"), &a.ID, nil)
		a2, _ := repo.Create(ctx, newEntity(composite.ID, "sub-a2"), &a.ID, &a1.ID)
		b, _ := repo.Create(ctx, newEntity(composite.ID, "sub-b"), &root.ID, &a.ID)

		// Move a (with a1, a2) after b
		if err := repo.Move(ctx, a.ID, &root.ID, &b.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Root's children should now be b, a
		children, err := repo.ListChildren(ctx, root.ID, &repository.Pagination{Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error listing children: %v", err)
		}
		if len(children.Data) != 2 || children.Data[0].ID != b.ID || children.Data[1].ID != a.ID {
			t.Errorf("expected children [b, a], got %v", children.Data)
		}

		// a's children should still be a1, a2 in order
		aChildren, err := repo.ListChildren(ctx, a.ID, &repository.Pagination{Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error listing a's children: %v", err)
		}
		if len(aChildren.Data) != 2 || aChildren.Data[0].ID != a1.ID || aChildren.Data[1].ID != a2.ID {
			t.Errorf("subtree children should be preserved as [a1, a2], got %v", aChildren.Data)
		}
	})

	t.Run("move rejects entity from different tree", func(t *testing.T) {
		rootA, _ := repo.Create(ctx, newEntity(composite.ID, "root-val-a"), nil, nil)
		rootB, _ := repo.Create(ctx, newEntity(composite.ID, "root-val-b"), nil, nil)
		child, _ := repo.Create(ctx, newEntity(composite.ID, "val-child"), &rootA.ID, nil)

		if err := repo.Move(ctx, child.ID, &rootB.ID, nil); err == nil {
			t.Error("expected error when moving to a different tree")
		}
	})
}

func TestEntityRepository_MoveRoot(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := repository.NewEntityRepository(db)
	compositeRepo := repository.NewCompositeRepository(db)

	t.Run("move root forward", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		r1, _ := repo.Create(ctx, newEntity(composite.ID, "r1"), nil, nil)
		r2, _ := repo.Create(ctx, newEntity(composite.ID, "r2"), nil, nil)
		r3, _ := repo.Create(ctx, newEntity(composite.ID, "r3"), nil, nil)

		// Move r1 after r3: order becomes r2, r3, r1
		if err := repo.MoveRoot(ctx, r1.ID, &r3.ID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := repo.ListByComposite(ctx, composite.ID, &repository.Pagination{Limit: 10, Offset: 0})
		if err != nil {
			t.Fatalf("unexpected error listing: %v", err)
		}
		roots := rootsOnly(page.Data)
		if len(roots) != 3 {
			t.Fatalf("expected 3 roots, got %d", len(roots))
		}
		if roots[0].ID != r2.ID || roots[1].ID != r3.ID || roots[2].ID != r1.ID {
			t.Errorf("expected order [r2, r3, r1], got %v", ids(roots))
		}
	})

	t.Run("move root backward to first", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		r1, _ := repo.Create(ctx, newEntity(composite.ID, "r1"), nil, nil)
		r2, _ := repo.Create(ctx, newEntity(composite.ID, "r2"), nil, nil)
		r3, _ := repo.Create(ctx, newEntity(composite.ID, "r3"), nil, nil)

		// Move r3 to first: order becomes r3, r1, r2
		if err := repo.MoveRoot(ctx, r3.ID, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		page, err := repo.ListByComposite(ctx, composite.ID, &repository.Pagination{Limit: 10, Offset: 0})
		if err != nil {
			t.Fatalf("unexpected error listing: %v", err)
		}

		roots := rootsOnly(page.Data)
		if len(roots) != 3 {
			t.Fatalf("expected 3 roots, got %d", len(roots))
		}
		if roots[0].ID != r3.ID || roots[1].ID != r1.ID || roots[2].ID != r2.ID {
			t.Errorf("expected order [r3, r1, r2], got %v", ids(roots))
		}
	})

	t.Run("MoveRoot rejects non-root node", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		root, _ := repo.Create(ctx, newEntity(composite.ID, "mr-root"), nil, nil)
		child, _ := repo.Create(ctx, newEntity(composite.ID, "mr-child"), &root.ID, nil)

		if err := repo.MoveRoot(ctx, child.ID, nil); err == nil {
			t.Error("expected error when calling MoveRoot on a non-root node")
		}
	})

	t.Run("MoveRoot rejects afterID from different composite", func(t *testing.T) {
		composite := seedComposite(t, compositeRepo)
		otherComposite := seedComposite(t, compositeRepo)
		r1, _ := repo.Create(ctx, newEntity(composite.ID, "mr-cross-1"), nil, nil)
		r2, _ := repo.Create(ctx, newEntity(otherComposite.ID, "mr-cross-2"), nil, nil)

		if err := repo.MoveRoot(ctx, r1.ID, &r2.ID); err == nil {
			t.Error("expected error when afterID belongs to a different composite")
		}
	})
}

// rootsOnly filters to only root entities (lft == 1) in lft order.
func rootsOnly(entities []models.Entity) []models.Entity {
	var roots []models.Entity
	for _, e := range entities {
		if e.Left == 1 {
			roots = append(roots, e)
		}
	}
	return roots
}

func ids(entities []models.Entity) []uuid.UUID {
	out := make([]uuid.UUID, len(entities))
	for i, e := range entities {
		out[i] = e.ID
	}
	return out
}
