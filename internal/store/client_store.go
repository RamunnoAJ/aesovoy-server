package store

import (
	"database/sql"
	"strings"
	"time"
)

type ClientType string

const (
	ClientTypeDistributer ClientType = "distributer"
	ClientTypeIndividual  ClientType = "individual"
)

type Client struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Address   string     `json:"address,omitempty"`
	Phone     string     `json:"phone,omitempty"`
	Reference string     `json:"reference"`
	Email     string     `json:"email,omitempty"`
	CUIT      string     `json:"cuit"`
	Type      ClientType `json:"type"`
	CreatedAt time.Time  `json:"created_at"`
}

type ClientStore interface {
	CreateClient(*Client) error
	UpdateClient(*Client) error
	GetClientByID(id int64) (*Client, error)
	GetAllClients() ([]*Client, error)
	SearchClientsFTS(q string, limit, offset int) ([]*Client, error)
	DeleteClient(id int64) error
}

type PostgresClientStore struct {
	db *sql.DB
}

func NewPostgresClientStore(db *sql.DB) *PostgresClientStore { return &PostgresClientStore{db: db} }

func scanClient(row interface {
	Scan(dest ...any) error
}) (*Client, error) {
	var c Client
	err := row.Scan(
		&c.ID, &c.Name, &c.Address, &c.Phone, &c.Reference, &c.Email, &c.CUIT, &c.Type, &c.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *PostgresClientStore) list(query string, args ...any) ([]*Client, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var clients []*Client
	for rows.Next() {
		var c Client
		if err := rows.Scan(&c.ID, &c.Name, &c.Address, &c.Phone, &c.Reference, &c.Email, &c.CUIT, &c.Type, &c.CreatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return clients, nil
}

func (s *PostgresClientStore) CreateClient(c *Client) error {
	const q = `
	INSERT INTO clients (name, address, phone, reference, email, cuit, type)
	VALUES ($1,$2,$3,$4,$5,$6,$7)
	RETURNING id, created_at`
	return s.db.QueryRow(q, c.Name, c.Address, c.Phone, c.Reference, c.Email, c.CUIT, c.Type).
		Scan(&c.ID, &c.CreatedAt)
}

func (s *PostgresClientStore) UpdateClient(c *Client) error {
	const q = `
	UPDATE clients
	SET name=$1, address=$2, phone=$3, reference=$4, email=$5, cuit=$6, type=$7
	WHERE id=$8`
	res, err := s.db.Exec(q, c.Name, c.Address, c.Phone, c.Reference, c.Email, c.CUIT, c.Type, c.ID)
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

func (s *PostgresClientStore) GetClientByID(id int64) (*Client, error) {
	const q = `SELECT id,name,address,phone,reference,email,cuit,type,created_at FROM clients WHERE id=$1`
	return scanClient(s.db.QueryRow(q, id))
}

func (s *PostgresClientStore) GetAllClients() ([]*Client, error) {
	const q = `
	SELECT id,name,address,phone,reference,email,cuit,type,created_at
	FROM clients
	ORDER BY name`
	return s.list(q)
}

func (s *PostgresClientStore) SearchClientsFTS(q string, limit, offset int) ([]*Client, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	if q == "" {
		const allq = `
		SELECT id,name,address,phone,reference,email,cuit,type,created_at
		FROM clients
		ORDER BY name
		LIMIT $1 OFFSET $2`
		return s.list(allq, limit, offset)
	}

	// Sanitize and format query for prefix matching
	// Replace characters that might break to_tsquery syntax with spaces
	safeQ := strings.Map(func(r rune) rune {
		if strings.ContainsRune("&|!():*", r) {
			return ' '
		}
		return r
	}, q)

	terms := strings.Fields(safeQ)
	if len(terms) == 0 {
		const allq = `
		SELECT id,name,address,phone,reference,email,cuit,type,created_at
		FROM clients
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
	SELECT id,name,address,phone,reference,email,cuit,type,created_at
	FROM clients
	WHERE search_tsv @@ to_tsquery('spanish', unaccent($1))
	ORDER BY ts_rank(search_tsv, to_tsquery('spanish', unaccent($1))) DESC, name
	LIMIT $2 OFFSET $3`
	return s.list(sqlq, formattedQuery, limit, offset)
}

func (s *PostgresClientStore) DeleteClient(id int64) error {
	const q = `DELETE FROM clients WHERE id=$1`
	res, err := s.db.Exec(q, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows // Or a custom error indicating not found
	}
	return nil
}
