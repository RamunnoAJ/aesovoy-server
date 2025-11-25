package store

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStockStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalStockStore(db)
	prod := setupProductForStockTest(t, db)

	tests := []struct {
		name      string
		productID int64
		quantity  int
		wantErr   bool
	}{
		{
			name:      "create valid stock",
			productID: prod.ID,
			quantity:  100,
			wantErr:   false,
		},
		{
			name:      "create duplicate stock",
			productID: prod.ID,
			quantity:  50,
			wantErr:   true, // Expect error because it's a duplicate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock, err := s.Create(tt.productID, tt.quantity)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, stock.ID)
				assert.Equal(t, tt.productID, stock.ProductID)
				assert.Equal(t, tt.quantity, stock.Quantity)
				assert.WithinDuration(t, time.Now(), stock.CreatedAt, 2*time.Second)
			}
		})
	}
}

func TestLocalStockStore_GetByProductID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalStockStore(db)
	prod := setupProductForStockTest(t, db)
	_, err := s.Create(prod.ID, 100)
	require.NoError(t, err)

	tests := []struct {
		name        string
		productID   int64
		want        *LocalStock
		wantErr     bool
		shouldExist bool
	}{
		{
			name:        "get existing stock",
			productID:   prod.ID,
			shouldExist: true,
		},
		{
			name:        "get non-existing stock",
			productID:   99999,
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stock, err := s.GetByProductID(tt.productID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.shouldExist {
				assert.NotNil(t, stock)
				assert.Equal(t, 100, stock.Quantity)
			} else {
				assert.Nil(t, stock)
			}
		})
	}
}
func TestLocalStockStore_AdjustQuantity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalStockStore(db)
	prod := setupProductForStockTest(t, db)
	_, err := s.Create(prod.ID, 100)
	require.NoError(t, err)

	tests := []struct {
		name    string
		delta   int
		wantQty int
		wantErr bool
	}{
		{name: "decrease stock", delta: -20, wantQty: 80, wantErr: false},
		{name: "increase stock", delta: 30, wantQty: 110, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjusted, err := s.AdjustQuantity(prod.ID, tt.delta)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantQty, adjusted.Quantity)

			// Verify persistence
			stock, err := s.GetByProductID(prod.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantQty, stock.Quantity)
		})
	}
}

func TestLocalStockStore_ListAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalStockStore(db)
	prod1 := setupProductForStockTest(t, db)
	prod2 := setupProductForStockTest(t, db)

	_, err := s.Create(prod1.ID, 10)
	require.NoError(t, err)
	_, err = s.Create(prod2.ID, 20)
	require.NoError(t, err)

	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
	}{
		{name: "list all two items", wantCount: 2, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := s.ListAll()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, list, tt.wantCount)
		})
	}
}

func setupProductForStockTest(t *testing.T, db *sql.DB) *Product {
	t.Helper()
	categoryStore := NewPostgresCategoryStore(db)
	productStore := NewPostgresProductStore(db)

	catName := fmt.Sprintf("Cat for Stock Test %d", time.Now().UnixNano())
	cat := &Category{Name: catName}
	require.NoError(t, categoryStore.CreateCategory(cat))

	prodName := fmt.Sprintf("Prod for Stock Test %d", time.Now().UnixNano())
	prod := &Product{
		Name:       prodName,
		CategoryID: cat.ID,
		UnitPrice:  10.0,
	}
	require.NoError(t, productStore.CreateProduct(prod))
	return prod
}
