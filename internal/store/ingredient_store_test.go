package store

import (
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateIngredient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresIngredientStore(db)

	tests := []struct {
		name       string
		ingredient *Ingredient
		wantErr    bool
	}{
		{
			name: "valid ingredient",
			ingredient: &Ingredient{
				Name: "Harina",
			},
			wantErr: false,
		},
		{
			name: "existing ingredient",
			ingredient: &Ingredient{
				Name: "Harina",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateIngredient(tt.ingredient)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotZero(t, tt.ingredient.ID)
		})
	}
}

func TestGetIngredientByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresIngredientStore(db)

	ingredient := &Ingredient{
		Name: "Azucar",
	}
	err := store.CreateIngredient(ingredient)
	require.NoError(t, err)

	tests := []struct {
		name         string
		ingredientID int64
		wantErr      bool
		wantRes      bool
	}{
		{
			name:         "existing ingredient",
			ingredientID: ingredient.ID,
			wantErr:      false,
			wantRes:      true,
		},
		{
			name:         "non existing ingredient",
			ingredientID: 0,
			wantErr:      false,
			wantRes:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetIngredientByID(tt.ingredientID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantRes {
				assert.Equal(t, got.Name, ingredient.Name)
				return
			}

			assert.Nil(t, got)
		})
	}
}

func TestUpdateIngredient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresIngredientStore(db)

	ingredient := &Ingredient{
		Name: "Levadura",
	}
	err := store.CreateIngredient(ingredient)
	require.NoError(t, err)

	tests := []struct {
		name       string
		updateFunc func(*Ingredient)
		wantErr    bool
	}{
		{
			name: "update name",
			updateFunc: func(i *Ingredient) {
				i.Name = "Levadura Fresca"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.updateFunc(ingredient)
			err := store.UpdateIngredient(ingredient)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			updatedIngredient, err := store.GetIngredientByID(ingredient.ID)
			require.NoError(t, err)
			assert.Equal(t, ingredient.Name, updatedIngredient.Name)
		})
	}
}

func TestGetAllIngredients(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresIngredientStore(db)

	// Add some ingredients
	_, err := db.Exec(`INSERT INTO ingredients (name) VALUES ('Harina'), ('Azucar'), ('Sal')`)
	require.NoError(t, err)

	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "get all",
			wantCount: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ingredients, err := store.GetAllIngredients()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, ingredients, tt.wantCount)
		})
	}
}

func TestDeleteIngredient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresIngredientStore(db)

	ingredient := &Ingredient{
		Name: "Aceite",
	}
	err := store.CreateIngredient(ingredient)
	require.NoError(t, err)

	tests := []struct {
		name         string
		ingredientID int64
		wantErr      bool
	}{
		{
			name:         "delete existing",
			ingredientID: ingredient.ID,
			wantErr:      false,
		},
		{
			name:         "delete non-existing",
			ingredientID: 999,
			wantErr:      true, // Expects sql.ErrNoRows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.DeleteIngredient(tt.ingredientID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			deletedIngredient, err := store.GetIngredientByID(tt.ingredientID)
			require.NoError(t, err)
			assert.Nil(t, deletedIngredient)
		})
	}
}
