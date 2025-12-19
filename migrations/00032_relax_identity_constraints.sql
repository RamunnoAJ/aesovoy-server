-- +goose Up
-- +goose StatementBegin
-- Clients
ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_reference_key;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_cuit_key;
ALTER TABLE clients ALTER COLUMN reference DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN cuit DROP NOT NULL;

-- Providers
ALTER TABLE providers DROP CONSTRAINT IF EXISTS providers_reference_key;
ALTER TABLE providers DROP CONSTRAINT IF EXISTS providers_cuit_key;
ALTER TABLE providers ALTER COLUMN reference DROP NOT NULL;
ALTER TABLE providers ALTER COLUMN cuit DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Note: Down migration might fail if duplicates were introduced
ALTER TABLE clients ADD CONSTRAINT clients_reference_key UNIQUE (reference);
ALTER TABLE clients ADD CONSTRAINT clients_cuit_key UNIQUE (cuit);
ALTER TABLE clients ALTER COLUMN reference SET NOT NULL;
ALTER TABLE clients ALTER COLUMN cuit SET NOT NULL;

ALTER TABLE providers ADD CONSTRAINT providers_reference_key UNIQUE (reference);
ALTER TABLE providers ADD CONSTRAINT providers_cuit_key UNIQUE (cuit);
ALTER TABLE providers ALTER COLUMN reference SET NOT NULL;
ALTER TABLE providers ALTER COLUMN cuit SET NOT NULL;
-- +goose StatementEnd
