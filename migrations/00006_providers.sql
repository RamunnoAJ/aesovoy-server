-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS providers (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL,
  address VARCHAR(100),
  phone VARCHAR(100),
  reference VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(100) UNIQUE,
  cuit VARCHAR(50) UNIQUE NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE providers;
-- +goose StatementEnd
