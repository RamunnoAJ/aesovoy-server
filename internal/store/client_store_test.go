package store

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresClientStore(db)

	tests := []struct {
		name    string
		client  *Client
		wantErr bool
	}{
		{
			name: "valid client",
			client: &Client{
				Name:      "Cliente Valido",
				Type:      ClientTypeIndividual,
				Reference: "ref-1",
				CUIT:      "cuit-1",
			},
			wantErr: false,
		},
		{
			name: "duplicate client name",
			client: &Client{
				Name:      "Cliente Valido",
				Type:      ClientTypeIndividual,
				Reference: "ref-2",
				CUIT:      "cuit-2",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.CreateClient(tt.client)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, tt.client.ID)
		})
	}
}

func TestGetClientByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresClientStore(db)

	client := &Client{
		Name:      "Cliente Existente",
		Type:      ClientTypeIndividual,
		Reference: "ref-get",
		CUIT:      "cuit-get",
	}
	require.NoError(t, store.CreateClient(client))

	tests := []struct {
		name      string
		clientID  int64
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "existing client",
			clientID:  client.ID,
			wantFound: true,
			wantErr:   false,
		},
		{
			name:      "non-existing client",
			clientID:  999,
			wantFound: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetClientByID(tt.clientID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantFound {
				require.NotNil(t, got)
				assert.Equal(t, client.Name, got.Name)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestUpdateClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresClientStore(db)

	client := &Client{
		Name:      "Original",
		Type:      ClientTypeIndividual,
		Reference: "ref-update",
		CUIT:      "cuit-update",
	}
	require.NoError(t, store.CreateClient(client))

	tests := []struct {
		name       string
		updateFunc func(*Client)
		wantErr    bool
	}{
		{
			name: "update name and email",
			updateFunc: func(c *Client) {
				c.Name = "Actualizado"
				c.Email = "actualizado@test.com"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.updateFunc(client)
			err := store.UpdateClient(client)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			updatedClient, err := store.GetClientByID(client.ID)
			require.NoError(t, err)
			require.NotNil(t, updatedClient)
			assert.Equal(t, client.Name, updatedClient.Name)
			assert.Equal(t, client.Email, updatedClient.Email)
		})
	}
}

func TestGetAllClients(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresClientStore(db)

	for i := 0; i < 2; i++ {
		c := &Client{
			Name:      fmt.Sprintf("Cliente %d", i),
			Type:      ClientTypeIndividual,
			Reference: fmt.Sprintf("ref-getall-%d", i),
			CUIT:      fmt.Sprintf("cuit-getall-%d", i),
			Email:     fmt.Sprintf("email-getall-%d", i),
		}
		require.NoError(t, store.CreateClient(c))
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
			clients, err := store.GetAllClients()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, clients, tt.wantCount)
		})
	}
}

func TestDeleteClient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := NewPostgresClientStore(db)

	client := &Client{
		Name:      "Cliente a Borrar",
		Type:      ClientTypeIndividual,
		Reference: "ref-delete",
		CUIT:      "cuit-delete",
	}
	require.NoError(t, store.CreateClient(client))

	t.Run("delete existing client", func(t *testing.T) {
		err := store.DeleteClient(client.ID)
		require.NoError(t, err)

		// Verify client is deleted
		deletedClient, err := store.GetClientByID(client.ID)
		require.NoError(t, err)
		assert.Nil(t, deletedClient)
	})

	t.Run("delete non-existent client", func(t *testing.T) {
		err := store.DeleteClient(999) // Non-existent ID
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}
