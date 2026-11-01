-- +goose Up
ALTER TABLE customers ADD COLUMN password TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE customers DROP COLUMN password;
