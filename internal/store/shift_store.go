package store

import (
	"database/sql"
	"time"
)

type Shift struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	StartCash       float64    `json:"start_cash"`
	EndCashExpected *float64   `json:"end_cash_expected"`
	EndCashDeclared *float64   `json:"end_cash_declared"`
	Difference      *float64   `json:"difference"`
	Status          string     `json:"status"` // 'open', 'closed'
	Notes           string     `json:"notes"`
}

type ShiftStore interface {
	Create(shift *Shift) error
	Update(shift *Shift) error
	GetByID(id int64) (*Shift, error)
	GetOpenShiftByUserID(userID int64) (*Shift, error)
	ListByUserID(userID int64, limit, offset int) ([]*Shift, error)
}

type PostgresShiftStore struct {
	db *sql.DB
}

func NewPostgresShiftStore(db *sql.DB) *PostgresShiftStore {
	return &PostgresShiftStore{db: db}
}

func (s *PostgresShiftStore) Create(shift *Shift) error {
	query := `
		INSERT INTO shifts (user_id, start_time, start_cash, status, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	
	if shift.StartTime.IsZero() {
		shift.StartTime = time.Now()
	}
	if shift.Status == "" {
		shift.Status = "open"
	}

	return s.db.QueryRow(query, shift.UserID, shift.StartTime, shift.StartCash, shift.Status, shift.Notes).Scan(&shift.ID)
}

func (s *PostgresShiftStore) Update(shift *Shift) error {
	query := `
		UPDATE shifts
		SET end_time=$1, end_cash_expected=$2, end_cash_declared=$3, difference=$4, status=$5, notes=$6
		WHERE id=$7`
	
	_, err := s.db.Exec(query, shift.EndTime, shift.EndCashExpected, shift.EndCashDeclared, shift.Difference, shift.Status, shift.Notes, shift.ID)
	return err
}

func (s *PostgresShiftStore) GetByID(id int64) (*Shift, error) {
	query := `
		SELECT id, user_id, start_time, end_time, start_cash, end_cash_expected, end_cash_declared, difference, status, COALESCE(notes, '')
		FROM shifts WHERE id = $1`
	
	var shift Shift
	err := s.db.QueryRow(query, id).Scan(
		&shift.ID, &shift.UserID, &shift.StartTime, &shift.EndTime, &shift.StartCash, 
		&shift.EndCashExpected, &shift.EndCashDeclared, &shift.Difference, &shift.Status, &shift.Notes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

func (s *PostgresShiftStore) GetOpenShiftByUserID(userID int64) (*Shift, error) {
	query := `
		SELECT id, user_id, start_time, end_time, start_cash, end_cash_expected, end_cash_declared, difference, status, COALESCE(notes, '')
		FROM shifts WHERE user_id = $1 AND status = 'open'
		ORDER BY start_time DESC LIMIT 1`
	
	var shift Shift
	err := s.db.QueryRow(query, userID).Scan(
		&shift.ID, &shift.UserID, &shift.StartTime, &shift.EndTime, &shift.StartCash, 
		&shift.EndCashExpected, &shift.EndCashDeclared, &shift.Difference, &shift.Status, &shift.Notes,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &shift, nil
}

func (s *PostgresShiftStore) ListByUserID(userID int64, limit, offset int) ([]*Shift, error) {
	query := `
		SELECT id, user_id, start_time, end_time, start_cash, end_cash_expected, end_cash_declared, difference, status, COALESCE(notes, '')
		FROM shifts WHERE user_id = $1
		ORDER BY start_time DESC LIMIT $2 OFFSET $3`
	
	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shifts []*Shift
	for rows.Next() {
		var s Shift
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.StartTime, &s.EndTime, &s.StartCash, 
			&s.EndCashExpected, &s.EndCashDeclared, &s.Difference, &s.Status, &s.Notes,
		); err != nil {
			return nil, err
		}
		shifts = append(shifts, &s)
	}
	return shifts, rows.Err()
}
