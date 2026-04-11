-- +goose Up
CREATE TYPE field_type AS ENUM (
    'association',
    'checkbox',
    'color',
    'date',
    'datetime',
    'dropdown',
    'email',
    'file',
    'long_text',
    'number',
    'phone',
    'short_text',
    'time',
    'url'
);

CREATE TABLE fields (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    composite_id  UUID        NOT NULL REFERENCES composites(id),
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL,
    type          field_type  NOT NULL,
    required      BOOLEAN     NOT NULL DEFAULT FALSE,
    position      INT         NOT NULL DEFAULT 0,
    default_value TEXT,
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    created_by    UUID,
    updated_by    UUID,
    deleted_by    UUID,
    UNIQUE (composite_id, slug)
);

-- +goose Down
DROP TABLE fields;
DROP TYPE field_type;
