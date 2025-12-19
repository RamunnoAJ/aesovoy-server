-- +goose Up
CREATE TABLE cash_movements (
    id SERIAL PRIMARY KEY,
    shift_id INT NOT NULL REFERENCES shifts(id),
    amount NUMERIC(15, 2) NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('in', 'out')),
    reason TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cash_movements_shift_id ON cash_movements(shift_id);

-- +goose Down
DROP TABLE cash_movements;
