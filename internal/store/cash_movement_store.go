package store

import (
	"database/sql"
	"time"
)

type CashMovementType string

const (
	CashMovementIn  CashMovementType = "in"
	CashMovementOut CashMovementType = "out"
)

type CashMovement struct {
	ID        int64            `json:"id"`
	ShiftID   int64            `json:"shift_id"`
	Amount    float64          `json:"amount"`
	Type      CashMovementType `json:"type"`
	Reason    string           `json:"reason"`
	CreatedAt time.Time        `json:"created_at"`
}

type CashMovementStore interface {
	Create(m *CashMovement) error
	ListByShiftID(shiftID int64) ([]*CashMovement, error)
	GetTotalByShiftID(shiftID int64) (totalIn float64, totalOut float64, err error)
}

type PostgresCashMovementStore struct {
	db *sql.DB
}

func NewPostgresCashMovementStore(db *sql.DB) *PostgresCashMovementStore {
	return &PostgresCashMovementStore{db: db}
}

func (s *PostgresCashMovementStore) Create(m *CashMovement) error {
	const q = `
	INSERT INTO cash_movements (shift_id, amount, type, reason)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at`
	return s.db.QueryRow(q, m.ShiftID, m.Amount, m.Type, m.Reason).Scan(&m.ID, &m.CreatedAt)
}

func (s *PostgresCashMovementStore) ListByShiftID(shiftID int64) ([]*CashMovement, error) {
	const q = `
	SELECT id, shift_id, amount, type, reason, created_at
	FROM cash_movements
	WHERE shift_id = $1
	ORDER BY created_at DESC`
	
	rows, err := s.db.Query(q, shiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*CashMovement
	for rows.Next() {
		m := &CashMovement{}
		if err := rows.Scan(&m.ID, &m.ShiftID, &m.Amount, &m.Type, &m.Reason, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (s *PostgresCashMovementStore) GetTotalByShiftID(shiftID int64) (float64, float64, error) {
	const q = `
	SELECT type, COALESCE(SUM(amount), 0)
	FROM cash_movements
	WHERE shift_id = $1
	GROUP BY type`

	rows, err := s.db.Query(q, shiftID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var totalIn, totalOut float64
	for rows.Next() {
		var t string
		var amount float64
		if err := rows.Scan(&t, &amount); err != nil {
			return 0, 0, err
		}
		if t == "in" {
			totalIn = amount
		} else if t == "out" {
			totalOut = amount
		}
	}
	return totalIn, totalOut, rows.Err()
}
