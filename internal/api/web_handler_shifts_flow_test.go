package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestShiftFlow_WithSales(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	// Initialize Stores
	shiftStore := store.NewPostgresShiftStore(db)
	saleStore := store.NewPostgresLocalSaleStore(db)
	stockStore := store.NewPostgresLocalStockStore(db)
	productStore := store.NewPostgresProductStore(db)
	paymentMethodStore := store.NewPostgresPaymentMethodStore(db)
	userStore := store.NewPostgresUserStore(db)

	// Initialize Services
	localSaleService := services.NewLocalSaleService(db, saleStore, stockStore, paymentMethodStore, productStore)
	
	// Mock cashMovementStore inside shiftService? 
	// No, NewShiftService requires it.
	cashMovementStore := store.NewPostgresCashMovementStore(db)
	shiftService := services.NewShiftService(shiftStore, saleStore, cashMovementStore)
	
	// Update handler with new service
	webHandler := api.NewWebHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, localSaleService, shiftService, nil, logger,
	)

	// 1. Setup Data: User, Payment Methods, Product, Stock
	user := &store.User{Username: "cashier", Email: "c@test.com", Role: "employee", IsActive: true}
	require.NoError(t, user.PasswordHash.Set("123456"))
	require.NoError(t, userStore.CreateUser(user))

	cashMethod := &store.PaymentMethod{Name: "Efectivo"}
	require.NoError(t, paymentMethodStore.CreatePaymentMethod(cashMethod)) // ID likely 1

	cardMethod := &store.PaymentMethod{Name: "Tarjeta"}
	require.NoError(t, paymentMethodStore.CreatePaymentMethod(cardMethod)) // ID likely 2

	product := &store.Product{Name: "Coca Cola", UnitPrice: 100.0, CategoryID: 1} // Assuming cat 1 exists from setupTestDB or we ignore fk if truncated differently? 
	// setupTestDB in shared file truncates categories. We need to create one.
	db.Exec("INSERT INTO categories (id, name) VALUES (1, 'General')")
	require.NoError(t, productStore.CreateProduct(product))
	
	_, err := stockStore.Create(product.ID, 100)
	require.NoError(t, err)

	// 2. Open Shift
	startCash := 1000.0
	formOpen := url.Values{"start_cash": {strconv.FormatFloat(startCash, 'f', 2, 64)}, "notes": {"Start"}}
	reqOpen := httptest.NewRequest("POST", "/shifts/open", strings.NewReader(formOpen.Encode()))
	reqOpen.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqOpen = middleware.SetUser(reqOpen, user)
	wOpen := httptest.NewRecorder()
	
	webHandler.HandleOpenShift(wOpen, reqOpen)
	require.Equal(t, http.StatusSeeOther, wOpen.Result().StatusCode)

	// 3. Make Sales
	// Sale 1: Cash ($200) -> Should affect Shift
	sale1 := services.CreateLocalSaleRequest{
		PaymentMethodID: cashMethod.ID,
		Items: []services.CreateLocalSaleItem{{ProductID: product.ID, Quantity: 2}},
	}
	_, err = localSaleService.CreateLocalSale(sale1)
	require.NoError(t, err)

	// Sale 2: Card ($300) -> Should NOT affect Shift Cash
	sale2 := services.CreateLocalSaleRequest{
		PaymentMethodID: cardMethod.ID,
		Items: []services.CreateLocalSaleItem{{ProductID: product.ID, Quantity: 3}},
	}
	_, err = localSaleService.CreateLocalSale(sale2)
	require.NoError(t, err)

	// 3.5. Register Cash Movement (Output)
	// Withdraw 50.00 for supplies
	formMovement := url.Values{
		"amount": {"50.00"},
		"type":   {"out"},
		"reason": {"Supplies"},
	}
	reqMovement := httptest.NewRequest("POST", "/shifts/movements", strings.NewReader(formMovement.Encode()))
	reqMovement.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqMovement = middleware.SetUser(reqMovement, user)
	wMovement := httptest.NewRecorder()

	webHandler.HandleRegisterMovement(wMovement, reqMovement)
	require.Equal(t, http.StatusSeeOther, wMovement.Result().StatusCode)

	// 4. Close Shift
	// Expected: 1000 (Start) + 200 (Sale 1) - 50 (Movement Out) = 1150.
	declaredCash := 1150.0 
	formClose := url.Values{"end_cash_declared": {strconv.FormatFloat(declaredCash, 'f', 2, 64)}, "notes": {"End"}}
	reqClose := httptest.NewRequest("POST", "/shifts/close", strings.NewReader(formClose.Encode()))
	reqClose.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqClose = middleware.SetUser(reqClose, user)
	wClose := httptest.NewRecorder()

	webHandler.HandleCloseShift(wClose, reqClose)
	require.Equal(t, http.StatusSeeOther, wClose.Result().StatusCode)

	// 5. Verify Results
	// We need to fetch the shift to check the Difference and Expected fields
	shifts, err := shiftStore.ListByUserID(user.ID, 1, 0)
	require.NoError(t, err)
	require.Len(t, shifts, 1)
	
	closedShift := shifts[0]
	require.Equal(t, "closed", closedShift.Status)
	require.NotNil(t, closedShift.EndCashExpected)
	require.Equal(t, 1150.0, *closedShift.EndCashExpected, "Expected cash should be Start + CashSales - MovementsOut")
	require.Equal(t, 0.0, *closedShift.Difference, "Difference should be 0 if declared matches expected")
}
