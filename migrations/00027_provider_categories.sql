-- +goose Up
CREATE TABLE provider_categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO provider_categories (name) VALUES ('Sin Categoría');

ALTER TABLE providers ADD COLUMN category_id INT REFERENCES provider_categories(id) DEFAULT 1;

-- +goose Down
ALTER TABLE providers DROP COLUMN category_id;
DROP TABLE provider_categories;
