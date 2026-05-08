-- +goose Up
CREATE TABLE IF NOT EXISTS bundles (
    id              SERIAL PRIMARY KEY,
    bundle_id       TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    bundle_type     TEXT NOT NULL,        -- 'data', 'voice', 'sms', 'hybrid'
    data_bytes      BIGINT,
    voice_seconds   BIGINT,
    sms_count       BIGINT,
    roaming_data_bytes BIGINT,
    validity_days   INTEGER NOT NULL DEFAULT 30,
    priority        SMALLINT NOT NULL DEFAULT 1,
    airtime_cost    BIGINT NOT NULL DEFAULT 0,  -- cost in airtime seconds to purchase
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscriber_bundles (
    id                  SERIAL PRIMARY KEY,
    subscriber_id       INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    imsi                TEXT NOT NULL,
    bundle_id           TEXT NOT NULL REFERENCES bundles(bundle_id),
    activated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    remaining_data_bytes      BIGINT,
    remaining_voice_seconds   BIGINT,
    remaining_sms_count       BIGINT,
    remaining_roaming_bytes   BIGINT,
    status              TEXT NOT NULL DEFAULT 'active',  -- 'active', 'exhausted', 'expired'
    purchased_with_airtime BOOLEAN NOT NULL DEFAULT FALSE,
    airtime_cost_paid   BIGINT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bundles_bundle_id ON bundles(bundle_id);
CREATE INDEX IF NOT EXISTS idx_bundles_is_active ON bundles(is_active);
CREATE INDEX IF NOT EXISTS idx_subscriber_bundles_imsi ON subscriber_bundles(imsi);
CREATE INDEX IF NOT EXISTS idx_subscriber_bundles_expires_at ON subscriber_bundles(expires_at);
CREATE INDEX IF NOT EXISTS idx_subscriber_bundles_status ON subscriber_bundles(status);

-- +goose Down
DROP TABLE IF EXISTS subscriber_bundles CASCADE;
DROP TABLE IF EXISTS bundles CASCADE;
