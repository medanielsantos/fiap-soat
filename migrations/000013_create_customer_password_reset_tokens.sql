-- +goose Up
CREATE TABLE customer_password_reset_tokens (
    token_hash CHAR(64) PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_customer_password_reset_customer ON customer_password_reset_tokens(customer_id);

-- +goose Down
DROP TABLE customer_password_reset_tokens;
