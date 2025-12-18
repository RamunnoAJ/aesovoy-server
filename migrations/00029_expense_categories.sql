-- +goose Up
CREATE TABLE expense_categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Insert default categories if needed, or migrate existing ones
INSERT INTO expense_categories (name)
SELECT DISTINCT category FROM expenses WHERE category IS NOT NULL AND category != '';

-- Add category_id to expenses
ALTER TABLE expenses ADD COLUMN category_id INT REFERENCES expense_categories(id);

-- Populate category_id based on existing text
UPDATE expenses e
SET category_id = ec.id
FROM expense_categories ec
WHERE e.category = ec.name;

-- Make category_id not null if we want to enforce it (optional but good practice)
-- First ensure no nulls (default category?)
-- For now, let's keep it nullable or flexible until migration is verified.
-- But user wants strict category management.

-- Drop old text column
ALTER TABLE expenses DROP COLUMN category;

-- +goose Down
ALTER TABLE expenses ADD COLUMN category TEXT;

UPDATE expenses e
SET category = ec.name
FROM expense_categories ec
WHERE e.category_id = ec.id;

ALTER TABLE expenses DROP COLUMN category_id;
DROP TABLE expense_categories;
