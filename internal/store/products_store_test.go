package store

import (
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGetProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	categoryStore := NewPostgresCategoryStore(db)
	productStore := NewPostgresProductStore(db)
	ingredientStore := NewPostgresIngredientStore(db)

	// 1. Create dependencies
	category := &Category{Name: "Panaderia"}
	require.NoError(t, categoryStore.CreateCategory(category))

	ingredient1 := &Ingredient{Name: "Harina"}
	require.NoError(t, ingredientStore.CreateIngredient(ingredient1))

	ingredient2 := &Ingredient{Name: "Agua"}
	require.NoError(t, ingredientStore.CreateIngredient(ingredient2))

	// 2. Create Product
	product := &Product{
		CategoryID:        category.ID,
		Name:              "Pan",
		Description:       "Pan casero",
		UnitPrice:         1.5,
		DistributionPrice: 1.0,
	}
	require.NoError(t, productStore.CreateProduct(product))
	assert.NotZero(t, product.ID)

	// 3. Add ingredients to recipe
	_, err := productStore.AddIngredientToProduct(product.ID, ingredient1.ID, 500, "grams")
	require.NoError(t, err)
	_, err = productStore.AddIngredientToProduct(product.ID, ingredient2.ID, 300, "ml")
	require.NoError(t, err)

	// 4. Get Product and verify
	gotProduct, err := productStore.GetProductByID(product.ID)
	require.NoError(t, err)
	require.NotNil(t, gotProduct)

	assert.Equal(t, product.Name, gotProduct.Name)
	assert.Equal(t, category.Name, gotProduct.CategoryName)
	require.Len(t, gotProduct.Recipe, 2)
	assert.Equal(t, "Agua", gotProduct.Recipe[0].Name)
	assert.Equal(t, 300.0, gotProduct.Recipe[0].Quantity)
	assert.Equal(t, "ml", gotProduct.Recipe[0].Unit)
	assert.Equal(t, "Harina", gotProduct.Recipe[1].Name)
	assert.Equal(t, 500.0, gotProduct.Recipe[1].Quantity)
	assert.Equal(t, "grams", gotProduct.Recipe[1].Unit)
}

func TestUpdateProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	categoryStore := NewPostgresCategoryStore(db)
	productStore := NewPostgresProductStore(db)

	category := &Category{Name: "Pasteleria"}
	require.NoError(t, categoryStore.CreateCategory(category))

	product := &Product{
		CategoryID: category.ID,
		Name:       "Torta",
		UnitPrice:  10.0,
	}
	require.NoError(t, productStore.CreateProduct(product))

	product.Name = "Torta de Chocolate"
	product.UnitPrice = 12.5
	require.NoError(t, productStore.UpdateProduct(product))

	updatedProduct, err := productStore.GetProductByID(product.ID)
	require.NoError(t, err)
	assert.Equal(t, "Torta de Chocolate", updatedProduct.Name)
	assert.Equal(t, 12.5, updatedProduct.UnitPrice)
}

func TestDeleteProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)
	category := &Category{Name: "Bebidas"}
	require.NoError(t, categoryStore.CreateCategory(category))

	product := &Product{CategoryID: category.ID, Name: "Jugo"}
	require.NoError(t, productStore.CreateProduct(product))

	require.NoError(t, productStore.DeleteProduct(product.ID))

	deletedProduct, err := productStore.GetProductByID(product.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedProduct)
}

func TestGetAllAndByCategoryProduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)

	cat1 := &Category{Name: "Carnes"}
	require.NoError(t, categoryStore.CreateCategory(cat1))
	cat2 := &Category{Name: "Verduras"}
	require.NoError(t, categoryStore.CreateCategory(cat2))

	p1 := &Product{CategoryID: cat1.ID, Name: "Lomo"}
	require.NoError(t, productStore.CreateProduct(p1))
	p2 := &Product{CategoryID: cat1.ID, Name: "Pollo"}
	require.NoError(t, productStore.CreateProduct(p2))
	p3 := &Product{CategoryID: cat2.ID, Name: "Tomate"}
	require.NoError(t, productStore.CreateProduct(p3))

	allProducts, err := productStore.GetAllProduct()
	require.NoError(t, err)
	assert.Len(t, allProducts, 3)

	cat1Products, err := productStore.GetProductsByCategoryID(cat1.ID)
	require.NoError(t, err)
	assert.Len(t, cat1Products, 2)
}

func TestUpdateAndRemoveProductIngredient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productStore := NewPostgresProductStore(db)
	categoryStore := NewPostgresCategoryStore(db)
	ingredientStore := NewPostgresIngredientStore(db)

	cat := &Category{Name: "Salsas"}
	require.NoError(t, categoryStore.CreateCategory(cat))
	ing := &Ingredient{Name: "Tomate"}
	require.NoError(t, ingredientStore.CreateIngredient(ing))
	prod := &Product{CategoryID: cat.ID, Name: "Salsa de Tomate"}
	require.NoError(t, productStore.CreateProduct(prod))

	pi, err := productStore.AddIngredientToProduct(prod.ID, ing.ID, 5, "unidades")
	require.NoError(t, err)

	// Update
	_, err = productStore.UpdateProductIngredient(prod.ID, pi.ID, 7, "unidades")
	require.NoError(t, err)

	updatedProd, err := productStore.GetProductByID(prod.ID)
	require.NoError(t, err)
	require.Len(t, updatedProd.Recipe, 1)
	assert.Equal(t, 7.0, updatedProd.Recipe[0].Quantity)

	// Remove
	require.NoError(t, productStore.RemoveIngredientFromProduct(prod.ID, pi.ID))

	finalProd, err := productStore.GetProductByID(prod.ID)
	require.NoError(t, err)
	assert.Len(t, finalProd.Recipe, 0)
}
