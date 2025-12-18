-- +goose Up
CREATE TABLE shifts (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id),
    start_time TIMESTAMP NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP,
    start_cash NUMERIC(15, 2) NOT NULL DEFAULT 0,
    end_cash_expected NUMERIC(15, 2),
    end_cash_declared NUMERIC(15, 2),
    difference NUMERIC(15, 2),
    status TEXT NOT NULL CHECK (status IN ('open', 'closed')) DEFAULT 'open',
    notes TEXT
);

CREATE INDEX idx_shifts_user_id ON shifts(user_id);
CREATE INDEX idx_shifts_status ON shifts(status);

-- +goose Down
DROP TABLE shifts;
