-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS order_products (
  id BIGSERIAL PRIMARY KEY NOT NULL,
  quantity INTEGER,
  price REAL,
  product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS order_products;
-- +goose StatementEnd
