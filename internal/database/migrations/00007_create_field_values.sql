-- +goose Up
CREATE TABLE field_values (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id  UUID        NOT NULL REFERENCES entities(id),
    field_id   UUID        NOT NULL REFERENCES fields(id),
    value      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    created_by UUID,
    updated_by UUID,
    deleted_by UUID,
    UNIQUE (entity_id, field_id)
);

-- +goose Down
DROP TABLE field_values;
