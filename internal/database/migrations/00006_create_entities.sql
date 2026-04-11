-- +goose Up
CREATE TABLE entities (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    composite_id UUID         NOT NULL REFERENCES composites(id),
    name         VARCHAR(255) NOT NULL,
    slug         VARCHAR(255) NOT NULL,
    lft          INT          NOT NULL,
    rgt          INT          NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,
    created_by   UUID,
    updated_by   UUID,
    deleted_by   UUID,
    UNIQUE (composite_id, slug)
);

-- +goose Down
DROP TABLE entities;
