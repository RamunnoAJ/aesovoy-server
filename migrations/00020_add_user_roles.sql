-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'employee';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN role;
-- +goose StatementEnd
