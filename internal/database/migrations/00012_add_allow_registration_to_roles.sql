-- +goose Up
ALTER TABLE roles ADD COLUMN allow_registration BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE roles DROP COLUMN allow_registration;
