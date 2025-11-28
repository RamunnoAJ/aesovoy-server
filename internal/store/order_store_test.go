package store

import (
	"testing"

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
