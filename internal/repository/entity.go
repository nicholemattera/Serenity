package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type EntityRepository interface {
	// Create inserts a new entity. parentID nil = root node. afterID nil = first among siblings,
	// otherwise inserted immediately after the specified sibling.
	Create(ctx context.Context, entity *models.Entity, parentID *uuid.UUID, afterID *uuid.UUID) (*models.Entity, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.Entity, error)
	GetBySlug(ctx context.Context, compositeID uuid.UUID, slug string) (*models.Entity, error)
	ListByComposite(ctx context.Context, compositeID uuid.UUID, p *Pagination) (*Page[models.Entity], error)
	ListChildren(ctx context.Context, parentID uuid.UUID, p *Pagination) (*Page[models.Entity], error)
	Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, afterID *uuid.UUID) error
	// MoveRoot repositions a root entity within its composite. afterID nil = move to first position.
	MoveRoot(ctx context.Context, id uuid.UUID, afterID *uuid.UUID) error
	Update(ctx context.Context, entity *models.Entity) (*models.Entity, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type entityRepository struct {
	db *pgxpool.Pool
}

func NewEntityRepository(db *pgxpool.Pool) EntityRepository {
	return &entityRepository{db: db}
}

const entityColumns = `id, composite_id, tree_id, name, slug, lft, rgt, root_position,
	created_at, updated_at, deleted_at, created_by, updated_by, deleted_by`

func scanEntity(s interface{ Scan(...any) error }, e *models.Entity) error {
	return s.Scan(
		&e.ID, &e.CompositeID, &e.TreeID, &e.Name, &e.Slug, &e.Left, &e.Right, &e.RootPosition,
		&e.CreatedAt, &e.UpdatedAt, &e.DeletedAt,
		&e.CreatedBy, &e.UpdatedBy, &e.DeletedBy,
	)
}

func (r *entityRepository) Create(ctx context.Context, entity *models.Entity, parentID *uuid.UUID, afterID *uuid.UUID) (*models.Entity, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var treeID uuid.UUID
	var insertLft int
	var rootPosition *int

	if parentID == nil && afterID == nil {
		// New root node: self-contained tree, append after current max root_position
		entity.ID = uuid.New()
		treeID = entity.ID
		insertLft = 1

		var maxPos *int
		if err = tx.QueryRow(ctx, `
			SELECT MAX(root_position) FROM entities WHERE composite_id = $1 AND lft = 1 AND deleted_at IS NULL
		`, entity.CompositeID).Scan(&maxPos); err != nil {
			return nil, fmt.Errorf("failed to find max root position: %w", err)
		}
		pos := 1
		if maxPos != nil {
			pos = *maxPos + 1
		}
		rootPosition = &pos
	} else {
		// Resolve tree_id and insertion point from parent or sibling
		var refID uuid.UUID
		if parentID != nil {
			refID = *parentID
		} else {
			refID = *afterID
		}

		var refTreeID uuid.UUID
		var refLft, refRgt int
		if err = tx.QueryRow(ctx, `
			SELECT tree_id, lft, rgt FROM entities WHERE id = $1 AND deleted_at IS NULL
		`, refID).Scan(&refTreeID, &refLft, &refRgt); err != nil {
			return nil, fmt.Errorf("failed to find reference entity: %w", err)
		}

		// Validate afterID is in the same tree as parentID (if both provided)
		if parentID != nil && afterID != nil {
			var afterTreeID uuid.UUID
			if err = tx.QueryRow(ctx, `
				SELECT tree_id FROM entities WHERE id = $1 AND deleted_at IS NULL
			`, *afterID).Scan(&afterTreeID); err != nil {
				return nil, fmt.Errorf("failed to find after entity: %w", err)
			}
			if afterTreeID != refTreeID {
				return nil, fmt.Errorf("after entity does not belong to the same tree as parent")
			}
			var afterRgt int
			if err = tx.QueryRow(ctx, `SELECT rgt FROM entities WHERE id = $1 AND deleted_at IS NULL`, *afterID).Scan(&afterRgt); err != nil {
				return nil, fmt.Errorf("failed to find after entity rgt: %w", err)
			}
			insertLft = afterRgt + 1
		} else if afterID != nil {
			insertLft = refRgt + 1
		} else {
			insertLft = refLft + 1
		}

		treeID = refTreeID
		entity.ID = uuid.New()
	}

	// Shift existing nodes in this tree to make room
	if _, err = tx.Exec(ctx, `
		UPDATE entities SET rgt = rgt + 2 WHERE tree_id = $1 AND rgt >= $2 AND deleted_at IS NULL
	`, treeID, insertLft); err != nil {
		return nil, fmt.Errorf("failed to shift rgt values: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		UPDATE entities SET lft = lft + 2 WHERE tree_id = $1 AND lft >= $2 AND deleted_at IS NULL
	`, treeID, insertLft); err != nil {
		return nil, fmt.Errorf("failed to shift lft values: %w", err)
	}

	entity.TreeID = treeID
	entity.Left = insertLft
	entity.Right = insertLft + 1
	entity.RootPosition = rootPosition
	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now

	if _, err = tx.Exec(ctx, `
		INSERT INTO entities (id, composite_id, tree_id, name, slug, lft, rgt, root_position, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, entity.ID, entity.CompositeID, entity.TreeID, entity.Name, entity.Slug,
		entity.Left, entity.Right, entity.RootPosition, entity.CreatedAt, entity.UpdatedAt, entity.CreatedBy); err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return entity, nil
}

func (r *entityRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Entity, error) {
	entity := &models.Entity{}
	err := scanEntity(r.db.QueryRow(ctx, `
		SELECT `+entityColumns+`
		FROM entities
		WHERE id = $1 AND deleted_at IS NULL
	`, id), entity)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	return entity, nil
}

func (r *entityRepository) GetBySlug(ctx context.Context, compositeID uuid.UUID, slug string) (*models.Entity, error) {
	entity := &models.Entity{}
	err := scanEntity(r.db.QueryRow(ctx, `
		SELECT `+entityColumns+`
		FROM entities
		WHERE composite_id = $1 AND slug = $2 AND deleted_at IS NULL
	`, compositeID, slug), entity)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity by slug: %w", err)
	}

	return entity, nil
}

func (r *entityRepository) ListByComposite(ctx context.Context, compositeID uuid.UUID, p *Pagination) (*Page[models.Entity], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM entities WHERE composite_id = $1 AND deleted_at IS NULL`, compositeID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count entities: %w", err)
	}

	// Root nodes ordered by root_position; non-root nodes follow their root by lft
	query, args := paginateQuery(`SELECT `+entityColumns+` FROM entities WHERE composite_id = $1 AND deleted_at IS NULL ORDER BY root_position ASC, lft ASC`, []any{compositeID}, p)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var entity models.Entity
		if err := scanEntity(rows, &entity); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	return pageResult(entities, total, p), nil
}

// ListChildren returns the direct children of the given entity using the Nested Set Model.
func (r *entityRepository) ListChildren(ctx context.Context, parentID uuid.UUID, p *Pagination) (*Page[models.Entity], error) {
	var treeID uuid.UUID
	var parentLft, parentRgt int
	err := r.db.QueryRow(ctx, `
		SELECT tree_id, lft, rgt FROM entities WHERE id = $1 AND deleted_at IS NULL
	`, parentID).Scan(&treeID, &parentLft, &parentRgt)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent entity: %w", err)
	}

	var total int
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM entities
		WHERE tree_id = $1 AND lft > $2 AND rgt < $3 AND deleted_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM entities p
		    WHERE p.tree_id = $1 AND p.deleted_at IS NULL
		      AND p.lft > $2 AND p.rgt < $3
		      AND entities.lft > p.lft AND entities.rgt < p.rgt
		  )
	`, treeID, parentLft, parentRgt).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count children: %w", err)
	}

	childBaseQuery := `SELECT ` + entityColumns + ` FROM entities WHERE tree_id = $1 AND lft > $2 AND rgt < $3 AND deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM entities p WHERE p.tree_id = $1 AND p.deleted_at IS NULL AND p.lft > $2 AND p.rgt < $3 AND entities.lft > p.lft AND entities.rgt < p.rgt) ORDER BY lft ASC`
	childQuery, childArgs := paginateQuery(childBaseQuery, []any{treeID, parentLft, parentRgt}, p)
	rows, err := r.db.Query(ctx, childQuery, childArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to list children: %w", err)
	}
	defer rows.Close()

	var entities []models.Entity
	for rows.Next() {
		var entity models.Entity
		if err := scanEntity(rows, &entity); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}
		entities = append(entities, entity)
	}

	return pageResult(entities, total, p), nil
}

