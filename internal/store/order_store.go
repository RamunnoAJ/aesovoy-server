package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type OrderState string

const (
	OrderTodo      OrderState = "todo"
	OrderDone      OrderState = "done"
	OrderCancelled OrderState = "cancelled"
	OrderDelivered OrderState = "delivered"
)

type Money = string

type Order struct {
	ID         int64       `json:"id"`
	ClientID   int64       `json:"client_id"`
	ClientName string      `json:"client_name,omitempty"`
	Total      Money       `json:"total"`
	Date       time.Time   `json:"date"`
	State      OrderState  `json:"state"`
	CreatedAt  time.Time   `json:"created_at"`
	Items      []OrderItem `json:"items,omitempty"`
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
	GetOrderByID(id int64) (*Order, error)
	ListOrders(f OrderFilter) ([]*Order, error)
	GetDailyStats(date time.Time) (*DailyOrderStats, error)
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

func (s *PostgresOrderStore) GetDailyStats(date time.Time) (*DailyOrderStats, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)

	stats := &DailyOrderStats{}
	query := `
		SELECT COALESCE(SUM(total), 0), COUNT(*)
		FROM orders
		WHERE date >= $1 AND date < $2 AND state != 'cancelled'`

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
	  INSERT INTO orders (client_id, total, state)
	  VALUES ($1, 0, $2)
	  RETURNING id, total, date, created_at`
	if err = tx.QueryRow(qOrder, o.ClientID, o.State).Scan(&o.ID, &o.Total, &o.Date, &o.CreatedAt); err != nil {
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
	const q = `UPDATE orders SET state=$1 WHERE id=$2`
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

func (s *PostgresOrderStore) GetOrderByID(id int64) (*Order, error) {
	const q = `
	SELECT o.id, o.client_id, c.name, o.total::text, o.date, o.state, o.created_at 
	FROM orders o
	JOIN clients c ON c.id = o.client_id
	WHERE o.id=$1`
	o := &Order{}
	if err := s.db.QueryRow(q, id).Scan(&o.ID, &o.ClientID, &o.ClientName, &o.Total, &o.Date, &o.State, &o.CreatedAt); err != nil {
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
	SELECT o.id, o.client_id, c.name, o.total::text, o.date, o.state, o.created_at 
	FROM orders o
	JOIN clients c ON c.id = o.client_id`
	where := ""
	args := []any{}

	if f.ClientID != nil {
		where = where + fmt.Sprintf("%s o.client_id=$%d", utils.Tern(where == "", "WHERE", " AND "), len(args)+1)
		args = append(args, *f.ClientID)
	}
	if f.State != nil {
		where = where + fmt.Sprintf("%s o.state=$%d", utils.Tern(where == "", "WHERE", " AND "), len(args)+1)
		args = append(args, *f.State)
	}
	if f.ClientName != "" {
		where = where + fmt.Sprintf("%s unaccent(c.name) ILIKE unaccent('%%' || $%d || '%%')", utils.Tern(where == "", "WHERE", " AND "), len(args)+1)
		args = append(args, f.ClientName)
	}
	if f.StartDate != nil {
		where = where + fmt.Sprintf("%s o.date >= $%d", utils.Tern(where == "", "WHERE", " AND "), len(args)+1)
		args = append(args, *f.StartDate)
	}
	if f.EndDate != nil {
		// Assuming EndDate is inclusive, and we might need to cover the whole day if time is 00:00:00.
		// But usually caller handles the time part (e.g. setting it to 23:59:59).
		// We'll just compare strictly here.
		where = where + fmt.Sprintf("%s o.date <= $%d", utils.Tern(where == "", "WHERE", " AND "), len(args)+1)
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
		if err := rows.Scan(&o.ID, &o.ClientID, &o.ClientName, &o.Total, &o.Date, &o.State, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
