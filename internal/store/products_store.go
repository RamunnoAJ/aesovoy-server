package store

import (
	"database/sql"
	"strings"
	"time"
)

type PostgresProductStore struct {
	db *sql.DB
}

func NewPostgresProductStore(db *sql.DB) *PostgresProductStore {
	return &PostgresProductStore{
		db: db,
	}
}

type Product struct {
	ID                int64                `json:"id"`
	CategoryID        int64                `json:"category_id"`
	CategoryName      string               `json:"category_name"`
	Name              string               `json:"name"`
	Description       string               `json:"description,omitempty"`
	UnitPrice         float64              `json:"unit_price"`
	DistributionPrice float64              `json:"distribution_price"`
	CreatedAt         time.Time            `json:"created_at"`
	CurrentStock      float64              `json:"current_stock"`
	Recipe            []*ProductIngredient `json:"recipe,omitempty"`
}

type ProductIngredient struct {
	ID           int64     `json:"id"`
	IngredientID int64     `json:"ingredient_id"`
	Name         string    `json:"name"`
	Quantity     float64   `json:"quantity"`
	Unit         string    `json:"unit"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ProductStore interface {
	CreateProduct(*Product) error
	GetProductByID(id int64) (*Product, error)
	UpdateProduct(*Product) error
	DeleteProduct(id int64) error
	GetAllProduct() ([]*Product, error)
	GetProductsByCategoryID(categoryID int64) ([]*Product, error)
	AddIngredientToProduct(productID int64, ingredientID int64, quantity float64, unit string) (*ProductIngredient, error)
	UpdateProductIngredient(productID, ingredientID int64, quantity float64, unit string) (*ProductIngredient, error)
	RemoveIngredientFromProduct(productID, ingredientID int64) error
	GetProductsByIDs(ids []int64) (map[int64]*Product, error)
	SearchProductsFTS(q string, limit, offset int) ([]*Product, error)
	GetTopSellingProducts(start, end time.Time) ([]*TopProduct, error)
	GetTopSellingProductsLocal(start, end time.Time) ([]*TopProduct, error)
	GetTopSellingProductsDistribution(start, end time.Time) ([]*TopProduct, error)
}

type TopProduct struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
}

func (s *PostgresProductStore) CreateProduct(product *Product) error {
	query := `
	INSERT INTO products (category_id, name, description, unit_price, distribution_price)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at 
	`

	err := s.db.QueryRow(
		query,
		product.CategoryID,
		product.Name,
		product.Description,
		product.UnitPrice,
		product.DistributionPrice,
	).Scan(
		&product.ID,
		&product.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresProductStore) GetProductByID(id int64) (*Product, error) {
	const q = `
	SELECT p.id, p.category_id, c.name AS category_name,
	       p.name, p.description, p.unit_price, p.distribution_price, p.created_at
	FROM products p
	JOIN categories c ON c.id = p.category_id
	WHERE p.id = $1`
	pr := &Product{}
	err := s.db.QueryRow(q, id).Scan(
		&pr.ID, &pr.CategoryID, &pr.CategoryName,
		&pr.Name, &pr.Description, &pr.UnitPrice, &pr.DistributionPrice, &pr.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	const qi = `
	SELECT pi.id, pi.ingredient_id, i.name, pi.quantity, pi.unit, pi.created_at, pi.updated_at
	FROM product_ingredients pi
	JOIN ingredients i ON i.id = pi.ingredient_id
	WHERE pi.product_id = $1
	ORDER BY i.name
	`
	rows, err := s.db.Query(qi, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		pi := &ProductIngredient{}
		if err := rows.Scan(&pi.ID, &pi.IngredientID, &pi.Name, &pi.Quantity, &pi.Unit, &pi.CreatedAt, &pi.UpdatedAt); err != nil {
			return nil, err
		}
		pr.Recipe = append(pr.Recipe, pi)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *PostgresProductStore) UpdateProduct(product *Product) error {
	query := `
	UPDATE products
	SET category_id = $1, name = $2, description = $3, unit_price = $4, distribution_price = $5
	WHERE id = $6
	`

	result, err := s.db.Exec(
		query,
		product.CategoryID,
		product.Name,
		product.Description,
		product.UnitPrice,
		product.DistributionPrice,
		product.ID,
	)
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

func (s *PostgresProductStore) DeleteProduct(id int64) error {
	query := `
	DELETE FROM products
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

func (s *PostgresProductStore) GetAllProduct() ([]*Product, error) {
	const q = `
	SELECT p.id, p.category_id, c.name AS category_name,
	       p.name, p.description, p.unit_price, p.distribution_price, p.created_at
	FROM products p
	JOIN categories c ON c.id = p.category_id
	ORDER BY p.name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		pr := &Product{}
		if err := rows.Scan(
			&pr.ID, &pr.CategoryID, &pr.CategoryName,
			&pr.Name, &pr.Description, &pr.UnitPrice, &pr.DistributionPrice, &pr.CreatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *PostgresProductStore) GetProductsByCategoryID(categoryID int64) ([]*Product, error) {
	const query = `
    SELECT p.id, p.category_id, c.name AS category_name,
           p.name, p.description, p.unit_price, p.distribution_price, p.created_at
    FROM products p
    JOIN categories c ON c.id = p.category_id
    WHERE p.category_id = $1
    ORDER BY p.name`
	rows, err := s.db.Query(query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		pr := &Product{}
		if err := rows.Scan(
			&pr.ID, &pr.CategoryID, &pr.CategoryName,
			&pr.Name, &pr.Description, &pr.UnitPrice, &pr.DistributionPrice, &pr.CreatedAt,
		); err != nil {
			return nil, err
		}
		products = append(products, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *PostgresProductStore) AddIngredientToProduct(productID int64, ingredientID int64, quantity float64, unit string) (*ProductIngredient, error) {
	pi := &ProductIngredient{}
	query := `
	INSERT INTO product_ingredients (product_id, ingredient_id, quantity, unit)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRow(query, productID, ingredientID, quantity, unit).Scan(&pi.ID, &pi.CreatedAt, &pi.UpdatedAt)
	if err != nil {
		return nil, err
	}
	pi.IngredientID = ingredientID
	pi.Quantity = quantity
	pi.Unit = unit
	return pi, nil
}

func (s *PostgresProductStore) UpdateProductIngredient(productID, ingredientID int64, quantity float64, unit string) (*ProductIngredient, error) {
	pi := &ProductIngredient{}
	query := `
	UPDATE product_ingredients
	SET quantity = $1, unit = $2, updated_at = NOW()
	WHERE product_id = $3 AND id = $4
	RETURNING created_at, updated_at
	`
	err := s.db.QueryRow(query, quantity, unit, productID, ingredientID).Scan(&pi.CreatedAt, &pi.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	pi.ID = ingredientID
	pi.Quantity = quantity
	pi.Unit = unit
	return pi, nil
}

func (s *PostgresProductStore) RemoveIngredientFromProduct(productID, ingredientID int64) error {
	query := `
	DELETE FROM product_ingredients
	WHERE product_id = $1 AND id = $2
	`
	result, err := s.db.Exec(query, productID, ingredientID)
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

func (s *PostgresProductStore) GetProductsByIDs(ids []int64) (map[int64]*Product, error) {
	const q = `
	SELECT p.id, p.category_id, c.name AS category_name,
	       p.name, p.description, p.unit_price, p.distribution_price, p.created_at
	FROM products p
	JOIN categories c ON c.id = p.category_id
	WHERE p.id = ANY($1)`

	rows, err := s.db.Query(q, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make(map[int64]*Product)
	for rows.Next() {
		pr := &Product{}
		if err := rows.Scan(
			&pr.ID, &pr.CategoryID, &pr.CategoryName,
			&pr.Name, &pr.Description, &pr.UnitPrice, &pr.DistributionPrice, &pr.CreatedAt,
		); err != nil {
			return nil, err
		}
		products[pr.ID] = pr
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *PostgresProductStore) list(query string, args ...any) ([]*Product, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		pr := &Product{}
		if err := rows.Scan(
			&pr.ID, &pr.CategoryID, &pr.CategoryName,
			&pr.Name, &pr.Description, &pr.UnitPrice, &pr.DistributionPrice, &pr.CreatedAt,
			&pr.CurrentStock,
		); err != nil {
			return nil, err
		}
		products = append(products, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *PostgresProductStore) SearchProductsFTS(q string, limit, offset int) ([]*Product, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if q == "" {
		const allq = `
		SELECT p.id, p.category_id, c.name AS category_name,
		       p.name, p.description, p.unit_price, p.distribution_price, p.created_at,
		       COALESCE(ls.quantity, 0) as current_stock
		FROM products p
		JOIN categories c ON c.id = p.category_id
		LEFT JOIN local_stock ls ON ls.product_id = p.id
		ORDER BY p.name
		LIMIT $1 OFFSET $2`
		return s.list(allq, limit, offset)
	}

	safeQ := strings.Map(func(r rune) rune {
		if strings.ContainsRune("&|!():*", r) {
			return ' '
		}
		return r
	}, q)

	terms := strings.Fields(safeQ)
	if len(terms) == 0 {
		const allq = `
		SELECT p.id, p.category_id, c.name AS category_name,
		       p.name, p.description, p.unit_price, p.distribution_price, p.created_at,
		       COALESCE(ls.quantity, 0) as current_stock
		FROM products p
		JOIN categories c ON c.id = p.category_id
		LEFT JOIN local_stock ls ON ls.product_id = p.id
		ORDER BY p.name
		LIMIT $1 OFFSET $2`
		return s.list(allq, limit, offset)
	}

	var queryParts []string
	for _, term := range terms {
		queryParts = append(queryParts, term+":*")
	}
	formattedQuery := strings.Join(queryParts, " & ")

	const sqlq = `
	SELECT p.id, p.category_id, c.name AS category_name,
	       p.name, p.description, p.unit_price, p.distribution_price, p.created_at,
	       COALESCE(ls.quantity, 0) as current_stock
	FROM products p
	JOIN categories c ON c.id = p.category_id
	LEFT JOIN local_stock ls ON ls.product_id = p.id
	WHERE p.search_tsv @@ to_tsquery('spanish', unaccent($1))
	ORDER BY ts_rank(p.search_tsv, to_tsquery('spanish', unaccent($1))) DESC, p.name
	LIMIT $2 OFFSET $3`
	return s.list(sqlq, formattedQuery, limit, offset)
}

func (s *PostgresProductStore) GetTopSellingProducts(start, end time.Time) ([]*TopProduct, error) {
	query := `
	WITH combined_sales AS (
		SELECT product_id, quantity FROM local_sale_items lsi
		JOIN local_sales ls ON ls.id = lsi.local_sale_id
		WHERE ls.created_at >= $1 AND ls.created_at < $2
		UNION ALL
		SELECT product_id, quantity FROM order_products op
		JOIN orders o ON o.id = op.order_id
		WHERE o.date >= $1 AND o.date < $2 AND o.state != 'cancelled'
	)
	SELECT p.id, p.name, SUM(cs.quantity) as total_qty
	FROM combined_sales cs
	JOIN products p ON p.id = cs.product_id
	GROUP BY p.id, p.name
	ORDER BY total_qty DESC
	LIMIT 3
	`

	rows, err := s.db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topProducts []*TopProduct
	for rows.Next() {
		tp := &TopProduct{}
		if err := rows.Scan(&tp.ID, &tp.Name, &tp.Quantity); err != nil {
			return nil, err
		}
		topProducts = append(topProducts, tp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return topProducts, nil
}

func (s *PostgresProductStore) GetTopSellingProductsLocal(start, end time.Time) ([]*TopProduct, error) {
	query := `
	SELECT p.id, p.name, SUM(lsi.quantity) as total_qty
	FROM local_sale_items lsi
	JOIN local_sales ls ON ls.id = lsi.local_sale_id
	JOIN products p ON p.id = lsi.product_id
	WHERE ls.created_at >= $1 AND ls.created_at < $2
	GROUP BY p.id, p.name
	ORDER BY total_qty DESC
	LIMIT 3
	`
	rows, err := s.db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topProducts []*TopProduct
	for rows.Next() {
		tp := &TopProduct{}
		if err := rows.Scan(&tp.ID, &tp.Name, &tp.Quantity); err != nil {
			return nil, err
		}
		topProducts = append(topProducts, tp)
	}
	return topProducts, rows.Err()
}

func (s *PostgresProductStore) GetTopSellingProductsDistribution(start, end time.Time) ([]*TopProduct, error) {
	query := `
	SELECT p.id, p.name, SUM(op.quantity) as total_qty
	FROM order_products op
	JOIN orders o ON o.id = op.order_id
	JOIN products p ON p.id = op.product_id
	WHERE o.date >= $1 AND o.date < $2 AND o.state != 'cancelled'
	GROUP BY p.id, p.name
	ORDER BY total_qty DESC
	LIMIT 3
	`
	rows, err := s.db.Query(query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topProducts []*TopProduct
	for rows.Next() {
		tp := &TopProduct{}
		if err := rows.Scan(&tp.ID, &tp.Name, &tp.Quantity); err != nil {
			return nil, err
		}
		topProducts = append(topProducts, tp)
	}
	return topProducts, rows.Err()
}
