-- +goose Up
ALTER TABLE entities ADD COLUMN tree_id UUID;

-- Existing root nodes self-reference
UPDATE entities SET tree_id = id;

ALTER TABLE entities ALTER COLUMN tree_id SET NOT NULL;

CREATE INDEX entities_tree_id_idx ON entities (tree_id);

-- +goose Down
DROP INDEX entities_tree_id_idx;
ALTER TABLE entities DROP COLUMN tree_id;
