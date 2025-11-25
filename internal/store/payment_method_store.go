package store

import (
	"database/sql"
	"time"
)

type PaymentMethod struct {
	ID        int64     `json:"id"`
	Owner     string    `json:"owner"`
	Reference string    `json:"reference"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaymentMethodStore interface {
	CreatePaymentMethod(pm *PaymentMethod) error
	GetPaymentMethodByID(id int64) (*PaymentMethod, error)
	GetAllPaymentMethods() ([]*PaymentMethod, error)
	DeletePaymentMethod(id int64) error
}

type PostgresPaymentMethodStore struct {
	DB *sql.DB
}

func NewPostgresPaymentMethodStore(db *sql.DB) *PostgresPaymentMethodStore {
	return &PostgresPaymentMethodStore{DB: db}
}

func (s *PostgresPaymentMethodStore) CreatePaymentMethod(pm *PaymentMethod) error {
	query := `
		INSERT INTO payment_methods (owner, reference)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	return s.DB.QueryRow(query, pm.Owner, pm.Reference).Scan(&pm.ID, &pm.CreatedAt, &pm.UpdatedAt)
}

func (s *PostgresPaymentMethodStore) GetPaymentMethodByID(id int64) (*PaymentMethod, error) {
	query := `
		SELECT id, owner, reference, created_at, updated_at
		FROM payment_methods
		WHERE id = $1`

	var pm PaymentMethod
	err := s.DB.QueryRow(query, id).Scan(&pm.ID, &pm.Owner, &pm.Reference, &pm.CreatedAt, &pm.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &pm, nil
}

func (s *PostgresPaymentMethodStore) GetAllPaymentMethods() ([]*PaymentMethod, error) {
	query := `
		SELECT id, owner, reference, created_at, updated_at
		FROM payment_methods
		ORDER BY owner`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paymentMethods []*PaymentMethod
	for rows.Next() {
		var pm PaymentMethod
		if err := rows.Scan(&pm.ID, &pm.Owner, &pm.Reference, &pm.CreatedAt, &pm.UpdatedAt); err != nil {
			return nil, err
		}
		paymentMethods = append(paymentMethods, &pm)
	}

	return paymentMethods, nil
}

func (s *PostgresPaymentMethodStore) DeletePaymentMethod(id int64) error {
	query := "DELETE FROM payment_methods WHERE id = $1"
	result, err := s.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
