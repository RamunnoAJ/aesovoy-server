package store

import (
	"database/sql"
	"time"
)

type PaymentMethod struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Reference string    `json:"reference"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PaymentMethodStore interface {
	CreatePaymentMethod(pm *PaymentMethod) error
	GetPaymentMethodByID(id int64) (*PaymentMethod, error)
	UpdatePaymentMethod(pm *PaymentMethod) error
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
		INSERT INTO payment_methods (name, reference)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	return s.DB.QueryRow(query, pm.Name, pm.Reference).Scan(&pm.ID, &pm.CreatedAt, &pm.UpdatedAt)
}

func (s *PostgresPaymentMethodStore) UpdatePaymentMethod(pm *PaymentMethod) error {
	query := `
		UPDATE payment_methods
		SET name = $1, reference = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at`
	
	return s.DB.QueryRow(query, pm.Name, pm.Reference, pm.ID).Scan(&pm.UpdatedAt)
}

func (s *PostgresPaymentMethodStore) GetPaymentMethodByID(id int64) (*PaymentMethod, error) {
	query := `
		SELECT id, name, reference, created_at, updated_at
		FROM payment_methods
		WHERE id = $1`

	var pm PaymentMethod
	err := s.DB.QueryRow(query, id).Scan(&pm.ID, &pm.Name, &pm.Reference, &pm.CreatedAt, &pm.UpdatedAt)
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
		SELECT id, name, reference, created_at, updated_at
		FROM payment_methods
		ORDER BY name`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paymentMethods []*PaymentMethod
	for rows.Next() {
		var pm PaymentMethod
		if err := rows.Scan(&pm.ID, &pm.Name, &pm.Reference, &pm.CreatedAt, &pm.UpdatedAt); err != nil {
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
