package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/app"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/routes"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/testutils"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock Implementations for Stores
type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) GetUserByID(id int64) (*store.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}
func (m *MockUserStore) GetUserByUsername(username string) (*store.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.User), args.Error(1)
}
func (m *MockUserStore) CreateUser(user *store.User) error {
	args := m.Called(user)
	return args.Error(0)
}
func (m *MockUserStore) UpdateUser(user *store.User) error {
	args := m.Called(user)
	return args.Error(0)
}
func (m *MockUserStore) DeleteUser(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *MockUserStore) ListUsers() ([]*store.User, error) {
	args := m.Called()
	return args.Get(0).([]*store.User), args.Error(1)
}
func (m *MockUserStore) ToggleUserStatus(id int64, isActive bool) error {
	args := m.Called(id, isActive)
	return args.Error(0)
}

type MockTokenStore struct {
	mock.Mock
}

func (m *MockTokenStore) Insert(token *store.Token) error {
	args := m.Called(token)
	return args.Error(0)
}
func (m *MockTokenStore) GetByPlaintext(plaintext string) (*store.Token, error) {
	args := m.Called(plaintext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Token), args.Error(1)
}
func (m *MockTokenStore) DeleteAllForUser(userID int64, scope string) error {
	args := m.Called(userID, scope)
	return args.Error(0)
}

type MockProductStore struct {
	mock.Mock
}

func (m *MockProductStore) CreateProduct(product *store.Product) error {
	args := m.Called(product)
	return args.Error(0)
}
func (m *MockProductStore) GetProductByID(id int64) (*store.Product, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Product), args.Error(1)
}
func (m *MockProductStore) UpdateProduct(product *store.Product) error {
	args := m.Called(product)
	return args.Error(0)
}
func (m *MockProductStore) DeleteProduct(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *MockProductStore) GetAllProduct() ([]*store.Product, error) {
	args := m.Called()
	return args.Get(0).([]*store.Product), args.Error(1)
}
func (m *MockProductStore) GetProductsByCategoryID(categoryID int64) ([]*store.Product, error) {
	args := m.Called(categoryID)
	return args.Get(0).([]*store.Product), args.Error(1)
}
func (m *MockProductStore) AddIngredientToProduct(productID int64, ingredientID int64, quantity float64, unit string) (*store.ProductIngredient, error) {
	args := m.Called(productID, ingredientID, quantity, unit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.ProductIngredient), args.Error(1)
}
func (m *MockProductStore) UpdateProductIngredient(productID, ingredientID int64, quantity float64, unit string) (*store.ProductIngredient, error) {
	args := m.Called(productID, ingredientID, quantity, unit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.ProductIngredient), args.Error(1)
}
func (m *MockProductStore) RemoveIngredientFromProduct(productID, ingredientID int64) error {
	args := m.Called(productID, ingredientID)
	return args.Error(0)
}
func (m *MockProductStore) GetProductsByIDs(ids []int64) (map[int64]*store.Product, error) {
	args := m.Called(ids)
	return args.Get(0).(map[int64]*store.Product), args.Error(1)
}
func (m *MockProductStore) SearchProductsFTS(q string, limit, offset int) ([]*store.Product, error) {
	args := m.Called(q, limit, offset)
	return args.Get(0).([]*store.Product), args.Error(1)
}
func (m *MockProductStore) GetTopSellingProducts(start, end time.Time) ([]*store.TopProduct, error) {
	args := m.Called(start, end)
	return args.Get(0).([]*store.TopProduct), args.Error(1)
}
func (m *MockProductStore) GetTopSellingProductsLocal(start, end time.Time) ([]*store.TopProduct, error) {
	args := m.Called(start, end)
	return args.Get(0).([]*store.TopProduct), args.Error(1)
}
func (m *MockProductStore) GetTopSellingProductsDistribution(start, end time.Time) ([]*store.TopProduct, error) {
	args := m.Called(start, end)
	return args.Get(0).([]*store.TopProduct), args.Error(1)
}

type MockOrderStore struct {
	mock.Mock
}

func (m *MockOrderStore) CreateOrder(o *store.Order, items []store.OrderItem) error {
	args := m.Called(o, items)
	return args.Error(0)
}
func (m *MockOrderStore) UpdateOrderState(id int64, state store.OrderState) error {
	args := m.Called(id, state)
	return args.Error(0)
}
func (m *MockOrderStore) GetOrderByID(id int64) (*store.Order, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Order), args.Error(1)
}
func (m *MockOrderStore) ListOrders(f store.OrderFilter) ([]*store.Order, error) {
	args := m.Called(f)
	return args.Get(0).([]*store.Order), args.Error(1)
}
func (m *MockOrderStore) GetStats(start, end time.Time) (*store.DailyOrderStats, error) {
	args := m.Called(start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.DailyOrderStats), args.Error(1)
}
func (m *MockOrderStore) GetPendingProductionRequirements() ([]*store.ProductionRequirement, error) {
	args := m.Called()
	return args.Get(0).([]*store.ProductionRequirement), args.Error(1)
}

