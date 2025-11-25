-- +goose Up
CREATE TABLE local_stock (
    id SERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL UNIQUE,
    quantity INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_product
        FOREIGN KEY(product_id)
        REFERENCES products(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_local_stock_product_id ON local_stock(product_id);

-- +goose Down
DROP TABLE IF EXISTS local_stock;
