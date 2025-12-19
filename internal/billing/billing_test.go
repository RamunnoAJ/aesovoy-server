package billing

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
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

func TestListInvoices(t *testing.T) {
	// 1. Setup temporary directory for invoices
	tempDir, err := os.MkdirTemp("", "invoices_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 2. Override invoiceDir
	originalInvoiceDir := invoiceDir
	invoiceDir = tempDir
	defer func() { invoiceDir = originalInvoiceDir }()

	// 3. Create dummy invoice files
	// Create 25 files to test pagination (assuming default limit is 20, or we test with custom limit)
	baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 25; i++ {
		filename := fmt.Sprintf("invoice_%02d.xlsx", i)
		filePath := filepath.Join(tempDir, filename)
		f, err := os.Create(filePath)
		require.NoError(t, err)
		f.Close()

		// Set modification time to ensure deterministic order (newest first)
		// We want invoice_25 to be the newest
		modTime := baseTime.Add(time.Duration(i) * time.Hour)
		err = os.Chtimes(filePath, modTime, modTime)
		require.NoError(t, err)
	}

	// 4. Test Cases
	tests := []struct {
		name           string
		page           int
		limit          int
		dateFilter     string
		expectedCount  int
		expectedFirst  string // Name of the first file in the returned list
		expectedTotal  int
	}{
		{
			name:          "Page 1, Limit 10",
			page:          1,
			limit:         10,
			dateFilter:    "",
			expectedCount: 10,
			expectedFirst: "invoice_25.xlsx", // Newest first
			expectedTotal: 25,
		},
		{
			name:          "Page 2, Limit 10",
			page:          2,
			limit:         10,
			dateFilter:    "",
			expectedCount: 10,
			expectedFirst: "invoice_15.xlsx",
			expectedTotal: 25,
		},
		{
			name:          "Page 3, Limit 10",
			page:          3,
			limit:         10,
			dateFilter:    "",
			expectedCount: 5,
			expectedFirst: "invoice_05.xlsx",
			expectedTotal: 25,
		},
		{
			name:          "Page 1, Limit 50 (More than total)",
			page:          1,
			limit:         50,
			dateFilter:    "",
			expectedCount: 25,
			expectedFirst: "invoice_25.xlsx",
			expectedTotal: 25,
		},
		{
			name:          "Page 10, Limit 10 (Out of range)",
			page:          10,
			limit:         10,
			dateFilter:    "",
			expectedCount: 0,
			expectedTotal: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, total, err := ListInvoices(tt.page, tt.limit, tt.dateFilter)
			require.NoError(t, err)
			require.Equal(t, tt.expectedTotal, total)
			require.Equal(t, tt.expectedCount, len(files))

			if tt.expectedCount > 0 {
				require.Equal(t, tt.expectedFirst, files[0].Name)
			}
		})
	}
}
