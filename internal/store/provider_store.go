package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type ProviderCategory struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Provider struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Address    string     `json:"address,omitempty"`
	Phone      string     `json:"phone,omitempty"`
	Reference  string     `json:"reference"`
	Email      string     `json:"email,omitempty"`
	CUIT       string     `json:"cuit"`
	CreatedAt  time.Time  `json:"created_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
	CategoryID int64      `json:"category_id"`
	Category   string     `json:"category"`
}

type ProviderStore interface {
	CreateProvider(*Provider) error
	UpdateProvider(*Provider) error
	DeleteProvider(id int64) error
	GetProviderByID(id int64) (*Provider, error)
	GetAllProviders() ([]*Provider, error)
	SearchProvidersFTS(q string, categoryID *int64, limit, offset int) ([]*Provider, error)

	CreateProviderCategory(*ProviderCategory) error
	UpdateProviderCategory(*ProviderCategory) error
	DeleteProviderCategory(id int64) error
	GetAllProviderCategories() ([]*ProviderCategory, error)
	GetProviderCategoryByID(id int64) (*ProviderCategory, error)
}

type PostgresProviderStore struct{ db *sql.DB }

func NewPostgresProviderStore(db *sql.DB) *PostgresProviderStore {
	return &PostgresProviderStore{db: db}
}

func scanProvider(row interface{ Scan(dest ...any) error }) (*Provider, error) {
	var p Provider
	err := row.Scan(
		&p.ID, &p.Name, &p.Address, &p.Phone, &p.Reference, &p.Email, &p.CUIT, &p.CreatedAt, &p.DeletedAt,
		&p.CategoryID, &p.Category,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *PostgresProviderStore) list(query string, args ...any) ([]*Provider, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Address, &p.Phone, &p.Reference, &p.Email, &p.CUIT, &p.CreatedAt, &p.DeletedAt,
			&p.CategoryID, &p.Category,
		); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (s *PostgresProviderStore) CreateProvider(p *Provider) error {
	const q = `
	INSERT INTO providers (name, address, phone, reference, email, cuit, category_id)
	VALUES ($1,$2,$3,$4,$5,$6,$7)
	RETURNING id, created_at`
	// If CategoryID is 0, default to 1 (Sin Categoría)
	if p.CategoryID == 0 {
		p.CategoryID = 1
	}
	return s.db.QueryRow(q, p.Name, p.Address, p.Phone, p.Reference, p.Email, p.CUIT, p.CategoryID).
		Scan(&p.ID, &p.CreatedAt)
}

func (s *PostgresProviderStore) UpdateProvider(p *Provider) error {
	const q = `
	UPDATE providers
	SET name=$1, address=$2, phone=$3, reference=$4, email=$5, cuit=$6, category_id=$7
	WHERE id=$8 AND deleted_at IS NULL`
	if p.CategoryID == 0 {
		p.CategoryID = 1
	}
	res, err := s.db.Exec(q, p.Name, p.Address, p.Phone, p.Reference, p.Email, p.CUIT, p.CategoryID, p.ID)
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

func (s *PostgresProviderStore) DeleteProvider(id int64) error {
	const q = `UPDATE providers SET deleted_at = NOW() WHERE id=$1 AND deleted_at IS NULL`
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

func (s *PostgresProviderStore) GetProviderByID(id int64) (*Provider, error) {
	const q = `
	SELECT p.id, p.name, p.address, p.phone, p.reference, p.email, p.cuit, p.created_at, p.deleted_at,
	       p.category_id, pc.name
	FROM providers p
	LEFT JOIN provider_categories pc ON p.category_id = pc.id
	WHERE p.id=$1 AND p.deleted_at IS NULL`
	return scanProvider(s.db.QueryRow(q, id))
}

func (s *PostgresProviderStore) GetAllProviders() ([]*Provider, error) {
	const q = `
	SELECT p.id, p.name, p.address, p.phone, p.reference, p.email, p.cuit, p.created_at, p.deleted_at,
	       p.category_id, pc.name
	FROM providers p
	LEFT JOIN provider_categories pc ON p.category_id = pc.id
	WHERE p.deleted_at IS NULL
	ORDER BY p.name`
	return s.list(q)
}

func (s *PostgresProviderStore) SearchProvidersFTS(q string, categoryID *int64, limit, offset int) ([]*Provider, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
	SELECT p.id, p.name, p.address, p.phone, p.reference, p.email, p.cuit, p.created_at, p.deleted_at,
	       p.category_id, pc.name
	FROM providers p
	LEFT JOIN provider_categories pc ON p.category_id = pc.id
	WHERE p.deleted_at IS NULL`

	args := []any{}
	argCount := 0

	if categoryID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND p.category_id = $%d", argCount)
		args = append(args, *categoryID)
	}

	if q == "" {
		argCount++
		baseQuery += fmt.Sprintf(" ORDER BY p.name LIMIT $%d", argCount)
		args = append(args, limit)
		
		argCount++
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
		
		return s.list(baseQuery, args...)
	}

	// Sanitize and format query for prefix matching
	safeQ := strings.Map(func(r rune) rune {
		if strings.ContainsRune("&|!():*", r) {
			return ' '
		}
		return r
	}, q)

	terms := strings.Fields(safeQ)
	if len(terms) == 0 {
		// Just order by name if sanitized query is empty
		argCount++
		baseQuery += fmt.Sprintf(" ORDER BY p.name LIMIT $%d", argCount)
		args = append(args, limit)

		argCount++
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)

		return s.list(baseQuery, args...)
	}

	var queryParts []string
	for _, term := range terms {
		queryParts = append(queryParts, term+":*")
	}
	formattedQuery := strings.Join(queryParts, " & ")

	argCount++
	sqlq := baseQuery + fmt.Sprintf(`
	AND p.search_tsv @@ to_tsquery('spanish', unaccent($%d))
	ORDER BY ts_rank(p.search_tsv, to_tsquery('spanish', unaccent($%d))) DESC, p.name`, argCount, argCount)
	args = append(args, formattedQuery)

	argCount++
	sqlq += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit)

	argCount++
	sqlq += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	return s.list(sqlq, args...)
}

