-- +goose Up
CREATE TABLE permissions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id      UUID        NOT NULL REFERENCES roles(id),
    composite_id UUID        NOT NULL REFERENCES composites(id),
    can_read     BOOLEAN     NOT NULL DEFAULT FALSE,
    can_write    BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    created_by   UUID,
    updated_by   UUID,
    deleted_by   UUID,
    UNIQUE (role_id, composite_id)
);

-- +goose Down
DROP TABLE permissions;
