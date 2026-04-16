-- +goose Up
ALTER TABLE permissions
    ALTER COLUMN composite_id DROP NOT NULL,
    ADD COLUMN resource_type VARCHAR(50);

ALTER TABLE permissions
    DROP CONSTRAINT permissions_role_id_composite_id_key;

ALTER TABLE permissions
    ADD CONSTRAINT permissions_composite_id_or_resource_type CHECK (
        (composite_id IS NOT NULL AND resource_type IS NULL) OR
        (composite_id IS NULL AND resource_type IS NOT NULL)
    );

ALTER TABLE permissions
    ADD CONSTRAINT permissions_resource_type_check CHECK (
        resource_type IN ('composite', 'field', 'user', 'role')
    );

CREATE UNIQUE INDEX permissions_role_composite_unique
    ON permissions (role_id, composite_id)
    WHERE composite_id IS NOT NULL;

CREATE UNIQUE INDEX permissions_role_resource_unique
    ON permissions (role_id, resource_type)
    WHERE resource_type IS NOT NULL;

-- +goose Down
DROP INDEX permissions_role_resource_unique;
DROP INDEX permissions_role_composite_unique;

ALTER TABLE permissions
    DROP CONSTRAINT permissions_resource_type_check,
    DROP CONSTRAINT permissions_composite_id_or_resource_type,
    DROP COLUMN resource_type,
    ALTER COLUMN composite_id SET NOT NULL;

ALTER TABLE permissions
    ADD CONSTRAINT permissions_role_id_composite_id_key UNIQUE (role_id, composite_id);