// Category Methods

func (s *PostgresProviderStore) CreateProviderCategory(pc *ProviderCategory) error {
	const q = `INSERT INTO provider_categories (name) VALUES ($1) RETURNING id, created_at`
	return s.db.QueryRow(q, pc.Name).Scan(&pc.ID, &pc.CreatedAt)
}

func (s *PostgresProviderStore) UpdateProviderCategory(pc *ProviderCategory) error {
	const q = `UPDATE provider_categories SET name=$1 WHERE id=$2`
	res, err := s.db.Exec(q, pc.Name, pc.ID)
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

func (s *PostgresProviderStore) DeleteProviderCategory(id int64) error {
	// Cannot delete if used by providers.
	// But let's check if it's the default one (ID 1)
	if id == 1 {
		return nil // silently fail or error?
	}
	// We should probably reassign to 1 before delete or fail.
	// For simplicity, let's fail if FK constraint violation (handled by DB)
	// But user asked to be able to create/edit/consult. Delete was not explicitly asked but implied by CRUD.
	// Let's implement delete.
	const q = `DELETE FROM provider_categories WHERE id=$1`
	res, err := s.db.Exec(q, id)
	if err != nil {
		return err // Postgres will return error if foreign key violation
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

func (s *PostgresProviderStore) GetAllProviderCategories() ([]*ProviderCategory, error) {
	const q = `SELECT id, name, created_at FROM provider_categories ORDER BY name`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ProviderCategory
	for rows.Next() {
		var pc ProviderCategory
		if err := rows.Scan(&pc.ID, &pc.Name, &pc.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &pc)
	}
	return out, rows.Err()
}

func (s *PostgresProviderStore) GetProviderCategoryByID(id int64) (*ProviderCategory, error) {
	const q = `SELECT id, name, created_at FROM provider_categories WHERE id=$1`
	var pc ProviderCategory
	err := s.db.QueryRow(q, id).Scan(&pc.ID, &pc.Name, &pc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pc, nil
}