package api_test

import (
	"bytes"
	"database/sql"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	"github.com/RamunnoAJ/aesovoy-server/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// Helper to setup DB (copied/adapted from store tests)
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
	
	// We use the migrations FS to migrate
	require.NoError(t, store.MigrateFS(db, migrations.FS, "."))

	_, err = db.Exec(`TRUNCATE expenses, expense_categories, providers, provider_categories, shifts, users, tokens RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	return db
}

func TestWebHandler_Expenses(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	expenseStore := store.NewPostgresExpenseStore(db)
	providerStore := store.NewPostgresProviderStore(db)

	// Create a minimal WebHandler with necessary stores
	// We only need expenseStore and providerStore for this test
	webHandler := api.NewWebHandler(
		nil, nil, nil, nil, nil, nil, providerStore, nil, nil, expenseStore, nil, nil, nil, nil, logger,
	)

	// Create a provider category
	err := providerStore.CreateProviderCategory(&store.ProviderCategory{Name: "General"})
	require.NoError(t, err)

	// Create a provider
	err = providerStore.CreateProvider(&store.Provider{
		Name: "Test Provider",
		CategoryID: 1, // Created above, ID should be 1
	})
	require.NoError(t, err)

	// Create an expense category
	err = expenseStore.CreateExpenseCategory(&store.ExpenseCategory{Name: "Test Category"})
	require.NoError(t, err)

	testUser := &store.User{
		Username: "admin",
		Role:     "administrator",
	}

	t.Run("Create Expense via Form", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		
		writer.WriteField("date", "2023-12-01")
		writer.WriteField("type", "local")
		writer.WriteField("category_id", "1") // Created above, ID should be 1
		writer.WriteField("amount", "123.45")
		writer.WriteField("provider_id", "1")
		writer.Close()

		req := httptest.NewRequest("POST", "/expenses/new", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req = middleware.SetUser(req, testUser)
		w := httptest.NewRecorder()

		webHandler.HandleCreateExpense(w, req)

		resp := w.Result()
		require.Equal(t, http.StatusSeeOther, resp.StatusCode)
		
		// Verify DB
		expenses, err := expenseStore.ListExpenses(store.ExpenseFilter{})
		require.NoError(t, err)
		require.Len(t, expenses, 1)
		require.Equal(t, "123.45", expenses[0].Amount)
		require.Equal(t, "local", string(expenses[0].Type))
		require.Equal(t, "Test Category", expenses[0].CategoryName)
	})

	t.Run("List Expenses", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/expenses", nil)
		req = middleware.SetUser(req, testUser)
		w := httptest.NewRecorder()

		webHandler.HandleListExpenses(w, req)

		resp := w.Result()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		
		body := w.Body.String()
		require.Contains(t, body, "Test Category")
		require.Contains(t, body, "123,45")
		
		// Verify Navbar/Sidebar (rendered when User is present)
		require.Contains(t, body, "A Eso Voy")
		require.Contains(t, body, "Gastos")
		require.Contains(t, body, "admin")
	})

	t.Run("Create Expense View", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/expenses/new", nil)
		req = middleware.SetUser(req, testUser)
		w := httptest.NewRecorder()

		webHandler.HandleCreateExpenseView(w, req)

		resp := w.Result()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		
		body := w.Body.String()
		require.Contains(t, body, "Registrar Nuevo Gasto")
		require.Contains(t, body, "admin")
		require.Contains(t, body, "A Eso Voy")
	})
}
