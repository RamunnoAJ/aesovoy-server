-- +goose Up
ALTER TABLE payment_methods DROP COLUMN owner;

-- +goose Down
ALTER TABLE payment_methods ADD COLUMN owner VARCHAR(255) NOT NULL DEFAULT '';
