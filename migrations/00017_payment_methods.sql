-- +goose Up
CREATE TABLE payment_methods (
    id SERIAL PRIMARY KEY,
    owner VARCHAR(255) NOT NULL,
    reference VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS payment_methods;
