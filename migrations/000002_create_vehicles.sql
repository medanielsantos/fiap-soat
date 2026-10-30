-- +goose Up
CREATE TABLE vehicles (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customers (id),
    plate VARCHAR(7) NOT NULL,
    brand TEXT NOT NULL,
    model TEXT NOT NULL,
    year INTEGER NOT NULL,
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT vehicles_plate_length CHECK (length(plate) = 7),
    CONSTRAINT vehicles_year CHECK (year >= 1900)
);

CREATE UNIQUE INDEX ux_vehicles_plate ON vehicles (plate) WHERE deleted_at IS NULL;
CREATE INDEX ix_vehicles_customer_id ON vehicles (customer_id);
CREATE INDEX ix_vehicles_deleted_at ON vehicles (deleted_at);

-- +goose Down
DROP TABLE vehicles;