type MockLocalSaleService struct {
	mock.Mock
}

func (m *MockLocalSaleService) CreateLocalSale(req api.CreateLocalSaleRequest) (*store.LocalSale, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.LocalSale), args.Error(1)
}
func (m *MockLocalSaleService) GetSale(id int64) (*store.LocalSale, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.LocalSale), args.Error(1)
}
func (m *MockLocalSaleService) ListSales() ([]*store.LocalSale, error) {
	args := m.Called()
	return args.Get(0).([]*store.LocalSale), args.Error(1)
}
func (m *MockLocalSaleService) ListSalesByDate(date time.Time) ([]*store.LocalSale, error) {
	args := m.Called(date)
	return args.Get(0).([]*store.LocalSale), args.Error(1)
}
func (m *MockLocalSaleService) GetStats(start, end time.Time) (*store.DailySalesStats, error) {
	args := m.Called(start, end)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.DailySalesStats), args.Error(1)
}

// MockIngredientStore
type MockIngredientStore struct {
	mock.Mock
}

func (m *MockIngredientStore) CreateIngredient(ingredient *store.Ingredient) error {
	args := m.Called(ingredient)
	return args.Error(0)
}
func (m *MockIngredientStore) GetIngredientByID(id int64) (*store.Ingredient, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Ingredient), args.Error(1)
}
func (m *MockIngredientStore) UpdateIngredient(ingredient *store.Ingredient) error {
	args := m.Called(ingredient)
	return args.Error(0)
}
func (m *MockIngredientStore) DeleteIngredient(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}
func (m *MockIngredientStore) GetAllIngredients() ([]*store.Ingredient, error) {
	args := m.Called()
	return args.Get(0).([]*store.Ingredient), args.Error(1)
}

// setupTestApp creates a new chi.Mux router with mocked dependencies.
func setupTestApp(t *testing.T) (*chi.Mux, *MockUserStore, *MockTokenStore, *MockProductStore, *MockOrderStore, *MockLocalSaleService, *MockIngredientStore) {
	mockUserStore := new(MockUserStore)
	mockTokenStore := new(MockTokenStore)
	mockProductStore := new(MockProductStore)
	mockOrderStore := new(MockOrderStore)
	mockLocalSaleService := new(MockLocalSaleService)
	mockIngredientStore := new(MockIngredientStore)

	// Minimal logger for tests
	testLogger := testutils.NewTestLogger(t)

	// Create an app instance with mocked stores
	testApp := &app.Application{
		Config: app.Config{
			Port: 8080,
			Env:  "test",
		},
		Logger: testLogger,
		// No DB connection needed for web handlers, as stores are mocked
		// db is nil

		// Handlers (partially mocked)
		WebHandler: api.NewWebHandler(
			mockUserStore,
			mockTokenStore,
			mockProductStore,
			nil, // categoryStore, not directly used in these handlers
			mockIngredientStore,
			nil, // clientStore
			nil, // providerStore
			nil, // paymentMethodStore
			mockOrderStore,
			nil, // localStockService
			mockLocalSaleService,
			nil, // mailer
			testLogger,
		),
		Middleware: middleware.New(mockUserStore, mockTokenStore, testLogger),
	}

	// Override the renderer to use a test template filesystem
	testApp.WebHandler.Renderer = views.NewRendererFS(template.FuncMap{
		"formatMoney": func(amount float64) string {
			return fmt.Sprintf("$%.2f", amount)
		},
	}, "./../../internal/views/templates") // Adjust path as needed for test execution environment

	// Setup routes
	r := routes.SetupRoutes(testApp)

	return r, mockUserStore, mockTokenStore, mockProductStore, mockOrderStore, mockLocalSaleService, mockIngredientStore
}

