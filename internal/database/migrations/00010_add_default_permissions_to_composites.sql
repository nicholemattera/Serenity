-- +goose Up
ALTER TABLE composites
    ADD COLUMN default_read  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN default_write BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE composites
    DROP COLUMN default_read,
    DROP COLUMN default_write;
