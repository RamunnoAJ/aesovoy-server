package store

import (
	"database/sql"
	"fmt"
	"time"
)

type ExpenseType string

const (
	ExpenseTypeLocal      ExpenseType = "local"
	ExpenseTypeProduction ExpenseType = "production"
)

type ExpenseCategory struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Expense struct {
	ID           int64       `json:"id"`
	Amount       string      `json:"amount"` // stored as NUMERIC, transferred as string
	ImagePath    string      `json:"image_path,omitempty"`
	ProviderID   *int64      `json:"provider_id,omitempty"`
	ProviderName string      `json:"provider_name,omitempty"`
	CategoryID   int64       `json:"category_id"`
	CategoryName string      `json:"category_name"`
	Type         ExpenseType `json:"type"`
	Date         time.Time   `json:"date"`
	CreatedAt    time.Time   `json:"created_at"`
	DeletedAt    *time.Time  `json:"deleted_at,omitempty"`
}

type ExpenseStore interface {
	CreateExpense(e *Expense) error
	UpdateExpense(e *Expense) error
	DeleteExpense(id int64) error
	GetExpenseByID(id int64) (*Expense, error)
	ListExpenses(f ExpenseFilter) ([]*Expense, error)

	CreateExpenseCategory(c *ExpenseCategory) error
	GetAllExpenseCategories() ([]*ExpenseCategory, error)
	GetExpenseCategoryByID(id int64) (*ExpenseCategory, error)
}

type ExpenseFilter struct {
	Type       *ExpenseType
	CategoryID *int64
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

type PostgresExpenseStore struct{ db *sql.DB }

func NewPostgresExpenseStore(db *sql.DB) *PostgresExpenseStore {
	return &PostgresExpenseStore{db: db}
}

func (s *PostgresExpenseStore) CreateExpense(e *Expense) error {
	const q = `
	INSERT INTO expenses (amount, image_path, provider_id, category_id, type, date)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING id, created_at`
	
	return s.db.QueryRow(q, e.Amount, e.ImagePath, e.ProviderID, e.CategoryID, e.Type, e.Date).
		Scan(&e.ID, &e.CreatedAt)
}

func (s *PostgresExpenseStore) UpdateExpense(e *Expense) error {
	const q = `
	UPDATE expenses
	SET amount=$1, image_path=$2, provider_id=$3, category_id=$4, type=$5, date=$6
	WHERE id=$7 AND deleted_at IS NULL`
	
	res, err := s.db.Exec(q, e.Amount, e.ImagePath, e.ProviderID, e.CategoryID, e.Type, e.Date, e.ID)
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

func (s *PostgresExpenseStore) DeleteExpense(id int64) error {
	const q = `UPDATE expenses SET deleted_at = NOW() WHERE id=$1 AND deleted_at IS NULL`
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

func (s *PostgresExpenseStore) GetExpenseByID(id int64) (*Expense, error) {
	const q = `
	SELECT e.id, e.amount::text, COALESCE(e.image_path, ''), e.provider_id, p.name, e.category_id, ec.name, e.type, e.date, e.created_at, e.deleted_at
	FROM expenses e
	LEFT JOIN providers p ON p.id = e.provider_id
	LEFT JOIN expense_categories ec ON ec.id = e.category_id
	WHERE e.id=$1 AND e.deleted_at IS NULL`
	
	e := &Expense{}
	var providerName sql.NullString
	
	err := s.db.QueryRow(q, id).Scan(
		&e.ID, &e.Amount, &e.ImagePath, &e.ProviderID, &providerName, &e.CategoryID, &e.CategoryName, &e.Type, &e.Date, &e.CreatedAt, &e.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if providerName.Valid {
		e.ProviderName = providerName.String
	}
	return e, nil
}

func (s *PostgresExpenseStore) ListExpenses(f ExpenseFilter) ([]*Expense, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	q := `
	SELECT e.id, e.amount::text, COALESCE(e.image_path, ''), e.provider_id, p.name, e.category_id, ec.name, e.type, e.date, e.created_at, e.deleted_at
	FROM expenses e
	LEFT JOIN providers p ON p.id = e.provider_id
	LEFT JOIN expense_categories ec ON ec.id = e.category_id`
	where := "WHERE e.deleted_at IS NULL"
	args := []any{}

	if f.Type != nil {
		where += fmt.Sprintf(" AND e.type=$%d", len(args)+1)
		args = append(args, *f.Type)
	}
	if f.CategoryID != nil {
		where += fmt.Sprintf(" AND e.category_id=$%d", len(args)+1)
		args = append(args, *f.CategoryID)
	}
	if f.StartDate != nil {
		where += fmt.Sprintf(" AND e.date >= $%d", len(args)+1)
		args = append(args, *f.StartDate)
	}
	if f.EndDate != nil {
		where += fmt.Sprintf(" AND e.date <= $%d", len(args)+1)
		args = append(args, *f.EndDate)
	}

	q = q + " " + where + fmt.Sprintf(" ORDER BY e.date DESC, e.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Expense
	for rows.Next() {
		e := &Expense{}
		var providerName sql.NullString
		if err := rows.Scan(
			&e.ID, &e.Amount, &e.ImagePath, &e.ProviderID, &providerName, &e.CategoryID, &e.CategoryName, &e.Type, &e.Date, &e.CreatedAt, &e.DeletedAt,
		); err != nil {
			return nil, err
		}
		if providerName.Valid {
			e.ProviderName = providerName.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *PostgresExpenseStore) CreateExpenseCategory(c *ExpenseCategory) error {
	const q = `INSERT INTO expense_categories (name) VALUES ($1) RETURNING id, created_at`
	return s.db.QueryRow(q, c.Name).Scan(&c.ID, &c.CreatedAt)
}

func (s *PostgresExpenseStore) GetAllExpenseCategories() ([]*ExpenseCategory, error) {
	const q = `SELECT id, name, created_at FROM expense_categories ORDER BY name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ExpenseCategory
	for rows.Next() {
		var c ExpenseCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

func (s *PostgresExpenseStore) GetExpenseCategoryByID(id int64) (*ExpenseCategory, error) {
	const q = `SELECT id, name, created_at FROM expense_categories WHERE id=$1`
	var c ExpenseCategory
	err := s.db.QueryRow(q, id).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}