-- +goose Up
ALTER TABLE payment_methods ADD COLUMN name VARCHAR(255);
UPDATE payment_methods SET name = owner WHERE name IS NULL; -- Default name to owner for existing
ALTER TABLE payment_methods ALTER COLUMN name SET NOT NULL;

-- +goose Down
ALTER TABLE payment_methods DROP COLUMN name;