// Helper function to create a request and record the response
func executeRequest(r *chi.Mux, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// Helper to add a user to context for authenticated routes
func addContextUser(req *http.Request, user *store.User) *http.Request {
	ctx := middleware.NewContextWithUser(req.Context(), user)
	return req.WithContext(ctx)
}

func TestHandleHomeAdmin(t *testing.T) {
	r, mockUserStore, _, mockProductStore, mockOrderStore, mockLocalSaleService, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore, mockOrderStore, mockLocalSaleService)

	adminUser := &store.User{ID: 1, Username: "admin", Role: "administrator"}

	// Mock expectations for HandleHome for an admin
	mockLocalSaleService.On("GetStats", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(&store.DailySalesStats{TotalAmount: 100.0, TotalCount: 1, ByMethod: map[string]float64{"Cash": 100.0}}, nil)
	mockOrderStore.On("GetStats", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(&store.DailyOrderStats{TotalAmount: 200.0, TotalCount: 2}, nil)
	mockProductStore.On("GetTopSellingProducts", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*store.TopProduct{{ID: 1, Name: "Product A", Quantity: 5}}, nil)
	mockProductStore.On("GetTopSellingProductsDistribution", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*store.TopProduct{{ID: 2, Name: "Product B", Quantity: 3}}, nil)
	mockOrderStore.On("GetPendingProductionRequirements").Return([]*store.ProductionRequirement{{ProductID: 1, ProductName: "Product A", Quantity: 10}}, nil)
	mockProductStore.On("GetTopSellingProductsLocal", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*store.TopProduct{{ID: 3, Name: "Product C", Quantity: 7}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = addContextUser(req, adminUser)
	rr := executeRequest(r, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Panel de Control")
	assert.Contains(t, rr.Body.String(), "Total Facturado")
	assert.Contains(t, rr.Body.String(), "Producción Pendiente")
	assert.Contains(t, rr.Body.String(), "Product A</td>\n                        <td class=\"px-4 py-2 whitespace-nowrap text-sm text-gray-900 text-right\">10")
}

func TestHandleHomeEmployee(t *testing.T) {
	r, mockUserStore, _, mockProductStore, mockOrderStore, mockLocalSaleService, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore, mockOrderStore, mockLocalSaleService)

	employeeUser := &store.User{ID: 2, Username: "employee", Role: "employee"}

	// Mock expectations for HandleHome for an employee
	mockLocalSaleService.On("GetStats", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(&store.DailySalesStats{TotalAmount: 100.0, TotalCount: 1, ByMethod: map[string]float64{"Cash": 100.0}}, nil)
	mockProductStore.On("GetTopSellingProductsLocal", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*store.TopProduct{{ID: 3, Name: "Product C", Quantity: 7}}, nil)
	// For employee, these should not be called, but we define them to ensure `On` is not triggered unexpectedly
	mockOrderStore.On("GetStats", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(&store.DailyOrderStats{}, nil).Maybe()
	mockProductStore.On("GetTopSellingProducts", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*store.TopProduct{}, nil).Maybe()
	mockProductStore.On("GetTopSellingProductsDistribution", mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return([]*store.TopProduct{}, nil).Maybe()
	mockOrderStore.On("GetPendingProductionRequirements").Return([]*store.ProductionRequirement{}, nil).Maybe()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = addContextUser(req, employeeUser)
	rr := executeRequest(r, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Panel de Control")
	assert.Contains(t, rr.Body.String(), "Ventas Totales") // Employee sees "Ventas Totales" instead of "Total Facturado"
	assert.NotContains(t, rr.Body.String(), "Producción Pendiente")
}

func TestHandleShowProductionCalculator(t *testing.T) {
	r, mockUserStore, _, mockProductStore, _, _, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore)

	anyUser := &store.User{ID: 1, Username: "testuser", Role: "employee"}
	products := []*store.Product{
		{ID: 1, Name: "Product A"},
		{ID: 2, Name: "Product B"},
	}

	mockProductStore.On("GetAllProduct").Return(products, nil)

	req := httptest.NewRequest(http.MethodGet, "/production-calculator", nil)
	req = addContextUser(req, anyUser)
	rr := executeRequest(r, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Calculadora de Producción")
	assert.Contains(t, rr.Body.String(), "<option value=\"1\">Product A</option>")
	assert.Contains(t, rr.Body.String(), "<option value=\"2\">Product B</option>")
}

func TestHandleCalculateProduction(t *testing.T) {
	r, mockUserStore, _, mockProductStore, _, _, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore)

	anyUser := &store.User{ID: 1, Username: "testuser", Role: "employee"}

	// Mock product with recipe
	productA := &store.Product{
		ID:   1,
		Name: "Product A",
		Recipe: []*store.ProductIngredient{
			{IngredientID: 101, Name: "Ingr X", Quantity: 0.5, Unit: "g"},
			{IngredientID: 102, Name: "Ingr Y", Quantity: 200, Unit: "gr"},
		},
	}

	mockProductStore.On("GetProductByID", int64(1)).Return(productA, nil)
	mockProductStore.On("GetAllProduct").Return([]*store.Product{}, nil).Maybe() // For error rendering path

	form := url.Values{}
	form.Add("product_id", "1")
	form.Add("quantity", "10")

	req := httptest.NewRequest(http.MethodPost, "/production-calculator", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = addContextUser(req, anyUser)
	rr := executeRequest(r, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Ingredientes Necesarios para 10 unidades de Product A")
	assert.Contains(t, body, "5.00 g de Ingr X")     // 0.5 * 10
	assert.Contains(t, body, "2000.00 gr de Ingr Y") // 200 * 10
}

func TestHandleCalculateProduction_InvalidInput(t *testing.T) {
	r, mockUserStore, _, mockProductStore, _, _, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore)

	anyUser := &store.User{ID: 1, Username: "testuser", Role: "employee"}

	// Mock for GetAllProduct to ensure dropdown is present even on error
	mockProductStore.On("GetAllProduct").Return([]*store.Product{}, nil)

	tests := []struct {
		name string
		form url.Values
		want string
	}{
		{"invalid product_id", url.Values{"product_id": {"abc"}, "quantity": {"10"}}, "ID de producto inválido"},
		{"invalid quantity", url.Values{"product_id": {"1"}, "quantity": {"abc"}}, "Cantidad inválida"},
		{"zero quantity", url.Values{"product_id": {"1"}, "quantity": {"0"}}, "Cantidad inválida"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/production-calculator", strings.NewReader(tt.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req = addContextUser(req, anyUser)
			rr := executeRequest(r, req)

			assert.Equal(t, http.StatusOK, rr.Code) // Rendered with error on the page
			assert.Contains(t, rr.Body.String(), tt.want)
		})
	}
}

func TestHandleCalculateProduction_ProductNotFound(t *testing.T) {
	r, mockUserStore, _, mockProductStore, _, _, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore)

	anyUser := &store.User{ID: 1, Username: "testuser", Role: "employee"}

	mockProductStore.On("GetProductByID", int64(999)).Return(nil, nil) // Product not found
	mockProductStore.On("GetAllProduct").Return([]*store.Product{}, nil)

	form := url.Values{}
	form.Add("product_id", "999")
	form.Add("quantity", "10")

	req := httptest.NewRequest(http.MethodPost, "/production-calculator", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = addContextUser(req, anyUser)
	rr := executeRequest(r, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Producto no encontrado")
}

func TestHandleShowPendingProductionIngredients(t *testing.T) {
	r, mockUserStore, _, mockProductStore, mockOrderStore, _, _ := setupTestApp(t)
	defer mock.AssertExpectationsForObjects(t, mockUserStore, mockProductStore, mockOrderStore)

	adminUser := &store.User{ID: 1, Username: "admin", Role: "administrator"}

	// Mock pending production requirements
	pendingReqs := []*store.ProductionRequirement{
		{ProductID: 1, ProductName: "Product Alpha", Quantity: 5},
		{ProductID: 2, ProductName: "Product Beta", Quantity: 2},
	}
	mockOrderStore.On("GetPendingProductionRequirements").Return(pendingReqs, nil)

	// Mock products with recipes
	productAlpha := &store.Product{
		ID:   1,
		Name: "Product Alpha",
		Recipe: []*store.ProductIngredient{
			{IngredientID: 101, Name: "Flour", Quantity: 0.1, Unit: "g"},
			{IngredientID: 102, Name: "Sugar", Quantity: 0.05, Unit: "g"},
		},
	}
	productBeta := &store.Product{
		ID:   2,
		Name: "Product Beta",
		Recipe: []*store.ProductIngredient{
			{IngredientID: 101, Name: "Flour", Quantity: 0.2, Unit: "g"},
			{IngredientID: 103, Name: "Butter", Quantity: 0.1, Unit: "g"},
		},
	}
	mockProductStore.On("GetProductByID", int64(1)).Return(productAlpha, nil)
	mockProductStore.On("GetProductByID", int64(2)).Return(productBeta, nil)

	req := httptest.NewRequest(http.MethodGet, "/pending-production-ingredients", nil)
	req = addContextUser(req, adminUser)
	rr := executeRequest(r, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()
	assert.Contains(t, body, "Ingredientes Totales para Producción Pendiente")

	// Expected aggregated quantities:
	// Flour: (0.1 * 5) + (0.2 * 2) = 0.5 + 0.4 = 0.9 g
	// Sugar: (0.05 * 5) = 0.25 g
	// Butter: (0.1 * 2) = 0.2 g

	assert.Contains(t, body, "0.90 g de Flour")
	assert.Contains(t, body, "0.25 g de Sugar")
	assert.Contains(t, body, "0.20 g de Butter")

	// Check if product details are also rendered
	assert.Contains(t, body, "Product Alpha")
	assert.Contains(t, body, "Product Beta")
	assert.Contains(t, body, "Cantidad Requerida")
	assert.Contains(t, body, "5</td>")
	assert.Contains(t, body, "2</td>")
}
