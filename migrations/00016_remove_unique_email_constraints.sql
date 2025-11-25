-- +goose Up
-- +goose StatementBegin
ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_email_key;
ALTER TABLE providers DROP CONSTRAINT IF EXISTS providers_email_key;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE clients ADD CONSTRAINT clients_email_key UNIQUE (email);
ALTER TABLE providers ADD CONSTRAINT providers_email_key UNIQUE (email);
-- +goose StatementEnd
