package services

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	host := utils.Getenv("DB_TEST_HOST", "localhost")
	port := utils.Getenv("DB_TEST_PORT", "5433")
	name := utils.Getenv("DB_TEST_NAME", "test_db")
	user := utils.Getenv("DB_TEST_USER", "postgres")
	pass := utils.Getenv("DB_TEST_PASSWORD", "postgres")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, pass, name, port,
	)

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(db, "../../migrations/"))

	_, err = db.Exec(`TRUNCATE order_products, orders, product_ingredients, products, categories, providers, clients, tokens, users, ingredients, payment_methods, local_stock, local_sales, local_sale_items RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return db
}
