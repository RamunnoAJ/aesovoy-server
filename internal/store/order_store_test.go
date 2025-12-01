package store

import (
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)

	client := &Client{Name: "Test Client", Type: ClientTypeIndividual, Reference: "ref-order", CUIT: "cuit-order"}
	require.NoError(t, clientStore.CreateClient(client))
	category := &Category{Name: "Test Category"}
	require.NoError(t, categoryStore.CreateCategory(category))
	product1 := &Product{CategoryID: category.ID, Name: "Test Product 1", UnitPrice: 10.0, DistributionPrice: 8.0}
	require.NoError(t, productStore.CreateProduct(product1))

	tests := []struct {
		name    string
		order   *Order
		items   []OrderItem
		wantErr bool
		wantSum string
	}{
		{
			name:  "valid order",
			order: &Order{ClientID: client.ID, State: OrderTodo},
			items: []OrderItem{
				{ProductID: product1.ID, Quantity: 2, Price: "10.0"},
			},
			wantErr: false,
			wantSum: "20.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orderStore.CreateOrder(tt.order, tt.items)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, tt.order.ID)
			assert.Equal(t, tt.wantSum, tt.order.Total)
		})
	}
}

func TestGetOrderByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)

	client := &Client{Name: "Test Client", Type: ClientTypeIndividual, Reference: "ref-order", CUIT: "cuit-order"}
	require.NoError(t, clientStore.CreateClient(client))
	category := &Category{Name: "Test Category"}
	require.NoError(t, categoryStore.CreateCategory(category))
	product1 := &Product{CategoryID: category.ID, Name: "Test Product 1", UnitPrice: 10.0, DistributionPrice: 8.0}
	require.NoError(t, productStore.CreateProduct(product1))
	order := &Order{ClientID: client.ID, State: OrderTodo}
	items := []OrderItem{{ProductID: product1.ID, Quantity: 1, Price: "10"}}
	require.NoError(t, orderStore.CreateOrder(order, items))

	tests := []struct {
		name      string
		orderID   int64
		wantFound bool
		wantErr   bool
	}{
		{name: "found", orderID: order.ID, wantFound: true, wantErr: false},
		{name: "not found", orderID: 999, wantFound: false, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orderStore.GetOrderByID(tt.orderID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantFound {
				assert.NotNil(t, got)
				assert.Len(t, got.Items, 1)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestUpdateOrderState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)

	client := &Client{Name: "Client", Type: ClientTypeIndividual, Reference: "ref-order-update", CUIT: "cuit-order-update"}
	require.NoError(t, clientStore.CreateClient(client))
	cat := &Category{Name: "Category"}
	require.NoError(t, categoryStore.CreateCategory(cat))
	prod := &Product{CategoryID: cat.ID, Name: "Product", UnitPrice: 1.0, DistributionPrice: 1.0}
	require.NoError(t, productStore.CreateProduct(prod))
	order := &Order{ClientID: client.ID, State: OrderTodo}
	items := []OrderItem{{ProductID: prod.ID, Quantity: 1, Price: "1.0"}}
	require.NoError(t, orderStore.CreateOrder(order, items))

	tests := []struct {
		name     string
		newState OrderState
		wantErr  bool
	}{
		{name: "update to done", newState: OrderDone, wantErr: false},
		{name: "update to cancelled", newState: OrderCancelled, wantErr: false},
		{name: "update to delivered", newState: OrderDelivered, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := orderStore.UpdateOrderState(order.ID, tt.newState)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			updatedOrder, err := orderStore.GetOrderByID(order.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.newState, updatedOrder.State)
		})
	}
}

func TestListOrders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)

	client1 := &Client{Name: "Client 1", Type: ClientTypeIndividual, Reference: "ref-list-1", CUIT: "cuit-list-1"}
	require.NoError(t, clientStore.CreateClient(client1))
	client2 := &Client{Name: "Client 2", Type: ClientTypeIndividual, Reference: "ref-list-2", CUIT: "cuit-list-2"}
	require.NoError(t, clientStore.CreateClient(client2))
	cat := &Category{Name: "Category"}
	require.NoError(t, categoryStore.CreateCategory(cat))
	prod := &Product{CategoryID: cat.ID, Name: "Product", UnitPrice: 1.0, DistributionPrice: 1.0}
	require.NoError(t, productStore.CreateProduct(prod))
	item := []OrderItem{{ProductID: prod.ID, Quantity: 1, Price: "1.0"}}

	require.NoError(t, orderStore.CreateOrder(&Order{ClientID: client1.ID, State: OrderTodo}, item))
	require.NoError(t, orderStore.CreateOrder(&Order{ClientID: client1.ID, State: OrderDone}, item))
	require.NoError(t, orderStore.CreateOrder(&Order{ClientID: client2.ID, State: OrderTodo}, item))

	todoState := OrderTodo
	doneState := OrderDone

	tests := []struct {
		name      string
		clientID  *int64
		state     *OrderState
		wantCount int
		wantErr   bool
	}{
		{name: "list all", clientID: nil, state: nil, wantCount: 3, wantErr: false},
		{name: "list for client 1", clientID: &client1.ID, state: nil, wantCount: 2, wantErr: false},
		{name: "list with state todo", clientID: nil, state: &todoState, wantCount: 2, wantErr: false},
		{name: "list for client 1 with state done", clientID: &client1.ID, state: &doneState, wantCount: 1, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, err := orderStore.ListOrders(
				OrderFilter{
					ClientID: tt.clientID,
					State:    tt.state,
					Limit:    10,
					Offset:   0,
				},
			)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, orders, tt.wantCount)
		})
	}
}

func TestGetStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)

	// Setup
	client := &Client{Name: "Client", Type: ClientTypeIndividual, Reference: "ref-stats", CUIT: "cuit-stats"}
	require.NoError(t, clientStore.CreateClient(client))
	cat := &Category{Name: "Category"}
	require.NoError(t, categoryStore.CreateCategory(cat))
	prod := &Product{CategoryID: cat.ID, Name: "Product", UnitPrice: 100.0, DistributionPrice: 100.0}
	require.NoError(t, productStore.CreateProduct(prod))

	createOrder := func(state OrderState, date time.Time) {
		o := &Order{ClientID: client.ID, State: state}
		items := []OrderItem{{ProductID: prod.ID, Quantity: 1, Price: "100.0"}}
		require.NoError(t, orderStore.CreateOrder(o, items))
		_, err := db.Exec("UPDATE orders SET date = $1 WHERE id = $2", date, o.ID)
		require.NoError(t, err)
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	// Create orders
	createOrder(OrderTodo, now)       // +100
	createOrder(OrderDone, now)       // +100
	createOrder(OrderCancelled, now)  // Ignored
	createOrder(OrderTodo, yesterday) // Ignored

	// Test
	stats, err := orderStore.GetStats(todayStart, todayEnd)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify
	assert.Equal(t, 2, stats.TotalCount)
	assert.Equal(t, 200.00, stats.TotalAmount)
}

