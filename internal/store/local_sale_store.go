package store

import (
	"database/sql"
	"time"
)

type LocalSale struct {
	ID              int64           `json:"id"`
	PaymentMethodID int64           `json:"payment_method_id"`
	Subtotal        string          `json:"subtotal"`
	Total           string          `json:"total"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	DeletedAt       *time.Time      `json:"deleted_at"`
	Items           []LocalSaleItem `json:"items,omitempty"`
}

type LocalSaleItem struct {
	ID           int64  `json:"id"`
	LocalSaleID  int64  `json:"local_sale_id"`
	ProductID    int64  `json:"product_id"`
	Quantity     int    `json:"quantity"`
	UnitPrice    string `json:"unit_price"`
	LineSubtotal string `json:"line_subtotal"`
}

type DailySalesStats struct {
	TotalAmount float64
	TotalCount  int
	ByMethod    map[string]float64
}

type SalesHistoryRecord struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

type LocalSaleStore interface {
	CreateInTx(tx *sql.Tx, sale *LocalSale, items []LocalSaleItem) error
	DeleteInTx(tx *sql.Tx, id int64) error
	GetByID(id int64) (*LocalSale, error)
	ListAll() ([]*LocalSale, error)
	ListByDate(start, end time.Time) ([]*LocalSale, error)
	GetStats(start, end time.Time) (*DailySalesStats, error)
	GetSalesHistory(days int) ([]*SalesHistoryRecord, error)
}

type PostgresLocalSaleStore struct {
	db *sql.DB
}

func NewPostgresLocalSaleStore(db *sql.DB) *PostgresLocalSaleStore {
	return &PostgresLocalSaleStore{db: db}
}

func (s *PostgresLocalSaleStore) DeleteInTx(tx *sql.Tx, id int64) error {
	query := `UPDATE local_sales SET deleted_at = NOW() WHERE id = $1`
	result, err := tx.Exec(query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PostgresLocalSaleStore) ListByDate(start, end time.Time) ([]*LocalSale, error) {
	query := `
		SELECT id, payment_method_id, subtotal::text, total::text, created_at, updated_at, deleted_at
		FROM local_sales 
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []*LocalSale
	for rows.Next() {
		var sale LocalSale
		if err := rows.Scan(&sale.ID, &sale.PaymentMethodID, &sale.Subtotal, &sale.Total, &sale.CreatedAt, &sale.UpdatedAt, &sale.DeletedAt); err != nil {
			return nil, err
		}
		sales = append(sales, &sale)
	}
	return sales, rows.Err()
}

func (s *PostgresLocalSaleStore) GetStats(start, end time.Time) (*DailySalesStats, error) {
	stats := &DailySalesStats{
		ByMethod: make(map[string]float64),
	}

	// 1. Total and Count (Exclude deleted)
	queryTotal := `
		SELECT COALESCE(SUM(total), 0), COUNT(*)
		FROM local_sales
		WHERE created_at >= $1 AND created_at < $2 AND deleted_at IS NULL`
	
	err := s.db.QueryRow(queryTotal, start, end).Scan(&stats.TotalAmount, &stats.TotalCount)
	if err != nil {
		return nil, err
	}

	// 2. Breakdown by Method (Exclude deleted)
	queryMethod := `
		SELECT pm.name, COALESCE(SUM(ls.total), 0)
		FROM local_sales ls
		JOIN payment_methods pm ON ls.payment_method_id = pm.id
		WHERE ls.created_at >= $1 AND ls.created_at < $2 AND ls.deleted_at IS NULL
		GROUP BY pm.name`
	
	rows, err := s.db.Query(queryMethod, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var total float64
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		stats.ByMethod[name] = total
	}

	return stats, rows.Err()
}

func (s *PostgresLocalSaleStore) CreateInTx(tx *sql.Tx, sale *LocalSale, items []LocalSaleItem) error {
	// 1. Create the LocalSale record
	saleQuery := `
		INSERT INTO local_sales (payment_method_id, subtotal, total)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`
	err := tx.QueryRow(saleQuery, sale.PaymentMethodID, sale.Subtotal, sale.Total).
		Scan(&sale.ID, &sale.CreatedAt, &sale.UpdatedAt)
	if err != nil {
		return err
	}

	// 2. Create the LocalSaleItem records
	itemQuery := `
		INSERT INTO local_sale_items (local_sale_id, product_id, quantity, unit_price, line_subtotal)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	for i := range items {
		item := &items[i]
		item.LocalSaleID = sale.ID
		err := tx.QueryRow(itemQuery, item.LocalSaleID, item.ProductID, item.Quantity, item.UnitPrice, item.LineSubtotal).
			Scan(&item.ID)
		if err != nil {
			return err // Rollback will be handled by the service
		}
	}
	sale.Items = items
	return nil
}

func (s *PostgresLocalSaleStore) GetByID(id int64) (*LocalSale, error) {
	query := `
		SELECT id, payment_method_id, subtotal::text, total::text, created_at, updated_at, deleted_at
		FROM local_sales WHERE id = $1`

	sale := &LocalSale{}
	err := s.db.QueryRow(query, id).Scan(&sale.ID, &sale.PaymentMethodID, &sale.Subtotal, &sale.Total, &sale.CreatedAt, &sale.UpdatedAt, &sale.DeletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	itemsQuery := `
		SELECT id, local_sale_id, product_id, quantity, unit_price::text, line_subtotal::text
		FROM local_sale_items WHERE local_sale_id = $1 ORDER BY id`
	rows, err := s.db.Query(itemsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item LocalSaleItem
		if err := rows.Scan(&item.ID, &item.LocalSaleID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.LineSubtotal); err != nil {
			return nil, err
		}
		sale.Items = append(sale.Items, item)
	}

	return sale, rows.Err()
}

func (s *PostgresLocalSaleStore) ListAll() ([]*LocalSale, error) {
	query := `
		SELECT id, payment_method_id, subtotal::text, total::text, created_at, updated_at, deleted_at
		FROM local_sales ORDER BY created_at DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []*LocalSale
	for rows.Next() {
		var sale LocalSale
		if err := rows.Scan(&sale.ID, &sale.PaymentMethodID, &sale.Subtotal, &sale.Total, &sale.CreatedAt, &sale.UpdatedAt, &sale.DeletedAt); err != nil {
			return nil, err
		}
		sales = append(sales, &sale)
	}
	return sales, rows.Err()
}

func (s *PostgresLocalSaleStore) GetSalesHistory(days int) ([]*SalesHistoryRecord, error) {
	query := `
		SELECT DATE(created_at) as sale_date, COALESCE(SUM(total), 0)
		FROM local_sales
		WHERE created_at >= NOW() - ($1 || ' days')::INTERVAL
		  AND deleted_at IS NULL
		GROUP BY sale_date
		ORDER BY sale_date ASC`

	rows, err := s.db.Query(query, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []*SalesHistoryRecord
	for rows.Next() {
		var r SalesHistoryRecord
		var t time.Time
		if err := rows.Scan(&t, &r.Amount); err != nil {
			return nil, err
		}
		r.Date = t.Format("2006-01-02")
		history = append(history, &r)
	}
	return history, rows.Err()
}
