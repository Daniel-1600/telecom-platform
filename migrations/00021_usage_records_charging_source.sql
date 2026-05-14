-- +goose Up
ALTER TABLE usage_records
    ADD COLUMN IF NOT EXISTS charging_source TEXT DEFAULT 'credit',
    ADD COLUMN IF NOT EXISTS subscriber_bundle_id INTEGER REFERENCES subscriber_bundles(id);

CREATE INDEX IF NOT EXISTS idx_usage_records_charging_source ON usage_records(charging_source);

-- +goose Down
ALTER TABLE usage_records
    DROP COLUMN IF EXISTS charging_source,
    DROP COLUMN IF EXISTS subscriber_bundle_id;
