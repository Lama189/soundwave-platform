-- +goose Up
-- +goose StatementBegin

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

CREATE TABLE IF NOT EXISTS processed_events (
    event_id UUID PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_type ON processed_events(event_type);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS payouts;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS wallets;
-- +goose StatementEnd
