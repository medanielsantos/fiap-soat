-- +goose Up
CREATE TABLE work_orders (
    id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customers(id),
    vehicle_id TEXT NOT NULL REFERENCES vehicles(id),
    status VARCHAR(24) NOT NULL CHECK (status IN ('received','diagnosis','awaiting_approval','in_progress','completed','delivered')),
    diagnosis TEXT NOT NULL DEFAULT '',
    quote_version INTEGER NOT NULL DEFAULT 0 CHECK (quote_version >= 0),
    created_by TEXT,
    updated_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX ix_work_orders_customer ON work_orders(customer_id, created_at DESC);
CREATE INDEX ix_work_orders_vehicle ON work_orders(vehicle_id, created_at DESC);
CREATE INDEX ix_work_orders_status ON work_orders(status);

CREATE TABLE work_order_items (
    id TEXT PRIMARY KEY,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id),
    item_type VARCHAR(12) NOT NULL CHECK (item_type IN ('service','part')),
    catalog_id TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT NOT NULL,
    unit TEXT NOT NULL DEFAULT 'service',
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    unit_price_in_cents BIGINT NOT NULL CHECK (unit_price_in_cents > 0),
    estimated_duration_minutes INTEGER NOT NULL DEFAULT 0 CHECK (estimated_duration_minutes >= 0),
    consumed_quantity BIGINT NOT NULL DEFAULT 0 CHECK (consumed_quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT work_order_items_consumed_lte_quantity CHECK (consumed_quantity <= quantity)
);
CREATE INDEX ix_work_order_items_order ON work_order_items(work_order_id);

CREATE TABLE work_order_quotes (
    id TEXT PRIMARY KEY,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id),
    version INTEGER NOT NULL CHECK (version > 0),
    total_in_cents BIGINT NOT NULL CHECK (total_in_cents >= 0),
    decision VARCHAR(12) NOT NULL DEFAULT 'pending' CHECK (decision IN ('pending','approved','rejected')),
    created_by TEXT,
    decided_by_client_id TEXT,
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(work_order_id, version)
);
CREATE TABLE work_order_quote_items (
    id TEXT PRIMARY KEY,
    quote_id TEXT NOT NULL REFERENCES work_order_quotes(id),
    item_type VARCHAR(12) NOT NULL CHECK (item_type IN ('service','part')),
    catalog_id TEXT NOT NULL,
    code TEXT NOT NULL,
    description TEXT NOT NULL,
    unit TEXT NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    unit_price_in_cents BIGINT NOT NULL CHECK (unit_price_in_cents > 0),
    subtotal_in_cents BIGINT NOT NULL CHECK (subtotal_in_cents >= 0),
    estimated_duration_minutes INTEGER NOT NULL DEFAULT 0 CHECK (estimated_duration_minutes >= 0)
);
CREATE INDEX ix_work_order_quotes_order ON work_order_quotes(work_order_id, version DESC);

CREATE TABLE work_order_status_history (
    id TEXT PRIMARY KEY,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id),
    from_status VARCHAR(24),
    to_status VARCHAR(24) NOT NULL,
    user_id TEXT,
    client_id TEXT,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT work_order_history_single_principal CHECK (
        num_nonnulls(user_id, client_id) = 1
    )
);
CREATE INDEX ix_work_order_history_order ON work_order_status_history(work_order_id, created_at);

ALTER TABLE parts ADD COLUMN reserved_stock BIGINT NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0 AND reserved_stock <= stock);
ALTER TABLE stock_movements ADD COLUMN work_order_id TEXT REFERENCES work_orders(id);
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_movement_type_check;
ALTER TABLE stock_movements ALTER COLUMN movement_type TYPE VARCHAR(12);
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_movement_type_check CHECK (movement_type IN ('entry','exit','reservation','release','consumption'));
CREATE INDEX ix_stock_movements_work_order ON stock_movements(work_order_id) WHERE work_order_id IS NOT NULL;

-- +goose Down
DROP INDEX ix_stock_movements_work_order;
DELETE FROM stock_movements WHERE movement_type IN ('reservation','release','consumption');
ALTER TABLE stock_movements DROP CONSTRAINT stock_movements_movement_type_check;
ALTER TABLE stock_movements ADD CONSTRAINT stock_movements_movement_type_check CHECK (movement_type IN ('entry','exit'));
ALTER TABLE stock_movements ALTER COLUMN movement_type TYPE VARCHAR(8);
ALTER TABLE stock_movements DROP COLUMN work_order_id;
ALTER TABLE parts DROP COLUMN reserved_stock;
DROP TABLE work_order_status_history;
DROP TABLE work_order_quote_items;
DROP TABLE work_order_quotes;
DROP TABLE work_order_items;
DROP TABLE work_orders;