func TestGetPendingProductionRequirements(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)
	ingredientStore := NewPostgresIngredientStore(db)

	// Create a client
	client := &Client{Name: "Prod Req Client", Type: ClientTypeIndividual, Reference: "prc-ref", CUIT: "prc-cuit"}
	require.NoError(t, clientStore.CreateClient(client))

	// Create categories
	cat1 := &Category{Name: "Category 1"}
	require.NoError(t, categoryStore.CreateCategory(cat1))
	cat2 := &Category{Name: "Category 2"}
	require.NoError(t, categoryStore.CreateCategory(cat2))

	// Create ingredients
	ing1 := &Ingredient{Name: "Ingredient A"}
	require.NoError(t, ingredientStore.CreateIngredient(ing1))
	ing2 := &Ingredient{Name: "Ingredient B"}
	require.NoError(t, ingredientStore.CreateIngredient(ing2))

	// Create products
	prod1 := &Product{CategoryID: cat1.ID, Name: "Product Alpha", UnitPrice: 10.0, DistributionPrice: 8.0}
	require.NoError(t, productStore.CreateProduct(prod1))
	_, err := productStore.AddIngredientToProduct(prod1.ID, ing1.ID, 0.5, "kg")
	require.NoError(t, err)

	prod2 := &Product{CategoryID: cat1.ID, Name: "Product Beta", UnitPrice: 15.0, DistributionPrice: 12.0}
	require.NoError(t, productStore.CreateProduct(prod2))
	_, err = productStore.AddIngredientToProduct(prod2.ID, ing1.ID, 0.1, "kg")
	require.NoError(t, err)
	_, err = productStore.AddIngredientToProduct(prod2.ID, ing2.ID, 20, "gr")
	require.NoError(t, err)

	prod3 := &Product{CategoryID: cat2.ID, Name: "Product Gamma", UnitPrice: 20.0, DistributionPrice: 18.0}
	require.NoError(t, productStore.CreateProduct(prod3))
	_, err = productStore.AddIngredientToProduct(prod3.ID, ing2.ID, 50, "gr")
	require.NoError(t, err)

	// Create orders
	// Order 1: Product Alpha (2 units), Product Beta (1 unit) - State: Todo
	order1 := &Order{ClientID: client.ID, State: OrderTodo}
	items1 := []OrderItem{
		{ProductID: prod1.ID, Quantity: 2, Price: "10.0"},
		{ProductID: prod2.ID, Quantity: 1, Price: "15.0"},
	}
	require.NoError(t, orderStore.CreateOrder(order1, items1))

	// Order 2: Product Alpha (3 units) - State: Todo (should aggregate with order1 for Prod Alpha)
	order2 := &Order{ClientID: client.ID, State: OrderTodo}
	items2 := []OrderItem{
		{ProductID: prod1.ID, Quantity: 3, Price: "10.0"},
	}
	require.NoError(t, orderStore.CreateOrder(order2, items2))

	// Order 3: Product Beta (2 units) - State: Done (should NOT be included)
	order3 := &Order{ClientID: client.ID, State: OrderDone}
	items3 := []OrderItem{
		{ProductID: prod2.ID, Quantity: 2, Price: "15.0"},
	}
	require.NoError(t, orderStore.CreateOrder(order3, items3))

	// Order 4: Product Gamma (1 unit) - State: Cancelled (should NOT be included)
	order4 := &Order{ClientID: client.ID, State: OrderCancelled}
	items4 := []OrderItem{
		{ProductID: prod3.ID, Quantity: 1, Price: "20.0"},
	}
	require.NoError(t, orderStore.CreateOrder(order4, items4))

	// Call the function under test
	requirements, err := orderStore.GetPendingProductionRequirements()
	require.NoError(t, err)
	require.NotNil(t, requirements)

	// Assertions
	assert.Len(t, requirements, 2, "Expected 2 products in pending production requirements")

	// Helper to find a requirement by product ID
	findRequirement := func(prodID int64) *ProductionRequirement {
		for _, req := range requirements {
			if req.ProductID == prodID {
				return req
			}
		}
		return nil
	}

	// Verify Product Alpha
	reqAlpha := findRequirement(prod1.ID)
	assert.NotNil(t, reqAlpha, "Product Alpha should be in requirements")
	assert.Equal(t, "Product Alpha", reqAlpha.ProductName)
	assert.Equal(t, 5, reqAlpha.Quantity, "Expected 5 units of Product Alpha (2 from order1 + 3 from order2)")

	// Verify Product Beta
	reqBeta := findRequirement(prod2.ID)
	assert.NotNil(t, reqBeta, "Product Beta should be in requirements")
	assert.Equal(t, "Product Beta", reqBeta.ProductName)
	assert.Equal(t, 1, reqBeta.Quantity, "Expected 1 unit of Product Beta (from order1)")

	// Verify Product Gamma is NOT present
	reqGamma := findRequirement(prod3.ID)
	assert.Nil(t, reqGamma, "Product Gamma should NOT be in requirements (cancelled order)")
}
