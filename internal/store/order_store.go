package store

import (
	"database/sql"
	"fmt"
	"time"
)

type OrderState string

const (
	OrderTodo      OrderState = "todo"
	OrderDone      OrderState = "done"
	OrderCancelled OrderState = "cancelled"
	OrderDelivered OrderState = "delivered"
	OrderPaid      OrderState = "paid"
)

type ProductionRequirement struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

type Money = string

type Order struct {
	ID                int64       `json:"id"`
	ClientID          int64       `json:"client_id"`
	ClientName        string      `json:"client_name,omitempty"`
	Total             Money       `json:"total"`
	Date              time.Time   `json:"date"`
	State             OrderState  `json:"state"`
	PaymentMethodID   *int64      `json:"payment_method_id,omitempty"`
	PaymentMethodName string      `json:"payment_method_name,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	DeletedAt         *time.Time  `json:"deleted_at"`
	Items             []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID          int64     `json:"id"`
	OrderID     int64     `json:"order_id"`
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name,omitempty"`
	Quantity    int       `json:"quantity"`
	Price       Money     `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderStore interface {
	CreateOrder(o *Order, items []OrderItem) error
	UpdateOrderState(id int64, state OrderState) error
	DeleteOrder(id int64) error
	GetOrderByID(id int64) (*Order, error)
	ListOrders(f OrderFilter) ([]*Order, error)
	GetStats(start, end time.Time) (*DailyOrderStats, error)
	GetPendingProductionRequirements() ([]*ProductionRequirement, error)
}

type DailyOrderStats struct {
	TotalAmount float64
	TotalCount  int
}

type OrderFilter struct {
	ClientID   *int64
	State      *OrderState
	ClientName string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

type PostgresOrderStore struct{ db *sql.DB }

func NewPostgresOrderStore(db *sql.DB) *PostgresOrderStore { return &PostgresOrderStore{db: db} }

func (s *PostgresOrderStore) GetStats(start, end time.Time) (*DailyOrderStats, error) {
	stats := &DailyOrderStats{}
	query := `
		SELECT COALESCE(SUM(total), 0), COUNT(*)
		FROM orders
		WHERE date >= $1 AND date < $2 AND state != 'cancelled' AND deleted_at IS NULL`

	err := s.db.QueryRow(query, start, end).Scan(&stats.TotalAmount, &stats.TotalCount)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (s *PostgresOrderStore) CreateOrder(o *Order, items []OrderItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// total lo calcula la DB desde items insertados
	const qOrder = `
	  INSERT INTO orders (client_id, total, state, payment_method_id)
	  VALUES ($1, 0, $2, $3)
	  RETURNING id, total, date, created_at`
	if err = tx.QueryRow(qOrder, o.ClientID, o.State, o.PaymentMethodID).Scan(&o.ID, &o.Total, &o.Date, &o.CreatedAt); err != nil {
		return err
	}

	const qItem = `
	  INSERT INTO order_products (quantity, price, product_id, order_id)
	  VALUES ($1,$2,$3,$4)
	  RETURNING id, created_at`
	for i := range items {
		items[i].OrderID = o.ID
		if err = tx.QueryRow(qItem, items[i].Quantity, items[i].Price, items[i].ProductID, items[i].OrderID).
			Scan(&items[i].ID, &items[i].CreatedAt); err != nil {
			return err
		}
	}

	const qRecalc = `
	  UPDATE orders o
	  SET total = COALESCE(t.sum, 0)
	  FROM (
	    SELECT op.order_id, SUM((op.quantity::numeric)*op.price) AS sum
	    FROM order_products op
	    WHERE op.order_id = $1
	    GROUP BY op.order_id
	  ) t
	  WHERE o.id = t.order_id AND o.id = $1
	  RETURNING o.total`
	if err = tx.QueryRow(qRecalc, o.ID).Scan(&o.Total); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	o.Items = items
	return nil
}

func (s *PostgresOrderStore) UpdateOrderState(id int64, state OrderState) error {
	const q = `UPDATE orders SET state=$1 WHERE id=$2 AND deleted_at IS NULL`
	res, err := s.db.Exec(q, state, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PostgresOrderStore) DeleteOrder(id int64) error {
	const q = `UPDATE orders SET deleted_at = NOW() WHERE id=$1 AND deleted_at IS NULL`
	res, err := s.db.Exec(q, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *PostgresOrderStore) GetOrderByID(id int64) (*Order, error) {
	const q = `
	SELECT o.id, o.client_id, c.name, o.total::text, o.date, o.state, o.payment_method_id, COALESCE(pm.name, ''), o.created_at, o.deleted_at
	FROM orders o
	JOIN clients c ON c.id = o.client_id
	LEFT JOIN payment_methods pm ON pm.id = o.payment_method_id
	WHERE o.id=$1 AND o.deleted_at IS NULL`
	o := &Order{}
	if err := s.db.QueryRow(q, id).Scan(&o.ID, &o.ClientID, &o.ClientName, &o.Total, &o.Date, &o.State, &o.PaymentMethodID, &o.PaymentMethodName, &o.CreatedAt, &o.DeletedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	const qi = `
	SELECT op.id, op.order_id, op.product_id, p.name, op.quantity, op.price::text, op.created_at 
	FROM order_products op
	JOIN products p ON p.id = op.product_id
	WHERE op.order_id=$1 
	ORDER BY op.id`
	rows, err := s.db.Query(qi, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Quantity, &it.Price, &it.CreatedAt); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, it)
	}
	return o, rows.Err()
}

func (s *PostgresOrderStore) ListOrders(f OrderFilter) ([]*Order, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	q := `
	SELECT o.id, o.client_id, c.name, o.total::text, o.date, o.state, o.payment_method_id, COALESCE(pm.name, ''), o.created_at, o.deleted_at
	FROM orders o
	JOIN clients c ON c.id = o.client_id
	LEFT JOIN payment_methods pm ON pm.id = o.payment_method_id`
	where := "WHERE o.deleted_at IS NULL"
	args := []any{}

	if f.ClientID != nil {
		where += fmt.Sprintf(" AND o.client_id=$%d", len(args)+1)
		args = append(args, *f.ClientID)
	}
	if f.State != nil {
		where += fmt.Sprintf(" AND o.state=$%d", len(args)+1)
		args = append(args, *f.State)
	}
	if f.ClientName != "" {
		where += fmt.Sprintf(" AND unaccent(c.name) ILIKE unaccent('%%' || $%d || '%%')", len(args)+1)
		args = append(args, f.ClientName)
	}
	if f.StartDate != nil {
		where += fmt.Sprintf(" AND o.date >= $%d", len(args)+1)
		args = append(args, *f.StartDate)
	}
	if f.EndDate != nil {
		where += fmt.Sprintf(" AND o.date <= $%d", len(args)+1)
		args = append(args, *f.EndDate)
	}

	q = q + " " + where + fmt.Sprintf(" ORDER BY o.date DESC, o.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Order
	for rows.Next() {
		o := &Order{}
		if err := rows.Scan(&o.ID, &o.ClientID, &o.ClientName, &o.Total, &o.Date, &o.State, &o.PaymentMethodID, &o.PaymentMethodName, &o.CreatedAt, &o.DeletedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *PostgresOrderStore) GetPendingProductionRequirements() ([]*ProductionRequirement, error) {
	const q = `
		SELECT 
			p.id, 
			p.name, 
			SUM(op.quantity) as total_quantity
		FROM order_products op
		JOIN orders o ON o.id = op.order_id
		JOIN products p ON p.id = op.product_id
		WHERE o.state = 'todo' AND o.deleted_at IS NULL AND p.deleted_at IS NULL
		GROUP BY p.id, p.name
		ORDER BY total_quantity DESC
	`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requirements []*ProductionRequirement
	for rows.Next() {
		pr := &ProductionRequirement{}
		if err := rows.Scan(&pr.ProductID, &pr.ProductName, &pr.Quantity); err != nil {
			return nil, err
		}
		requirements = append(requirements, pr)
	}
	return requirements, rows.Err()
}
