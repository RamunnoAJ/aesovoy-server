package services

import (
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStockService_CreateInitialStock(t *testing.T) {
	// Each test case will get a fresh DB
	db := setupTestDB(t)
	defer db.Close()

	productStore := store.NewPostgresProductStore(db)
	categoryStore := store.NewPostgresCategoryStore(db)
	localStockStore := store.NewPostgresLocalStockStore(db)
	service := NewLocalStockService(localStockStore, productStore)

	// Setup a product that exists for all subtests
	cat := &store.Category{Name: "Category For Create Test"}
	require.NoError(t, categoryStore.CreateCategory(cat))
	prod := &store.Product{CategoryID: cat.ID, Name: "Product For Create Test", UnitPrice: 1}
	require.NoError(t, productStore.CreateProduct(prod))

	// Pre-create a stock record for the "already exists" case
	_, err := localStockStore.Create(prod.ID, 50)
	require.NoError(t, err)

	tests := []struct {
		name            string
		productID       int64
		initialQuantity int
		wantErr         error
	}{
		{
			name:            "product not found",
			productID:       9999,
			initialQuantity: 10,
			wantErr:         ErrProductNotFound,
		},
		{
			name:            "stock record already exists",
			productID:       prod.ID,
			initialQuantity: 20,
			wantErr:         ErrStockRecordExists,
		},
		{
			name:            "invalid initial quantity",
			productID:       prod.ID,
			initialQuantity: -5,
			wantErr:         ErrInitialQuantityInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock, err := service.CreateInitialStock(tt.productID, tt.initialQuantity)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, stock)
		})
	}

	// Test success case separately to avoid state pollution
	t.Run("success", func(t *testing.T) {
		prod2 := &store.Product{CategoryID: cat.ID, Name: "Product 2 For Create Test", UnitPrice: 1}
		require.NoError(t, productStore.CreateProduct(prod2))

		stock, err := service.CreateInitialStock(prod2.ID, 10)
		require.NoError(t, err)
		assert.NotNil(t, stock)
		assert.Equal(t, 10, stock.Quantity)
	})
}

func TestLocalStockService_AdjustStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productStore := store.NewPostgresProductStore(db)
	categoryStore := store.NewPostgresCategoryStore(db)
	localStockStore := store.NewPostgresLocalStockStore(db)
	service := NewLocalStockService(localStockStore, productStore)

	cat := &store.Category{Name: "Category For Adjust Test"}
	require.NoError(t, categoryStore.CreateCategory(cat))
	prod := &store.Product{CategoryID: cat.ID, Name: "Product For Adjust Test", UnitPrice: 1}
	require.NoError(t, productStore.CreateProduct(prod))
	_, err := localStockStore.Create(prod.ID, 50)
	require.NoError(t, err)

	tests := []struct {
		name    string
		delta   int
		wantQty int
		wantErr error
	}{
		{
			name:    "successful decrease",
			delta:   -20,
			wantQty: 30,
			wantErr: nil,
		},
		{
			name:    "adjustment to zero",
			delta:   -30, // Current is 30
			wantQty: 0,
			wantErr: nil,
		},
		{
			name:    "successful increase",
			delta:   50, // Current is 0
			wantQty: 50,
			wantErr: nil,
		},
		{
			name:    "adjustment results in negative stock",
			delta:   -100, // Current is 50
			wantErr: ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock, err := service.AdjustStock(prod.ID, tt.delta)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, stock)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, stock)
				assert.Equal(t, tt.wantQty, stock.Quantity)

				// Verify persistence
				persisted, err := localStockStore.GetByProductID(prod.ID)
				require.NoError(t, err)
				assert.Equal(t, tt.wantQty, persisted.Quantity)
			}
		})
	}

	t.Run("adjust non-existent stock record", func(t *testing.T) {
		_, err := service.AdjustStock(9999, 10)
		assert.ErrorIs(t, err, ErrStockRecordNotFound)
	})
}

func TestLocalStockService_ListStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productStore := store.NewPostgresProductStore(db)
	categoryStore := store.NewPostgresCategoryStore(db)
	localStockStore := store.NewPostgresLocalStockStore(db)
	service := NewLocalStockService(localStockStore, productStore)

	cat := &store.Category{Name: "Category List Test"}
	require.NoError(t, categoryStore.CreateCategory(cat))

	prodA := &store.Product{CategoryID: cat.ID, Name: "Product A", UnitPrice: 10.5}
	require.NoError(t, productStore.CreateProduct(prodA))
	prodB := &store.Product{CategoryID: cat.ID, Name: "Product B", UnitPrice: 20.0}
	require.NoError(t, productStore.CreateProduct(prodB))

	// Create stock for A
	_, err := localStockStore.Create(prodA.ID, 50)
	require.NoError(t, err)

	// Test ListStock
	list, err := service.ListStock()
	require.NoError(t, err)
	require.Len(t, list, 2)

	// Verify order (by name) and content
	assert.Equal(t, prodA.ID, list[0].ProductID)
	assert.Equal(t, "Product A", list[0].ProductName)
	assert.Equal(t, 10.5, list[0].Price)
	assert.Equal(t, 50, list[0].Quantity)

	assert.Equal(t, prodB.ID, list[1].ProductID)
	assert.Equal(t, "Product B", list[1].ProductName)
	assert.Equal(t, 20.0, list[1].Price)
	assert.Equal(t, 0, list[1].Quantity)
}
