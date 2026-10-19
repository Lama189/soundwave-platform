-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_user_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_payment',
    total_amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    payment_id UUID,
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_orders_total_amount_positive CHECK (total_amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_orders_buyer_user_id ON orders(buyer_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_payment_id ON orders(payment_id) WHERE payment_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    item_type VARCHAR(32) NOT NULL,
    item_id UUID NOT NULL,
    artist_id UUID NOT NULL,
    price BIGINT NOT NULL,
    platform_fee BIGINT NOT NULL,
    artist_share BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    CONSTRAINT chk_order_items_price CHECK (price > 0),
    CONSTRAINT chk_order_items_shares CHECK (price = platform_fee + artist_share)
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_artist_id ON order_items(artist_id);
CREATE INDEX IF NOT EXISTS idx_order_items_item ON order_items(item_type, item_id);

CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id UUID NOT NULL UNIQUE,
    currency VARCHAR(3) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    locked_balance BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_wallets_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT chk_wallets_locked_balance_non_negative CHECK (locked_balance >= 0)
);

CREATE INDEX IF NOT EXISTS idx_wallets_artist_id ON wallets(artist_id);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    operation_type VARCHAR(32) NOT NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    reference_type VARCHAR(32) NOT NULL,
    reference_id UUID NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_wallet_id ON ledger_entries(wallet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_reference ON ledger_entries(reference_type, reference_id);

CREATE TABLE IF NOT EXISTS payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    artist_id UUID NOT NULL,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    destination_card VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    failure_reason TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    CONSTRAINT chk_payouts_amount_positive CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_payouts_artist_id ON payouts(artist_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_payouts_wallet_id ON payouts(wallet_id);
CREATE INDEX IF NOT EXISTS idx_payouts_status ON payouts(status);

CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(64) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_billing_outbox_unprocessed ON outbox_events(created_at ASC) WHERE processed_at IS NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
-- +goose StatementEnd
