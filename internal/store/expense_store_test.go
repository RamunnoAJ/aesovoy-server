package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresExpenseStore_CRUD(t *testing.T) {
	db := setupTestDB(t)
	store := NewPostgresExpenseStore(db)
	providerStore := NewPostgresProviderStore(db)

	// Ensure default category exists
	pc := &ProviderCategory{Name: "General"}
	err := providerStore.CreateProviderCategory(pc)
	require.NoError(t, err)
	
	// Ensure expense category exists
	ec := &ExpenseCategory{Name: "Supplies"}
	err = store.CreateExpenseCategory(ec)
	require.NoError(t, err)

	ec2 := &ExpenseCategory{Name: "Cleaning"}
	err = store.CreateExpenseCategory(ec2)
	require.NoError(t, err)

	// Setup provider
	p := &Provider{Name: "Test Provider", CategoryID: pc.ID}
	err = providerStore.CreateProvider(p)
	require.NoError(t, err)

	e := &Expense{
		Amount:     "150.50",
		CategoryID: ec.ID,
		Type:       ExpenseTypeProduction,
		Date:       time.Now().UTC().Truncate(time.Second),
		ProviderID: &p.ID,
	}

	// Create
	t.Run("CreateExpense", func(t *testing.T) {
		err := store.CreateExpense(e)
		require.NoError(t, err)
		assert.NotEmpty(t, e.ID)
		assert.NotEmpty(t, e.CreatedAt)
	})

	// Get
	t.Run("GetExpenseByID", func(t *testing.T) {
		got, err := store.GetExpenseByID(e.ID)
		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Equal(t, e.Amount, got.Amount)
		assert.Equal(t, e.CategoryID, got.CategoryID)
		assert.Equal(t, ec.Name, got.CategoryName)
		assert.Equal(t, e.Type, got.Type)
		assert.Equal(t, *e.ProviderID, *got.ProviderID)
		assert.Equal(t, "Test Provider", got.ProviderName)
		assert.Equal(t, e.Date.Unix(), got.Date.Unix()) // Compare Unix timestamp to avoid timezone nuances
	})

	// Update
	t.Run("UpdateExpense", func(t *testing.T) {
		e.Amount = "200.00"
		// Change category
		e.CategoryID = ec2.ID
		err := store.UpdateExpense(e)
		require.NoError(t, err)

		got, err := store.GetExpenseByID(e.ID)
		require.NoError(t, err)
		assert.Equal(t, "200.00", got.Amount)
		assert.Equal(t, ec2.ID, got.CategoryID)
		assert.Equal(t, ec2.Name, got.CategoryName)
	})

	// List
	t.Run("ListExpenses", func(t *testing.T) {
		// Create another expense
		e2 := &Expense{
			Amount:     "50.00",
			CategoryID: ec2.ID,
			Type:       ExpenseTypeLocal,
			Date:       time.Now().Add(-24 * time.Hour),
		}
		err := store.CreateExpense(e2)
		require.NoError(t, err)

		// List all
		list, err := store.ListExpenses(ExpenseFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)

		// Filter by Type
		typ := ExpenseTypeLocal
		listLocal, err := store.ListExpenses(ExpenseFilter{Type: &typ})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(listLocal), 1)
		assert.Equal(t, ExpenseTypeLocal, listLocal[0].Type)
		
		// Filter by Category
		catID := ec2.ID
		listCat, err := store.ListExpenses(ExpenseFilter{CategoryID: &catID})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(listCat), 1)
		for _, ex := range listCat {
			assert.Equal(t, ec2.ID, ex.CategoryID)
		}
	})

	// Delete
	t.Run("DeleteExpense", func(t *testing.T) {
		err := store.DeleteExpense(e.ID)
		require.NoError(t, err)

		got, err := store.GetExpenseByID(e.ID)
		require.NoError(t, err)
		assert.Nil(t, got)
	})
}