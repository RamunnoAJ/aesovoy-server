-- +goose Up
-- +goose StatementBegin
-- Add 'paid' to the state enum
ALTER TYPE state ADD VALUE 'paid';

-- Add payment_method_id to orders
ALTER TABLE orders ADD COLUMN payment_method_id BIGINT REFERENCES payment_methods(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Note: Removing a value from an ENUM is complex in Postgres. 
-- Usually involve creating a new type, updating columns, and dropping the old one.
-- For a development/test environment, we might just leave it or handle it manually if needed.
ALTER TABLE orders DROP COLUMN IF EXISTS payment_method_id;
-- +goose StatementEnd
