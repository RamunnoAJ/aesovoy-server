-- +goose Up
CREATE TABLE expenses (
    id SERIAL PRIMARY KEY,
    amount NUMERIC(15, 2) NOT NULL,
    image_path TEXT,
    provider_id INT REFERENCES providers(id),
    category TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('local', 'production')),
    date TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_expenses_provider_id ON expenses(provider_id);
CREATE INDEX idx_expenses_type ON expenses(type);
CREATE INDEX idx_expenses_date ON expenses(date);

-- +goose Down
DROP TABLE expenses;
