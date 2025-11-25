-- +goose Up
CREATE TABLE local_sales (
    id SERIAL PRIMARY KEY,
    payment_method_id BIGINT NOT NULL,
    subtotal NUMERIC(10, 2) NOT NULL,
    total NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_payment_method
        FOREIGN KEY(payment_method_id)
        REFERENCES payment_methods(id)
);

CREATE TABLE local_sale_items (
    id SERIAL PRIMARY KEY,
    local_sale_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    quantity INT NOT NULL,
    unit_price NUMERIC(10, 2) NOT NULL,
    line_subtotal NUMERIC(10, 2) NOT NULL,
    CONSTRAINT fk_local_sale
        FOREIGN KEY(local_sale_id)
        REFERENCES local_sales(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_product
        FOREIGN KEY(product_id)
        REFERENCES products(id)
);

CREATE INDEX idx_local_sale_items_sale_id ON local_sale_items(local_sale_id);
CREATE INDEX idx_local_sale_items_product_id ON local_sale_items(product_id);


-- +goose Down
DROP TABLE IF EXISTS local_sale_items;
DROP TABLE IF EXISTS local_sales;
