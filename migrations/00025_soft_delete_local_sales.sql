-- +goose Up
ALTER TABLE local_sales ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL;

-- +goose Down
ALTER TABLE local_sales DROP COLUMN deleted_at;