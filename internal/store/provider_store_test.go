package store

import (
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProvider(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresProviderStore(db)

	tests := []struct {
		name     string
		provider *Provider
		wantErr  bool
	}{
		{
			name: "valid provider",
			provider: &Provider{
				Name:      "Proveedor Valido",
				Reference: "ref-1",
				CUIT:      "cuit-1",
				Email:     "email-1",
			},
			wantErr: false,
		},
		{
			name: "duplicate provider name",
			provider: &Provider{
				Name:      "Proveedor Valido",
				Reference: "ref-2",
				CUIT:      "cuit-2",
				Email:     "email-2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateProvider(tt.provider)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, tt.provider.ID)
		})
	}
}

func TestGetProviderByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresProviderStore(db)

	provider := &Provider{
		Name:      "Proveedor Existente",
		Reference: "ref-get",
		CUIT:      "cuit-get",
		Email:     "email-get",
	}
	require.NoError(t, store.CreateProvider(provider))

	tests := []struct {
		name       string
		providerID int64
		wantFound  bool
		wantErr    bool
	}{
		{
			name:       "existing provider",
			providerID: provider.ID,
			wantFound:  true,
			wantErr:    false,
		},
		{
			name:       "non-existing provider",
			providerID: 999,
			wantFound:  false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetProviderByID(tt.providerID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantFound {
				require.NotNil(t, got)
				assert.Equal(t, provider.Name, got.Name)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestUpdateProvider(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresProviderStore(db)

	provider := &Provider{
		Name:      "Original",
		Reference: "ref-update",
		CUIT:      "cuit-update",
		Email:     "email-update",
	}
	require.NoError(t, store.CreateProvider(provider))

	tests := []struct {
		name       string
		updateFunc func(*Provider)
		wantErr    bool
	}{
		{
			name: "update name and email",
			updateFunc: func(p *Provider) {
				p.Name = "Actualizado"
				p.Email = "actualizado@test.com"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.updateFunc(provider)
			err := store.UpdateProvider(provider)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			updatedProvider, err := store.GetProviderByID(provider.ID)
			require.NoError(t, err)
			assert.Equal(t, provider.Name, updatedProvider.Name)
			assert.Equal(t, provider.Email, updatedProvider.Email)
		})
	}
}

func TestGetAllProviders(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresProviderStore(db)

	for i := 0; i < 2; i++ {
		p := &Provider{
			Name:      fmt.Sprintf("Proveedor %d", i),
			Reference: fmt.Sprintf("ref-getall-%d", i),
			CUIT:      fmt.Sprintf("cuit-getall-%d", i),
			Email:     fmt.Sprintf("email-getall-%d", i),
		}
		require.NoError(t, store.CreateProvider(p))
	}

	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "get all",
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers, err := store.GetAllProviders()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, providers, tt.wantCount)
		})
	}
}

func TestSearchProvidersFTS(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresProviderStore(db)

	p1 := &Provider{
		Name:      "Lacteos S.A.",
		Address:   "Calle Falsa 123",
		Reference: "ref-fts-1",
		CUIT:      "cuit-fts-1",
		Email:     "email-fts-1",
	}
	require.NoError(t, store.CreateProvider(p1))
	p2 := &Provider{
		Name:      "Carnes S.R.L.",
		Phone:     "987654321",
		Reference: "ref-fts-2",
		CUIT:      "cuit-fts-2",
		Email:     "email-fts-2",
	}
	require.NoError(t, store.CreateProvider(p2))

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantErr   bool
	}{
		{name: "search by name", query: "Lacteos", wantCount: 1, wantErr: false},
		{name: "search by address", query: "falsa", wantCount: 1, wantErr: false},
		{name: "search by phone", query: "987654321", wantCount: 1, wantErr: false},
		{name: "no results", query: "inexistente", wantCount: 0, wantErr: false},
		{name: "empty query returns all", query: "", wantCount: 2, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.SearchProvidersFTS(tt.query, 10, 0)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}
