package store

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCategory(t *testing.T, db *sql.DB) *Category {
	t.Helper()
	categoryStore := NewPostgresCategoryStore(db)
	name := fmt.Sprintf("Test Category %d", time.Now().UnixNano())
	category := &Category{Name: name}
	require.NoError(t, categoryStore.CreateCategory(category))
	return category
}

func TestProductStore_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	tests := []struct {
		name    string
		product *Product
		wantErr bool
	}{
		{
			name: "create valid product",
			product: &Product{
				CategoryID:        category.ID,
				Name:              "Pan",
				Description:       "Pan casero",
				UnitPrice:         1.5,
				DistributionPrice: 1.0,
			},
			wantErr: false,
		},
		{
			name:    "create with invalid category",
			product: &Product{CategoryID: 999, Name: "Invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreateProduct(tt.product)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotZero(t, tt.product.ID)
			}
		})
	}
}

func TestProductStore_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Test Product",
		UnitPrice:  10,
	}
	require.NoError(t, s.CreateProduct(product))

	// Get and verify
	got, err := s.GetProductByID(product.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, product.Name, got.Name)
	assert.Equal(t, category.Name, got.CategoryName)

	// Get non-existent
	got, err = s.GetProductByID(9999)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestProductStore_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	product := &Product{
		CategoryID: category.ID,
		Name:       "Original Name",
		UnitPrice:  10.0,
	}
	require.NoError(t, s.CreateProduct(product))

	// Update fields
	product.Name = "Updated Name"
	product.Description = "Updated Desc"
	product.UnitPrice = 15.5
	err := s.UpdateProduct(product)
	require.NoError(t, err)

	// Get and verify
	updated, err := s.GetProductByID(product.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "Updated Desc", updated.Description)
	assert.Equal(t, 15.5, updated.UnitPrice)
}

func TestProductStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	product := &Product{CategoryID: category.ID, Name: "To Delete", UnitPrice: 1}
	require.NoError(t, s.CreateProduct(product))

	// Delete
	err := s.DeleteProduct(product.ID)
	require.NoError(t, err)

	// Verify gone
	got, err := s.GetProductByID(product.ID)
	require.NoError(t, err)
	assert.Nil(t, got)

	// Delete non-existent should error
	err = s.DeleteProduct(9999)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestProductStore_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	cat1 := setupCategory(t, db)
	cat2 := setupCategory(t, db)

	// Create products
	p1 := &Product{CategoryID: cat1.ID, Name: "A Product"}
	require.NoError(t, s.CreateProduct(p1))
	p2 := &Product{CategoryID: cat1.ID, Name: "B Product"}
	require.NoError(t, s.CreateProduct(p2))
	p3 := &Product{CategoryID: cat2.ID, Name: "C Product"}
	require.NoError(t, s.CreateProduct(p3))

	// Test GetAll
	all, err := s.GetAllProduct()
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// Test GetByCategoryID
	cat1Products, err := s.GetProductsByCategoryID(cat1.ID)
	require.NoError(t, err)
	assert.Len(t, cat1Products, 2)
}

func TestProductStore_Ingredients(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	ingredientStore := NewPostgresIngredientStore(db)
	ing1 := &Ingredient{Name: "Ing 1"}
	require.NoError(t, ingredientStore.CreateIngredient(ing1))
	ing2 := &Ingredient{Name: "Ing 2"}
	require.NoError(t, ingredientStore.CreateIngredient(ing2))

	product := &Product{CategoryID: category.ID, Name: "Prod With Ingredients"}
	require.NoError(t, s.CreateProduct(product))

	// 1. Add
	pi, err := s.AddIngredientToProduct(product.ID, ing1.ID, 10, "g")
	require.NoError(t, err)

	got, err := s.GetProductByID(product.ID)
	require.NoError(t, err)
	require.Len(t, got.Recipe, 1)
	assert.Equal(t, "Ing 1", got.Recipe[0].Name)

	// 2. Update
	_, err = s.UpdateProductIngredient(product.ID, pi.ID, 20, "g")
	require.NoError(t, err)

	got, err = s.GetProductByID(product.ID)
	require.NoError(t, err)
	require.Len(t, got.Recipe, 1)
	assert.Equal(t, 20.0, got.Recipe[0].Quantity)
	assert.Equal(t, "g", got.Recipe[0].Unit)

	// 3. Remove
	err = s.RemoveIngredientFromProduct(product.ID, pi.ID)
	require.NoError(t, err)

	got, err = s.GetProductByID(product.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Recipe)
}

