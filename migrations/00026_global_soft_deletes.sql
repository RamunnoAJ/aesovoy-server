-- +goose Up
-- Add deleted_at column to tables
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE categories ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE products ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE clients ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE providers ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE orders ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE ingredients ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;
ALTER TABLE payment_methods ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

-- Drop old unique constraints and replace with partial unique indexes (where deleted_at IS NULL)

-- Users
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
CREATE UNIQUE INDEX users_username_idx ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_email_idx ON users (email) WHERE deleted_at IS NULL;

-- Categories
ALTER TABLE categories DROP CONSTRAINT IF EXISTS categories_name_key;
CREATE UNIQUE INDEX categories_name_idx ON categories (name) WHERE deleted_at IS NULL;

-- Products
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_key;
CREATE UNIQUE INDEX products_name_idx ON products (name) WHERE deleted_at IS NULL;

-- Ingredients
ALTER TABLE ingredients DROP CONSTRAINT IF EXISTS ingredients_name_key;
CREATE UNIQUE INDEX ingredients_name_idx ON ingredients (name) WHERE deleted_at IS NULL;

-- +goose Down
-- Drop partial indexes and restore constraints (This might fail if duplicates exist, but it's best effort for Down)
DROP INDEX IF EXISTS users_username_idx;
DROP INDEX IF EXISTS users_email_idx;
DROP INDEX IF EXISTS categories_name_idx;
DROP INDEX IF EXISTS products_name_idx;
DROP INDEX IF EXISTS ingredients_name_idx;

ALTER TABLE users ADD CONSTRAINT users_username_key UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE categories ADD CONSTRAINT categories_name_key UNIQUE (name);
ALTER TABLE products ADD CONSTRAINT products_name_key UNIQUE (name);
ALTER TABLE ingredients ADD CONSTRAINT ingredients_name_key UNIQUE (name);

ALTER TABLE users DROP COLUMN deleted_at;
ALTER TABLE categories DROP COLUMN deleted_at;
ALTER TABLE products DROP COLUMN deleted_at;
ALTER TABLE clients DROP COLUMN deleted_at;
ALTER TABLE providers DROP COLUMN deleted_at;
ALTER TABLE orders DROP COLUMN deleted_at;
ALTER TABLE ingredients DROP COLUMN deleted_at;
ALTER TABLE payment_methods DROP COLUMN deleted_at;
