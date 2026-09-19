-- +goose Up

ALTER TABLE orders
ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255) NOT NULL;

CREATE UNIQUE INDEX idx_uniq_order_ikey_user ON orders (user_id, idempotency_key);

-- +goose Down

DROP INDEX IF EXISTS idx_uniq_order_ikey_user;

ALTER TABLE orders DROP COLUMN IF EXISTS idempotency_key;