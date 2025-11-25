package store

import (
	"database/sql"
	"time"
)

type Ingredient struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IngredientStore interface {
	CreateIngredient(*Ingredient) error
	GetIngredientByID(id int64) (*Ingredient, error)
	GetAllIngredients() ([]*Ingredient, error)
	UpdateIngredient(*Ingredient) error
	DeleteIngredient(id int64) error
}

type PostgresIngredientStore struct {
	db *sql.DB
}

func NewPostgresIngredientStore(db *sql.DB) *PostgresIngredientStore {
	return &PostgresIngredientStore{
		db: db,
	}
}

func (s *PostgresIngredientStore) CreateIngredient(ingredient *Ingredient) error {
	query := `
	INSERT INTO ingredients (name)
	VALUES ($1)
	RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(query, ingredient.Name).Scan(
		&ingredient.ID,
		&ingredient.CreatedAt,
		&ingredient.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresIngredientStore) GetAllIngredients() ([]*Ingredient, error) {
	query := `
	SELECT id, name, created_at, updated_at
	FROM ingredients
	ORDER BY name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ingredients []*Ingredient
	for rows.Next() {
		i := &Ingredient{}
		if err := rows.Scan(&i.ID, &i.Name, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		ingredients = append(ingredients, i)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ingredients, nil
}

func (s *PostgresIngredientStore) GetIngredientByID(id int64) (*Ingredient, error) {
	ingredient := &Ingredient{}

	query := `
	SELECT id, name, created_at, updated_at
	FROM ingredients
	WHERE id = $1
	`

	err := s.db.QueryRow(query, id).Scan(
		&ingredient.ID,
		&ingredient.Name,
		&ingredient.CreatedAt,
		&ingredient.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return ingredient, nil
}

func (s *PostgresIngredientStore) UpdateIngredient(ingredient *Ingredient) error {
	query := `
	UPDATE ingredients
	SET name = $1, updated_at = NOW()
	WHERE id = $2
	RETURNING updated_at
	`

	err := s.db.QueryRow(query, ingredient.Name, ingredient.ID).Scan(&ingredient.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}

	return nil
}

func (s *PostgresIngredientStore) DeleteIngredient(id int64) error {
	query := `
	DELETE FROM ingredients
	WHERE id = $1
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
