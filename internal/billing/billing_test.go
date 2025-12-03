package billing

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

var (
	once sync.Once
	db   *sql.DB
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

	var err error
	once.Do(func() {
		db, err = sql.Open("pgx", dsn)
		require.NoError(t, err)

		require.NoError(t, store.Migrate(db, "../../migrations/"))
	})
	require.NoError(t, err)

	_, err = db.Exec(`TRUNCATE order_products, orders, product_ingredients, products, categories, providers, clients, tokens, users, ingredients, payment_methods, local_stock, local_sales, local_sale_items RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return db
}

func TestGenerateInvoice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	clientStore := store.NewPostgresClientStore(db)
	categoryStore := store.NewPostgresCategoryStore(db)
	productStore := store.NewPostgresProductStore(db)

	// --- Setup ---
	client := &store.Client{
		Name:      "Cliente de Prueba Factura",
		Address:   "Calle Falsa 123",
		Type:      store.ClientTypeIndividual,
		Reference: "ref-billing-test",
		CUIT:      "cuit-billing-test",
	}
	require.NoError(t, clientStore.CreateClient(client))

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

	order := &store.Order{
		ID:       123,
		ClientID: client.ID,
		Date:     time.Date(2001, 2, 28, 8, 30, 0, 0, &time.Location{}),
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

	dateStr := order.Date.Format("2006-01-02")
	fileName := fmt.Sprintf("remito_produccion-%s.xlsx", dateStr)
	filePath := filepath.Join(invoiceDir, fileName)
	os.Remove(filePath) // Clean up before test

	tests := []struct {
		name     string
		order    *store.Order
		client   *store.Client
		products map[int64]*store.Product
		wantErr  bool
	}{
		{
			name:     "valid invoice generation",
			order:    order,
			client:   client,
			products: productsMap,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateInvoice(tt.order, tt.client, tt.products)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				_, err := os.Stat(filePath)
				require.NoError(t, err, "El archivo de factura no fue creado")
				// Clean up after successful test
				os.Remove(filePath)
			}
		})
	}
}
