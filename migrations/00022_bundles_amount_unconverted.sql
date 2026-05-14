-- +goose Up
ALTER TABLE bundles
    ADD COLUMN IF NOT EXISTS amount_unconverted BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE bundles DROP COLUMN IF EXISTS amount_unconverted;
