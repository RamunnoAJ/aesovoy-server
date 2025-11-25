package store

import (
	"database/sql"
	"time"
)

type LocalStock struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LocalStockStore interface {
	Create(productID int64, quantity int) (*LocalStock, error)
	GetByProductID(productID int64) (*LocalStock, error)
	ListAll() ([]*LocalStock, error)
	AdjustQuantity(productID int64, delta int) (*LocalStock, error)

	// Transactional methods
	CreateInTx(tx *sql.Tx, productID int64, quantity int) (*LocalStock, error)
	AdjustQuantityTx(tx *sql.Tx, productID int64, delta int) (*LocalStock, error)
}

type PostgresLocalStockStore struct {
	DB *sql.DB
}

func NewPostgresLocalStockStore(db *sql.DB) *PostgresLocalStockStore {
	return &PostgresLocalStockStore{DB: db}
}

func (s *PostgresLocalStockStore) Create(productID int64, quantity int) (*LocalStock, error) {
	query := `
		INSERT INTO local_stock (product_id, quantity)
		VALUES ($1, $2)
		RETURNING id, product_id, quantity, created_at, updated_at`

	var stock LocalStock
	err := s.DB.QueryRow(query, productID, quantity).Scan(&stock.ID, &stock.ProductID, &stock.Quantity, &stock.CreatedAt, &stock.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (s *PostgresLocalStockStore) CreateInTx(tx *sql.Tx, productID int64, quantity int) (*LocalStock, error) {
	query := `
		INSERT INTO local_stock (product_id, quantity)
		VALUES ($1, $2)
		RETURNING id, product_id, quantity, created_at, updated_at`

	var stock LocalStock
	err := tx.QueryRow(query, productID, quantity).Scan(&stock.ID, &stock.ProductID, &stock.Quantity, &stock.CreatedAt, &stock.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (s *PostgresLocalStockStore) GetByProductID(productID int64) (*LocalStock, error) {
	query := `
		SELECT id, product_id, quantity, created_at, updated_at
		FROM local_stock
		WHERE product_id = $1`

	var stock LocalStock
	err := s.DB.QueryRow(query, productID).Scan(&stock.ID, &stock.ProductID, &stock.Quantity, &stock.CreatedAt, &stock.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}
	return &stock, nil
}

func (s *PostgresLocalStockStore) ListAll() ([]*LocalStock, error) {
	query := `
		SELECT id, product_id, quantity, created_at, updated_at
		FROM local_stock
		ORDER BY id`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []*LocalStock
	for rows.Next() {
		var stock LocalStock
		if err := rows.Scan(&stock.ID, &stock.ProductID, &stock.Quantity, &stock.CreatedAt, &stock.UpdatedAt); err != nil {
			return nil, err
		}
		stocks = append(stocks, &stock)
	}
	return stocks, nil
}

func (s *PostgresLocalStockStore) AdjustQuantity(productID int64, delta int) (*LocalStock, error) {
	query := `
		UPDATE local_stock
		SET quantity = quantity + $1, updated_at = NOW()
		WHERE product_id = $2
		RETURNING id, product_id, quantity, created_at, updated_at`

	var stock LocalStock
	err := s.DB.QueryRow(query, delta, productID).Scan(&stock.ID, &stock.ProductID, &stock.Quantity, &stock.CreatedAt, &stock.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}

func (s *PostgresLocalStockStore) AdjustQuantityTx(tx *sql.Tx, productID int64, delta int) (*LocalStock, error) {
	query := `
		UPDATE local_stock
		SET quantity = quantity + $1, updated_at = NOW()
		WHERE product_id = $2
		RETURNING id, product_id, quantity, created_at, updated_at`

	var stock LocalStock
	err := tx.QueryRow(query, delta, productID).Scan(&stock.ID, &stock.ProductID, &stock.Quantity, &stock.CreatedAt, &stock.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &stock, nil
}