func TestProductStore_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	// Create products for search
	p1 := &Product{CategoryID: category.ID, Name: "Torta de Chocolate", Description: "Deliciosa torta de chocolate con dulce de leche"}
	require.NoError(t, s.CreateProduct(p1))
	p2 := &Product{CategoryID: category.ID, Name: "Torta de Vainilla", Description: "Torta clásica de vainilla"}
	require.NoError(t, s.CreateProduct(p2))
	p3 := &Product{CategoryID: category.ID, Name: "Alfajor de Maicena", Description: "Alfajor tradicional"}
	require.NoError(t, s.CreateProduct(p3))

	tests := []struct {
		name     string
		query    string
		limit    int
		offset   int
		wantLen  int
		wantName string // Check first result name if expected > 0
	}{
		{
			name:     "search by name match",
			query:    "Chocolate",
			limit:    10,
			offset:   0,
			wantLen:  1,
			wantName: "Torta de Chocolate",
		},
		{
			name:     "search by description match",
			query:    "Dulce de leche",
			limit:    10,
			offset:   0,
			wantLen:  1,
			wantName: "Torta de Chocolate",
		},
		{
			name:     "search partial match",
			query:    "Vaini",
			limit:    10,
			offset:   0,
			wantLen:  1,
			wantName: "Torta de Vainilla",
		},
		{
			name:    "search multiple match",
			query:   "Torta",
			limit:   10,
			offset:  0,
			wantLen: 2,
			// Order is by rank then name. "Torta" appears in both name (A) and description (B) for both?
			// Actually 'Torta' is in Name (weight A) for both.
			// Sorting by name should put Chocolate before Vainilla.
			wantName: "Torta de Chocolate",
		},
		{
			name:    "search no match",
			query:   "Pizza",
			limit:   10,
			offset:  0,
			wantLen: 0,
		},
		{
			name:    "empty query returns all (paginated)",
			query:   "",
			limit:   2,
			offset:  0,
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.SearchProductsFTS(tt.query, tt.limit, tt.offset)
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 && tt.wantName != "" {
				assert.Equal(t, tt.wantName, got[0].Name)
			}
		})
	}
}

