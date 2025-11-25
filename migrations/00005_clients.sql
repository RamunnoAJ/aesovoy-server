-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'client_type') THEN
        CREATE TYPE client_type AS ENUM ('distributer', 'individual');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS clients (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL,
  address VARCHAR(100),
  phone VARCHAR(100),
  reference VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(100) UNIQUE,
  cuit VARCHAR(50) UNIQUE NOT NULL,
  type client_type NOT NULL DEFAULT 'individual',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE clients;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_type WHERE typname = 'client_type') THEN
        DROP TYPE client_type;
    END IF;
END$$;
-- +goose StatementEnd
