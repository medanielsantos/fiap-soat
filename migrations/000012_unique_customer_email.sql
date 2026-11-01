-- +goose Up
CREATE UNIQUE INDEX ux_customers_email ON customers (lower(email)) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX ux_customers_email;
