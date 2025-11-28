package services

import (
	"errors"
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test is now an integration test, as unit testing the transactional logic
// requires a real database connection with the current service design.
func TestLocalSaleService_CreateLocalSale_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Initialize real stores and services
	productStore := store.NewPostgresProductStore(db)
	categoryStore := store.NewPostgresCategoryStore(db)
	paymentMethodStore := store.NewPostgresPaymentMethodStore(db)
	localStockStore := store.NewPostgresLocalStockStore(db)
	localSaleStore := store.NewPostgresLocalSaleStore(db)

	service := NewLocalSaleService(db, localSaleStore, localStockStore, paymentMethodStore, productStore)

	// --- Setup Data ---
	cat := &store.Category{Name: "Category For Sale Test"}
	require.NoError(t, categoryStore.CreateCategory(cat))

	pm := &store.PaymentMethod{Name: "sale-tester", Reference: "cash"}
	require.NoError(t, paymentMethodStore.CreatePaymentMethod(pm))

	prod10 := &store.Product{CategoryID: cat.ID, Name: "Product 10", UnitPrice: 100}
	require.NoError(t, productStore.CreateProduct(prod10))

	prod20 := &store.Product{CategoryID: cat.ID, Name: "Product 20", UnitPrice: 50}
	require.NoError(t, productStore.CreateProduct(prod20))

	// Create initial stock
	_, err := localStockStore.Create(prod10.ID, 10)
	require.NoError(t, err)
	_, err = localStockStore.Create(prod20.ID, 5)
	require.NoError(t, err)

	tests := []struct {
		name      string
		req       CreateLocalSaleRequest
		wantErr   error
		wantTotal string
		postCheck func(t *testing.T) // Optional check to run after the test
	}{
		{
			name: "successful sale",
			req: CreateLocalSaleRequest{
				PaymentMethodID: pm.ID,
				Items: []CreateLocalSaleItem{
					{ProductID: prod10.ID, Quantity: 2}, // 2 * 100 = 200
					{ProductID: prod20.ID, Quantity: 1}, // 1 * 50 = 50
				},
			},
			wantErr:   nil,
			wantTotal: "250.00",
			postCheck: func(t *testing.T) {
				// Check if stock was correctly deduced
				stock10, err := localStockStore.GetByProductID(prod10.ID)
				require.NoError(t, err)
				assert.Equal(t, 8, stock10.Quantity) // 10 - 2

				stock20, err := localStockStore.GetByProductID(prod20.ID)
				require.NoError(t, err)
				assert.Equal(t, 4, stock20.Quantity) // 5 - 1
			},
		},
		{
			name: "payment method not found",
			req: CreateLocalSaleRequest{
				PaymentMethodID: 99,
				Items:           []CreateLocalSaleItem{{ProductID: 10, Quantity: 1}}, // Must be non-empty
			},
			// Variable updated to Spanish in service
			wantErr: ErrPaymentMethodNotFound,
		},
		{
			name: "product not found",
			req: CreateLocalSaleRequest{
				PaymentMethodID: pm.ID,
				Items:           []CreateLocalSaleItem{{ProductID: 99, Quantity: 1}},
			},
			wantErr: errors.New("producto no encontrado"),
		},
		{
			name: "insufficient stock",
			req: CreateLocalSaleRequest{
				PaymentMethodID: pm.ID,
				Items:           []CreateLocalSaleItem{{ProductID: prod10.ID, Quantity: 11}}, // Only 10 in stock
			},
			wantErr: errors.New("stock insuficiente"),
		},
		{
			name: "sale with no items",
			req: CreateLocalSaleRequest{
				PaymentMethodID: pm.ID,
				Items:           []CreateLocalSaleItem{},
			},
			wantErr: errors.New("la venta debe tener al menos un ítem"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sale, err := service.CreateLocalSale(tt.req)

			if tt.wantErr != nil {
				assert.Error(t, err)
				// Check substring match for Spanish messages
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				require.NoError(t, err)
				assert.NotNil(t, sale)
				assert.Equal(t, tt.wantTotal, sale.Total)
				if tt.postCheck != nil {
					tt.postCheck(t)
				}
			}
		})
	}
}
