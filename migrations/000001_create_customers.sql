-- +goose Up
CREATE TABLE customers (
    id TEXT PRIMARY KEY,
    document VARCHAR(14) NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT NOT NULL,
    status VARCHAR(8) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT customers_document_length CHECK (length(document) IN (11, 14)),
    CONSTRAINT customers_status CHECK (status IN ('active', 'inactive'))
);

CREATE UNIQUE INDEX ux_customers_document ON customers (document);

-- +goose Down
DROP TABLE customers;
