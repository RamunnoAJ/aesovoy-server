package store

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresPaymentMethodStore(db)

	tests := []struct {
		name    string
		pm      *PaymentMethod
		wantErr bool
	}{
		{
			name: "valid payment method",
			pm: &PaymentMethod{
				Name:      "Test Method 1",
				Reference: "Test Reference 1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.CreatePaymentMethod(tt.pm)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotZero(t, tt.pm.ID)
		})
	}
}

func TestGetPaymentMethodByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresPaymentMethodStore(db)

	pm := &PaymentMethod{
		Name:      "Existing Method",
		Reference: "Existing Reference",
	}
	require.NoError(t, s.CreatePaymentMethod(pm))

	tests := []struct {
		name      string
		id        int64
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "existing payment method",
			id:        pm.ID,
			wantFound: true,
			wantErr:   false,
		},
		{
			name:      "non-existing payment method",
			id:        99999,
			wantFound: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := s.GetPaymentMethodByID(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantFound {
				require.NotNil(t, found)
				assert.Equal(t, pm.Name, found.Name)
				assert.Equal(t, pm.Reference, found.Reference)
			} else {
				assert.Nil(t, found)
			}
		})
	}
}

func TestUpdatePaymentMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresPaymentMethodStore(db)

	pm := &PaymentMethod{
		Name:      "Original Method",
		Reference: "Original Ref",
	}
	require.NoError(t, s.CreatePaymentMethod(pm))

	tests := []struct {
		name    string
		update  *PaymentMethod
		wantErr bool
	}{
		{
			name: "successful update",
			update: &PaymentMethod{
				ID:        pm.ID,
				Name:      "Updated Method",
				Reference: "Updated Ref",
			},
			wantErr: false,
		},
		{
			name: "update non-existent",
			update: &PaymentMethod{
				ID:        99999,
				Name:      "Nobody",
				Reference: "Nowhere",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.UpdatePaymentMethod(tt.update)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Verify update
			updated, err := s.GetPaymentMethodByID(tt.update.ID)
			require.NoError(t, err)
			require.NotNil(t, updated)
			assert.Equal(t, tt.update.Name, updated.Name)
			assert.Equal(t, tt.update.Reference, updated.Reference)
			assert.NotEqual(t, pm.UpdatedAt, updated.UpdatedAt)
		})
	}
}

func TestGetAllPaymentMethods(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresPaymentMethodStore(db)

	totalToCreate := 3
	for i := range totalToCreate {
		pm := &PaymentMethod{
			Name:      fmt.Sprintf("Method %d", i),
			Reference: fmt.Sprintf("Reference %d", i),
		}
		require.NoError(t, s.CreatePaymentMethod(pm))
	}

	tests := []struct {
		name      string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "get all created",
			wantCount: totalToCreate,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pms, err := s.GetAllPaymentMethods()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, pms, tt.wantCount)
		})
	}
}

func TestDeletePaymentMethod(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresPaymentMethodStore(db)

	pm := &PaymentMethod{
		Name:      "To Be Deleted",
		Reference: "Delete Me",
	}
	require.NoError(t, s.CreatePaymentMethod(pm))

	tests := []struct {
		name    string
		id      int64
		wantErr bool
		errType error
	}{
		{
			name:    "delete existing",
			id:      pm.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existing",
			id:      99999,
			wantErr: true,
			errType: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.DeletePaymentMethod(tt.id)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			require.NoError(t, err)
			found, err := s.GetPaymentMethodByID(tt.id)
			require.NoError(t, err)
			assert.Nil(t, found)
		})
	}
}
