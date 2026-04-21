-- +goose Up
ALTER TABLE roles DROP CONSTRAINT roles_name_key;
CREATE UNIQUE INDEX roles_name_unique ON roles (name) WHERE deleted_at IS NULL;

ALTER TABLE composites DROP CONSTRAINT composites_slug_key;
CREATE UNIQUE INDEX composites_slug_unique ON composites (slug) WHERE deleted_at IS NULL;

ALTER TABLE users DROP CONSTRAINT users_email_key;
CREATE UNIQUE INDEX users_email_unique ON users (email) WHERE deleted_at IS NULL;

ALTER TABLE fields DROP CONSTRAINT fields_composite_id_slug_key;
CREATE UNIQUE INDEX fields_composite_id_slug_unique ON fields (composite_id, slug) WHERE deleted_at IS NULL;

ALTER TABLE entities DROP CONSTRAINT entities_composite_id_slug_key;
CREATE UNIQUE INDEX entities_composite_id_slug_unique ON entities (composite_id, slug) WHERE deleted_at IS NULL;

ALTER TABLE field_values DROP CONSTRAINT field_values_entity_id_field_id_key;
CREATE UNIQUE INDEX field_values_entity_id_field_id_unique ON field_values (entity_id, field_id) WHERE deleted_at IS NULL;

DROP INDEX permissions_role_composite_unique;
CREATE UNIQUE INDEX permissions_role_composite_unique
    ON permissions (role_id, composite_id)
    WHERE composite_id IS NOT NULL AND deleted_at IS NULL;

DROP INDEX permissions_role_resource_unique;
CREATE UNIQUE INDEX permissions_role_resource_unique
    ON permissions (role_id, resource_type)
    WHERE resource_type IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP INDEX permissions_role_resource_unique;
CREATE UNIQUE INDEX permissions_role_resource_unique
    ON permissions (role_id, resource_type)
    WHERE resource_type IS NOT NULL;

DROP INDEX permissions_role_composite_unique;
CREATE UNIQUE INDEX permissions_role_composite_unique
    ON permissions (role_id, composite_id)
    WHERE composite_id IS NOT NULL;

DROP INDEX field_values_entity_id_field_id_unique;
ALTER TABLE field_values ADD CONSTRAINT field_values_entity_id_field_id_key UNIQUE (entity_id, field_id);

DROP INDEX entities_composite_id_slug_unique;
ALTER TABLE entities ADD CONSTRAINT entities_composite_id_slug_key UNIQUE (composite_id, slug);

DROP INDEX fields_composite_id_slug_unique;
ALTER TABLE fields ADD CONSTRAINT fields_composite_id_slug_key UNIQUE (composite_id, slug);

DROP INDEX users_email_unique;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);

DROP INDEX composites_slug_unique;
ALTER TABLE composites ADD CONSTRAINT composites_slug_key UNIQUE (slug);

DROP INDEX roles_name_unique;
ALTER TABLE roles ADD CONSTRAINT roles_name_key UNIQUE (name);
