-- +goose Up
CREATE TABLE catalog_services (
    id TEXT PRIMARY KEY,
    internal_code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sale_price_in_cents BIGINT NOT NULL CHECK (sale_price_in_cents > 0),
    estimated_duration_minutes INTEGER NOT NULL CHECK (estimated_duration_minutes > 0),
    created_by TEXT,
    updated_by TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX ix_catalog_services_name ON catalog_services (name);
CREATE UNIQUE INDEX ux_catalog_services_internal_code ON catalog_services (internal_code) WHERE deleted_at IS NULL;

CREATE TABLE parts (
    id TEXT PRIMARY KEY,
    sku TEXT NOT NULL,
    internal_code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    unit TEXT NOT NULL,
    purchase_price_in_cents BIGINT NOT NULL DEFAULT 0 CHECK (purchase_price_in_cents >= 0),
    sale_price_in_cents BIGINT NOT NULL CHECK (sale_price_in_cents > 0),
    stock BIGINT NOT NULL DEFAULT 0 CHECK (stock >= 0),
    created_by TEXT,
    updated_by TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX ix_parts_name ON parts (name);
CREATE UNIQUE INDEX ux_parts_sku ON parts (sku) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX ux_parts_internal_code ON parts (internal_code) WHERE deleted_at IS NULL;

CREATE TABLE stock_movements (
    id TEXT PRIMARY KEY,
    part_id TEXT NOT NULL REFERENCES parts (id),
    movement_type VARCHAR(8) NOT NULL CHECK (movement_type IN ('entry', 'exit')),
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    description TEXT NOT NULL DEFAULT '',
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ix_stock_movements_part_created ON stock_movements (part_id, created_at DESC);

-- +goose Down
DROP TABLE stock_movements;
DROP TABLE parts;
DROP TABLE catalog_services;
