-- +goose Up
CREATE UNIQUE INDEX entities_composite_id_root_position_unique ON entities (composite_id, root_position) WHERE lft = 1 AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS entities_composite_id_root_position_unique;
