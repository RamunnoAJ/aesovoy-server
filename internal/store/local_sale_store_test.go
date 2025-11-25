package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSaleStore_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalSaleStore(db)

	// --- Setup dependencies ---
	pm := &PaymentMethod{Owner: "test", Reference: "test"}
	require.NoError(t, NewPostgresPaymentMethodStore(db).CreatePaymentMethod(pm))
	prod := setupProductForStockTest(t, db)

	// --- Test Data ---
	sale := &LocalSale{
		PaymentMethodID: pm.ID,
		Subtotal:        "200.00",
		Total:           "200.00",
	}
	items := []LocalSaleItem{
		{ProductID: prod.ID, Quantity: 2, UnitPrice: "100.00", LineSubtotal: "200.00"},
	}

	// --- Create in Transaction ---
	tx, err := db.Begin()
	require.NoError(t, err)
	err = s.CreateInTx(tx, sale, items)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	assert.NotZero(t, sale.ID)
	require.Len(t, sale.Items, 1)
	assert.NotZero(t, sale.Items[0].ID)

	// --- Get and Verify ---
	t.Run("get created sale", func(t *testing.T) {
		gotSale, err := s.GetByID(sale.ID)
		require.NoError(t, err)
		require.NotNil(t, gotSale)

		assert.Equal(t, sale.ID, gotSale.ID)
		assert.Equal(t, sale.PaymentMethodID, gotSale.PaymentMethodID)
		assert.Equal(t, sale.Total, gotSale.Total)
		require.Len(t, gotSale.Items, 1)
		assert.Equal(t, sale.Items[0].ID, gotSale.Items[0].ID)
		assert.Equal(t, sale.Items[0].Quantity, gotSale.Items[0].Quantity)
	})

	// --- List and Verify ---
	t.Run("list sales", func(t *testing.T) {
		// Create another sale
		sale2 := &LocalSale{PaymentMethodID: pm.ID, Subtotal: "50", Total: "50"}
		items2 := []LocalSaleItem{{ProductID: prod.ID, Quantity: 1, UnitPrice: "50", LineSubtotal: "50"}}
		tx2, err := db.Begin()
		require.NoError(t, err)
		require.NoError(t, s.CreateInTx(tx2, sale2, items2))
		require.NoError(t, tx2.Commit())

		allSales, err := s.ListAll()
		require.NoError(t, err)
		assert.Len(t, allSales, 2)
	})
}
