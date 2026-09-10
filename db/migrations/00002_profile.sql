-- +goose Up
ALTER TABLE users ADD COLUMN display_name text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN display_name;
