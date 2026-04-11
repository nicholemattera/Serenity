-- +goose Up
ALTER TABLE entities ADD COLUMN root_position INT;

-- Existing root nodes (lft = 1) get sequential positions per composite
UPDATE entities
SET root_position = sub.rn
FROM (
    SELECT id, ROW_NUMBER() OVER (PARTITION BY composite_id ORDER BY lft) AS rn
    FROM entities
    WHERE lft = 1 AND deleted_at IS NULL
) sub
WHERE entities.id = sub.id;

-- +goose Down
ALTER TABLE entities DROP COLUMN root_position;
