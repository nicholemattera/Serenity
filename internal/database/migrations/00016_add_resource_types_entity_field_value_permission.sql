-- +goose Up
ALTER TABLE permissions
    DROP CONSTRAINT permissions_resource_type_check;

ALTER TABLE permissions
    ADD CONSTRAINT permissions_resource_type_check CHECK (
        resource_type IN ('composite', 'field', 'user', 'role', 'entity', 'field_value', 'permission')
    );

-- +goose Down
ALTER TABLE permissions
    DROP CONSTRAINT permissions_resource_type_check;

ALTER TABLE permissions
    ADD CONSTRAINT permissions_resource_type_check CHECK (
        resource_type IN ('composite', 'field', 'user', 'role')
    );
