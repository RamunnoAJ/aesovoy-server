-- +goose Up
-- +goose StatementBegin
ALTER TABLE products
  ALTER COLUMN unit_price TYPE NUMERIC(12,2) USING unit_price::numeric(12,2),
  ALTER COLUMN distribution_price TYPE NUMERIC(12,2) USING distribution_price::numeric(12,2);

ALTER TABLE orders
  ALTER COLUMN total TYPE NUMERIC(12,2) USING total::numeric(12,2);

ALTER TABLE order_products 
  ALTER COLUMN price TYPE NUMERIC(12,2) USING price::numeric(12,2);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products
  ALTER COLUMN unit_price TYPE REAL USING unit_price::real,
  ALTER COLUMN distribution_price TYPE REAL USING distribution_price::real;

ALTER TABLE orders
  ALTER COLUMN total TYPE REAL USING total::real;

ALTER TABLE order_products 
  ALTER COLUMN price TYPE REAL USING price::real;
-- +goose StatementEnd
