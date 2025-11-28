package store

import (
	"database/sql"
	"strings"
	"time"
)

type Provider struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Reference string    `json:"reference"`
	Email     string    `json:"email,omitempty"`
	CUIT      string    `json:"cuit"`
	CreatedAt time.Time `json:"created_at"`
}

type ProviderStore interface {
	CreateProvider(*Provider) error
	UpdateProvider(*Provider) error
	DeleteProvider(id int64) error
	GetProviderByID(id int64) (*Provider, error)
	GetAllProviders() ([]*Provider, error)
	SearchProvidersFTS(q string, limit, offset int) ([]*Provider, error)
}

type PostgresProviderStore struct{ db *sql.DB }

func NewPostgresProviderStore(db *sql.DB) *PostgresProviderStore {
	return &PostgresProviderStore{db: db}
}

func scanProvider(row interface{ Scan(dest ...any) error }) (*Provider, error) {
	var p Provider
	err := row.Scan(&p.ID, &p.Name, &p.Address, &p.Phone, &p.Reference, &p.Email, &p.CUIT, &p.CreatedAt)
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Address, &p.Phone, &p.Reference, &p.Email, &p.CUIT, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (s *PostgresProviderStore) CreateProvider(p *Provider) error {
	const q = `
	INSERT INTO providers (name, address, phone, reference, email, cuit)
	VALUES ($1,$2,$3,$4,$5,$6)
	RETURNING id, created_at`
	return s.db.QueryRow(q, p.Name, p.Address, p.Phone, p.Reference, p.Email, p.CUIT).
		Scan(&p.ID, &p.CreatedAt)
}

func (s *PostgresProviderStore) UpdateProvider(p *Provider) error {
	const q = `
	UPDATE providers
	SET name=$1, address=$2, phone=$3, reference=$4, email=$5, cuit=$6
	WHERE id=$7`
	res, err := s.db.Exec(q, p.Name, p.Address, p.Phone, p.Reference, p.Email, p.CUIT, p.ID)
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
	const q = `DELETE FROM providers WHERE id=$1`
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
	const q = `SELECT id,name,address,phone,reference,email,cuit,created_at FROM providers WHERE id=$1`
	return scanProvider(s.db.QueryRow(q, id))
}

func (s *PostgresProviderStore) GetAllProviders() ([]*Provider, error) {
	const q = `
	SELECT id,name,address,phone,reference,email,cuit,created_at
	FROM providers
	ORDER BY name`
	return s.list(q)
}

func (s *PostgresProviderStore) SearchProvidersFTS(q string, limit, offset int) ([]*Provider, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if q == "" {
		const allq = `
		SELECT id,name,address,phone,reference,email,cuit,created_at
		FROM providers
		ORDER BY name
		LIMIT $1 OFFSET $2`
		return s.list(allq, limit, offset)
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
		const allq = `
		SELECT id,name,address,phone,reference,email,cuit,created_at
		FROM providers
		ORDER BY name
		LIMIT $1 OFFSET $2`
		return s.list(allq, limit, offset)
	}

	var queryParts []string
	for _, term := range terms {
		queryParts = append(queryParts, term+":*")
	}
	formattedQuery := strings.Join(queryParts, " & ")

	const sqlq = `
	SELECT id,name,address,phone,reference,email,cuit,created_at
	FROM providers
	WHERE search_tsv @@ to_tsquery('spanish', unaccent($1))
	ORDER BY ts_rank(search_tsv, to_tsquery('spanish', unaccent($1))) DESC, name
	LIMIT $2 OFFSET $3`
	return s.list(sqlq, formattedQuery, limit, offset)
}
