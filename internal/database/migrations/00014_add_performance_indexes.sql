-- +goose Up
CREATE INDEX field_values_entity_id ON field_values (entity_id);
CREATE INDEX entities_composite_id ON entities (composite_id);
CREATE INDEX fields_composite_id ON fields (composite_id);
CREATE INDEX entities_tree_id_lft_rgt ON entities (tree_id, lft, rgt);

-- +goose Down
DROP INDEX IF EXISTS field_values_entity_id;
DROP INDEX IF EXISTS entities_composite_id;
DROP INDEX IF EXISTS fields_composite_id;
DROP INDEX IF EXISTS entities_tree_id_lft_rgt;
