package store

import (
	"database/sql"
	"time"
)

type PostgresCategoryStore struct {
	db *sql.DB
}

func NewPostgresCategoryStore(db *sql.DB) *PostgresCategoryStore {
	return &PostgresCategoryStore{
		db: db,
	}
}

type Category struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at"`
}

type CategoryStore interface {
	CreateCategory(*Category) error
	GetCategoryByID(id int64) (*Category, error)
	UpdateCategory(*Category) error
	DeleteCategory(id int64) error
	GetAllCategories() ([]*Category, error)
}

func (s *PostgresCategoryStore) CreateCategory(category *Category) error {
	query := `
	INSERT INTO categories (name, description)
	VALUES ($1, $2)
	RETURNING id, created_at 
	`

	err := s.db.QueryRow(query, category.Name, category.Description).Scan(
		&category.ID,
		&category.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresCategoryStore) GetCategoryByID(id int64) (*Category, error) {
	category := &Category{}

	query := `
	SELECT id, name, description, created_at, deleted_at
	FROM categories
	WHERE id = $1 AND deleted_at IS NULL
	`

	err := s.db.QueryRow(query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.CreatedAt,
		&category.DeletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return category, nil
}

func (s *PostgresCategoryStore) UpdateCategory(category *Category) error {
	query := `
	UPDATE categories
	SET name = $1, description = $2 
	WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := s.db.Exec(query, category.Name, category.Description, category.ID)
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

func (s *PostgresCategoryStore) DeleteCategory(id int64) error {
	query := `
	UPDATE categories
	SET deleted_at = NOW()
	WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := s.db.Exec(query, id)
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

func (s *PostgresCategoryStore) GetAllCategories() ([]*Category, error) {
	query := `
    SELECT id, name, description, created_at, deleted_at
    FROM categories
	WHERE deleted_at IS NULL
    ORDER BY name
    `

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category

	for rows.Next() {
		c := &Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.DeletedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}
