package services

import (
	"database/sql"
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", "host=localhost user=postgres password=postgres dbname=postgres port=5433 sslmode=disable")
	require.NoError(t, err)

	require.NoError(t, store.Migrate(db, "../../migrations/"))
	_, err = db.Exec(`TRUNCATE order_products, orders, product_ingredients, products, categories, providers, clients, tokens, users, ingredients, payment_methods, local_stock, local_sales, local_sale_items RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return db
}
