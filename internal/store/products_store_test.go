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
	_, err = s.UpdateProductIngredient(product.ID, pi.ID, 20, "kg")
	require.NoError(t, err)

	got, err = s.GetProductByID(product.ID)
	require.NoError(t, err)
	require.Len(t, got.Recipe, 1)
	assert.Equal(t, 20.0, got.Recipe[0].Quantity)
	assert.Equal(t, "kg", got.Recipe[0].Unit)

	// 3. Remove
	err = s.RemoveIngredientFromProduct(product.ID, pi.ID)
	require.NoError(t, err)

	got, err = s.GetProductByID(product.ID)
	require.NoError(t, err)
	assert.Empty(t, got.Recipe)
}