// Move repositions an entity (and its subtree) within the same tree.
// parentID nil = move to root level. afterID nil = move to first among siblings.
func (r *entityRepository) Move(ctx context.Context, id uuid.UUID, parentID *uuid.UUID, afterID *uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var treeID uuid.UUID
	var nodeLft, nodeRgt int
	if err := tx.QueryRow(ctx, `
		SELECT tree_id, lft, rgt FROM entities WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&treeID, &nodeLft, &nodeRgt); err != nil {
		return fmt.Errorf("failed to find entity: %w", err)
	}

	// Validate parentID and afterID belong to the same tree
	if parentID != nil {
		var parentTreeID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT tree_id FROM entities WHERE id = $1 AND deleted_at IS NULL`, *parentID).Scan(&parentTreeID); err != nil {
			return fmt.Errorf("failed to find parent entity: %w", err)
		}
		if parentTreeID != treeID {
			return fmt.Errorf("parent entity does not belong to the same tree")
		}
	}
	if afterID != nil {
		var afterTreeID uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT tree_id FROM entities WHERE id = $1 AND deleted_at IS NULL`, *afterID).Scan(&afterTreeID); err != nil {
			return fmt.Errorf("failed to find after entity: %w", err)
		}
		if afterTreeID != treeID {
			return fmt.Errorf("after entity does not belong to the same tree")
		}
	}

	subtreeWidth := nodeRgt - nodeLft + 1

	// Resolve the target insertion lft (before shifting)
	var targetLft int
	if afterID != nil {
		if err := tx.QueryRow(ctx, `SELECT rgt FROM entities WHERE id = $1 AND deleted_at IS NULL`, *afterID).Scan(&targetLft); err != nil {
			return fmt.Errorf("failed to find after entity: %w", err)
		}
		targetLft++ // insert after sibling
	} else if parentID != nil {
		if err := tx.QueryRow(ctx, `SELECT lft FROM entities WHERE id = $1 AND deleted_at IS NULL`, *parentID).Scan(&targetLft); err != nil {
			return fmt.Errorf("failed to find parent entity: %w", err)
		}
		targetLft++ // first child slot
	} else {
		var maxRgt *int
		if err := tx.QueryRow(ctx, `SELECT MAX(rgt) FROM entities WHERE tree_id = $1 AND deleted_at IS NULL`, treeID).Scan(&maxRgt); err != nil {
			return fmt.Errorf("failed to find max rgt: %w", err)
		}
		if maxRgt == nil {
			targetLft = 1
		} else {
			targetLft = *maxRgt + 1
		}
	}

	// Temporarily mark the subtree with negative values to exclude it from shifts
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET lft = -(lft), rgt = -(rgt)
		WHERE tree_id = $1 AND lft >= $2 AND rgt <= $3 AND deleted_at IS NULL
	`, treeID, nodeLft, nodeRgt); err != nil {
		return fmt.Errorf("failed to detach subtree: %w", err)
	}

	// Close the gap left by the removed subtree
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET rgt = rgt - $1 WHERE tree_id = $2 AND rgt > $3 AND deleted_at IS NULL
	`, subtreeWidth, treeID, nodeRgt); err != nil {
		return fmt.Errorf("failed to close gap (rgt): %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET lft = lft - $1 WHERE tree_id = $2 AND lft > $3 AND deleted_at IS NULL
	`, subtreeWidth, treeID, nodeRgt); err != nil {
		return fmt.Errorf("failed to close gap (lft): %w", err)
	}

	// Adjust targetLft if it was after the removed subtree
	if targetLft > nodeRgt {
		targetLft -= subtreeWidth
	}

	// Open space at target
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET rgt = rgt + $1 WHERE tree_id = $2 AND rgt >= $3 AND deleted_at IS NULL
	`, subtreeWidth, treeID, targetLft); err != nil {
		return fmt.Errorf("failed to open space (rgt): %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET lft = lft + $1 WHERE tree_id = $2 AND lft >= $3 AND deleted_at IS NULL
	`, subtreeWidth, treeID, targetLft); err != nil {
		return fmt.Errorf("failed to open space (lft): %w", err)
	}

	// Re-insert the subtree at its new position
	offset := targetLft - nodeLft
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET lft = -(lft) + $1, rgt = -(rgt) + $1
		WHERE tree_id = $2 AND lft < 0 AND deleted_at IS NULL
	`, offset, treeID); err != nil {
		return fmt.Errorf("failed to reinsert subtree: %w", err)
	}

	return tx.Commit(ctx)
}

// MoveRoot repositions a root entity among siblings within its composite.
// afterID nil = move to first position. afterID set = move immediately after that root entity.
func (r *entityRepository) MoveRoot(ctx context.Context, id uuid.UUID, afterID *uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var compositeID uuid.UUID
	var currentPos int
	var lft int
	if err := tx.QueryRow(ctx, `
		SELECT composite_id, root_position, lft FROM entities WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(&compositeID, &currentPos, &lft); err != nil {
		return fmt.Errorf("failed to find entity: %w", err)
	}
	if lft != 1 {
		return fmt.Errorf("entity is not a root node")
	}

	var targetPos int
	if afterID == nil {
		targetPos = 1
	} else {
		var afterCompositeID uuid.UUID
		var afterLft int
		if err := tx.QueryRow(ctx, `
			SELECT composite_id, root_position, lft FROM entities WHERE id = $1 AND deleted_at IS NULL
		`, *afterID).Scan(&afterCompositeID, &targetPos, &afterLft); err != nil {
			return fmt.Errorf("failed to find after entity: %w", err)
		}
		if afterCompositeID != compositeID {
			return fmt.Errorf("after entity does not belong to the same composite")
		}
		if afterLft != 1 {
			return fmt.Errorf("after entity is not a root node")
		}
		targetPos++ // insert after
	}

	if currentPos == targetPos {
		return nil
	}

	// Step 1: temporarily park the moving node
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET root_position = -(root_position) WHERE id = $1
	`, id); err != nil {
		return fmt.Errorf("failed to park root node: %w", err)
	}

	if currentPos > targetPos {
		// Step 2a: negate range [targetPos, currentPos-1]
		if _, err := tx.Exec(ctx, `
			UPDATE entities SET root_position = -(root_position)
			WHERE composite_id = $1 AND lft = 1 AND deleted_at IS NULL
			AND root_position >= $2 AND root_position < $3
		`, compositeID, targetPos, currentPos); err != nil {
			return fmt.Errorf("failed to negate range: %w", err)
		}
		// Step 2b: denegate + apply +1 (WHERE targets the now-negative range, excludes parked node)
		if _, err := tx.Exec(ctx, `
			UPDATE entities SET root_position = -(root_position) + 1
			WHERE composite_id = $1 AND lft = 1 AND deleted_at IS NULL
			AND root_position > $2 AND root_position <= $3
		`, compositeID, -currentPos, -targetPos); err != nil {
			return fmt.Errorf("failed to shift range: %w", err)
		}
	} else {
		// Step 2a: negate range [currentPos+1, targetPos]
		if _, err := tx.Exec(ctx, `
			UPDATE entities SET root_position = -(root_position)
			WHERE composite_id = $1 AND lft = 1 AND deleted_at IS NULL
			AND root_position > $2 AND root_position <= $3
		`, compositeID, currentPos, targetPos); err != nil {
			return fmt.Errorf("failed to negate range: %w", err)
		}
		// Step 2b: denegate + apply -1
		if _, err := tx.Exec(ctx, `
			UPDATE entities SET root_position = -(root_position) - 1
			WHERE composite_id = $1 AND lft = 1 AND deleted_at IS NULL
			AND root_position >= $2 AND root_position < $3
		`, compositeID, -targetPos, -currentPos); err != nil {
			return fmt.Errorf("failed to shift range: %w", err)
		}
	}

	// Step 3: place the node at its final position
	if _, err := tx.Exec(ctx, `
		UPDATE entities SET root_position = $1 WHERE id = $2
	`, targetPos, id); err != nil {
		return fmt.Errorf("failed to update root position: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *entityRepository) Update(ctx context.Context, entity *models.Entity) (*models.Entity, error) {
	entity.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE entities
		SET name = $1, slug = $2, updated_at = $3, updated_by = $4
		WHERE id = $5 AND deleted_at IS NULL
	`, entity.Name, entity.Slug, entity.UpdatedAt, entity.UpdatedBy, entity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity: %w", err)
	} else if result.RowsAffected() == 0 {
		return nil, ErrNoRowsAffected
	}

	return entity, nil
}

func (r *entityRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	result, err := r.db.Exec(ctx, `
		UPDATE entities SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	} else if result.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}

	return nil
}
