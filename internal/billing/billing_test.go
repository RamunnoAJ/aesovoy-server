package billing

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

// setupTestDB is a helper function to set up a test database.
// It connects to the database, runs migrations, and truncates tables.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// IMPORTANT: Ensure this connection string points to your TEST database.
	db, err := sql.Open("pgx", "host=localhost user=postgres password=postgres dbname=postgres port=5433 sslmode=disable")
	require.NoError(t, err)

	// Run migrations
	err = store.MigrateFS(db, migrations.FS, ".")
	require.NoError(t, err)

	// Truncate all tables to ensure a clean state
	_, err = db.Exec(`TRUNCATE order_products, orders, product_ingredients, products, categories, providers, clients, tokens, users, ingredients RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	return db
}

func TestGenerateInvoice(t *testing.T) {
	// --- Setup ---
	db := setupTestDB(t)
	defer db.Close()

	clientStore := store.NewPostgresClientStore(db)
	categoryStore := store.NewPostgresCategoryStore(db)
	productStore := store.NewPostgresProductStore(db)

	// 1. Create a test client
	client := &store.Client{
		Name:      "Cliente de Prueba Factura",
		Address:   "Calle Falsa 123",
		Type:      store.ClientTypeIndividual,
		Reference: "ref-billing-test",
		CUIT:      "cuit-billing-test",
	}
	require.NoError(t, clientStore.CreateClient(client))

	// 2. Create a test category and products
	category := &store.Category{Name: "Productos de Prueba"}
	require.NoError(t, categoryStore.CreateCategory(category))

	product1 := &store.Product{
		CategoryID: category.ID,
		Name:       "Producto A",
		UnitPrice:  150.50,
	}
	require.NoError(t, productStore.CreateProduct(product1))

	product2 := &store.Product{
		CategoryID: category.ID,
		Name:       "Producto B",
		UnitPrice:  200.00,
	}
	require.NoError(t, productStore.CreateProduct(product2))

	// 3. Define the order details
	order := &store.Order{
		ID:       123,
		ClientID: client.ID,
		Date:     time.Now(),
		State:    store.OrderTodo,
		Items: []store.OrderItem{
			{ProductID: product1.ID, Quantity: 2, Price: "150.50"},
			{ProductID: product2.ID, Quantity: 1, Price: "200.00"},
		},
	}

	productsMap := map[int64]*store.Product{
		product1.ID: product1,
		product2.ID: product2,
	}

	// Define the expected file path
	dateStr := order.Date.Format("2006-01-02")
	fileName := fmt.Sprintf("remito_produccion-%s.xlsx", dateStr)
	filePath := filepath.Join(invoiceDir, fileName)

	// Clean up any existing file from previous failed runs
	os.Remove(filePath)

	// --- Act ---
	err := GenerateInvoice(order, client, productsMap)
	require.NoError(t, err)

	// --- Assert ---
	_, err = os.Stat(filePath)
	require.NoError(t, err, "El archivo de factura no fue creado")

	// --- Teardown ---
	// err = os.Remove(filePath)
	require.NoError(t, err, "No se pudo eliminar el archivo de factura de prueba")
}