func TestProductStore_GetTopSellingProducts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ps := NewPostgresProductStore(db)
	cat := setupCategory(t, db)

	// Create products
	p1 := &Product{CategoryID: cat.ID, Name: "Top 1"}
	require.NoError(t, ps.CreateProduct(p1))
	p2 := &Product{CategoryID: cat.ID, Name: "Top 2"}
	require.NoError(t, ps.CreateProduct(p2))
	p3 := &Product{CategoryID: cat.ID, Name: "Not Top"}
	require.NoError(t, ps.CreateProduct(p3))

	// Setup other stores to create sales
	pmStore := NewPostgresPaymentMethodStore(db)
	pm := &PaymentMethod{Name: "Cash", Reference: "cash"}
	require.NoError(t, pmStore.CreatePaymentMethod(pm))
	lsStore := NewPostgresLocalSaleStore(db)

	orderStore := NewPostgresOrderStore(db)
	clientStore := NewPostgresClientStore(db)
	client := &Client{Name: "C", Type: ClientTypeIndividual, Reference: "r", CUIT: "c"}
	require.NoError(t, clientStore.CreateClient(client))

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	// Create Local Sale: p1 (5 units), p2 (2 units)
	tx, _ := db.Begin()
	ls := &LocalSale{PaymentMethodID: pm.ID, Subtotal: "0", Total: "0"}
	items := []LocalSaleItem{
		{ProductID: p1.ID, Quantity: 5, UnitPrice: "1", LineSubtotal: "5"},
		{ProductID: p2.ID, Quantity: 2, UnitPrice: "1", LineSubtotal: "2"},
	}
	_ = lsStore.CreateInTx(tx, ls, items)
	tx.Commit()
	// Set date manually to ensure it falls in range if needed (Postgres uses NOW() default)
	// But default is NOW(), so it's fine.

	// Create Order: p1 (3 units), p2 (1 unit), p3 (0 unit)
	o := &Order{ClientID: client.ID, State: OrderDone}
	oItems := []OrderItem{
		{ProductID: p1.ID, Quantity: 3, Price: "1"}, // Total p1 = 8
		{ProductID: p2.ID, Quantity: 1, Price: "1"}, // Total p2 = 3
	}
	require.NoError(t, orderStore.CreateOrder(o, oItems))

	// Create Order (Cancelled): p2 (10 units) - Should be ignored
	oCan := &Order{ClientID: client.ID, State: OrderCancelled}
	oCanItems := []OrderItem{{ProductID: p2.ID, Quantity: 10, Price: "1"}}
	require.NoError(t, orderStore.CreateOrder(oCan, oCanItems))

	// Test
	top, err := ps.GetTopSellingProducts(todayStart, todayEnd)
	require.NoError(t, err)
	require.Len(t, top, 2) // We only sold p1 and p2 effectively. p3 sold 0. The query joins on sales tables.

	assert.Equal(t, p1.ID, top[0].ID)
	assert.Equal(t, 8.0, top[0].Quantity) // 5 local + 3 order

	assert.Equal(t, p2.ID, top[1].ID)

	assert.Equal(t, 3.0, top[1].Quantity) // 2 local + 1 order

	// Test Local

	topLocal, err := ps.GetTopSellingProductsLocal(todayStart, todayEnd)

	require.NoError(t, err)

	require.Len(t, topLocal, 2)

	assert.Equal(t, p1.ID, topLocal[0].ID)

	assert.Equal(t, 5.0, topLocal[0].Quantity)

	assert.Equal(t, p2.ID, topLocal[1].ID)

	assert.Equal(t, 2.0, topLocal[1].Quantity)

	// Test Distribution

	topDistrib, err := ps.GetTopSellingProductsDistribution(todayStart, todayEnd)

	require.NoError(t, err)

	require.Len(t, topDistrib, 2)

	assert.Equal(t, p1.ID, topDistrib[0].ID)

	assert.Equal(t, 3.0, topDistrib[0].Quantity)

	assert.Equal(t, p2.ID, topDistrib[1].ID)

	assert.Equal(t, 1.0, topDistrib[1].Quantity)

}

func TestProductStore_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewPostgresProductStore(db)
	category := setupCategory(t, db)

	p := &Product{CategoryID: category.ID, Name: "UniqueProduct", UnitPrice: 10}
	require.NoError(t, s.CreateProduct(p))

	// Verify exists
	got, err := s.GetProductByID(p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	// Delete
	require.NoError(t, s.DeleteProduct(p.ID))

	// Verify not found
	gotDeleted, err := s.GetProductByID(p.ID)
	require.NoError(t, err)
	assert.Nil(t, gotDeleted)

	// Create duplicate name should succeed
	p2 := &Product{CategoryID: category.ID, Name: "UniqueProduct", UnitPrice: 20}
	require.NoError(t, s.CreateProduct(p2))
	
	// Verify new product
	gotNew, err := s.GetProductByID(p2.ID)
	require.NoError(t, err)
	require.NotNil(t, gotNew)
	assert.Equal(t, 20.0, gotNew.UnitPrice)
}

