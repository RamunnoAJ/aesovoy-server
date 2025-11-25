package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresCategoryStore(db)

	tests := []struct {
		name     string
		category *Category
		wantErr  bool
	}{
		{
			name: "valid category",
			category: &Category{
				Name:        "Harinas",
				Description: "Panes, pizzas",
			},
			wantErr: false,
		},
		{
			name: "existing category",
			category: &Category{
				Name:        "Harinas",
				Description: "Panes, pizzas",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateCategory(tt.category)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetCategoryByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresCategoryStore(db)

	category := &Category{
		Name:        "Harinas",
		Description: "Panes, pizzas",
	}
	err := store.CreateCategory(category)
	require.NoError(t, err)

	tests := []struct {
		name     string
		category *Category
		wantErr  bool
		wantRes  bool
	}{
		{
			name:     "existing category",
			category: category,
			wantErr:  false,
			wantRes:  true,
		},
		{
			name: "non existing category",
			category: &Category{
				ID:          0,
				Name:        "",
				Description: "",
			},
			wantErr: false,
			wantRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetCategoryByID(int64(tt.category.ID))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.wantRes {
				assert.Equal(t, got.Name, tt.category.Name)
				assert.Equal(t, got.Description, tt.category.Description)
				return
			}

			assert.Nil(t, got)
		})
	}
}
