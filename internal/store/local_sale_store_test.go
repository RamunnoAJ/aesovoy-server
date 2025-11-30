package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalSaleStore_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalSaleStore(db)

	// --- Setup dependencies ---
	pm := &PaymentMethod{Name: "test", Reference: "test"}
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

func TestLocalSaleStore_GetStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalSaleStore(db)
	pmStore := NewPostgresPaymentMethodStore(db)

	// Setup
	pm1 := &PaymentMethod{Name: "Cash", Reference: "cash"}
	require.NoError(t, pmStore.CreatePaymentMethod(pm1))
	pm2 := &PaymentMethod{Name: "Card", Reference: "card"}
	require.NoError(t, pmStore.CreatePaymentMethod(pm2))

	prod := setupProductForStockTest(t, db)

	createSale := func(pmID int64, amount string, date time.Time) {
		sale := &LocalSale{PaymentMethodID: pmID, Subtotal: amount, Total: amount}
		items := []LocalSaleItem{{ProductID: prod.ID, Quantity: 1, UnitPrice: amount, LineSubtotal: amount}}
		tx, _ := db.Begin()
		_ = s.CreateInTx(tx, sale, items)
		tx.Commit()

		// Manually update date
		_, err := db.Exec("UPDATE local_sales SET created_at = $1 WHERE id = $2", date, sale.ID)
		require.NoError(t, err)
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	// Create sales
	createSale(pm1.ID, "100.00", now)
	createSale(pm1.ID, "50.00", now)
	createSale(pm2.ID, "200.00", now)
	createSale(pm1.ID, "500.00", yesterday) // Should be ignored

	// Test
	stats, err := s.GetStats(todayStart, todayEnd)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// Verify
	assert.Equal(t, 3, stats.TotalCount)
	assert.Equal(t, 350.00, stats.TotalAmount)

	assert.Equal(t, 150.00, stats.ByMethod["Cash"])
	assert.Equal(t, 200.00, stats.ByMethod["Card"])
	assert.NotContains(t, stats.ByMethod, "Other")
}

func TestLocalSaleStore_ListByDate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	s := NewPostgresLocalSaleStore(db)
	pmStore := NewPostgresPaymentMethodStore(db)

	// Setup
	pm := &PaymentMethod{Name: "Cash", Reference: "cash"}
	require.NoError(t, pmStore.CreatePaymentMethod(pm))
	prod := setupProductForStockTest(t, db)

	createSale := func(amount string, date time.Time) {
		sale := &LocalSale{PaymentMethodID: pm.ID, Subtotal: amount, Total: amount}
		items := []LocalSaleItem{{ProductID: prod.ID, Quantity: 1, UnitPrice: amount, LineSubtotal: amount}}
		tx, _ := db.Begin()
		_ = s.CreateInTx(tx, sale, items)
		tx.Commit()
		_, err := db.Exec("UPDATE local_sales SET created_at = $1 WHERE id = $2", date, sale.ID)
		require.NoError(t, err)
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	createSale("10.00", now)       // In range
	createSale("20.00", yesterday) // Out of range

	sales, err := s.ListByDate(todayStart, todayEnd)
	require.NoError(t, err)
	assert.Len(t, sales, 1)
	assert.Equal(t, "10.00", sales[0].Total)

	// Test empty range
	sales, err = s.ListByDate(todayEnd, todayEnd.Add(24*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, sales)
}