-- +goose Up

ALTER TABLE orders
ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255) NOT NULL UNIQUE;

-- +goose Down

ALTER TABLE orders DROP COLUMN IF EXISTS idempotency_key;