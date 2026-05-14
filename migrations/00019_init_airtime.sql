-- +goose Up
CREATE TABLE IF NOT EXISTS airtime_balances (
    id              SERIAL PRIMARY KEY,
    subscriber_id   INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    imsi            TEXT NOT NULL UNIQUE,
    home_seconds    BIGINT NOT NULL DEFAULT 0,
    roaming_seconds BIGINT NOT NULL DEFAULT 0,
    last_topped_up  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS airtime_transactions (
    id              SERIAL PRIMARY KEY,
    subscriber_id   INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    imsi            TEXT NOT NULL,
    transaction_type TEXT NOT NULL,   -- 'topup', 'deduction', 'bundle_purchase'
    seconds_delta   BIGINT NOT NULL,  -- positive = credit, negative = debit
    balance_after   BIGINT NOT NULL,
    roaming         BOOLEAN NOT NULL DEFAULT FALSE,
    reference_id    TEXT,             -- bundle_id if bundle_purchase, session_id if deduction
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_airtime_balances_imsi ON airtime_balances(imsi);
CREATE INDEX IF NOT EXISTS idx_airtime_transactions_imsi ON airtime_transactions(imsi);
CREATE INDEX IF NOT EXISTS idx_airtime_transactions_type ON airtime_transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_airtime_transactions_created_at ON airtime_transactions(created_at);

-- +goose Down
DROP TABLE IF EXISTS airtime_transactions CASCADE;
DROP TABLE IF EXISTS airtime_balances CASCADE;
