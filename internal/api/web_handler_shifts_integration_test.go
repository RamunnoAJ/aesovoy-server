package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

func TestWebHandler_Shifts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	shiftStore := store.NewPostgresShiftStore(db)
	saleStore := store.NewPostgresLocalSaleStore(db)
	// Mock other stores as nil since we focused on shifts
	shiftService := services.NewShiftService(shiftStore, saleStore)
	userStore := store.NewPostgresUserStore(db)

	webHandler := api.NewWebHandler(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, shiftService, nil, logger,
	)

	testUser := &store.User{
		Username: "admin",
		Email:    "admin@test.com",
		Role:     "administrator",
		IsActive: true,
	}
	err := testUser.PasswordHash.Set("password")
	require.NoError(t, err)
	
	err = userStore.CreateUser(testUser)
	require.NoError(t, err)

	t.Run("Open Shift", func(t *testing.T) {
		form := url.Values{}
		form.Add("start_cash", "100.50")
		form.Add("notes", "Opening shift")

		req := httptest.NewRequest("POST", "/shifts/open", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = middleware.SetUser(req, testUser)
		w := httptest.NewRecorder()

		webHandler.HandleOpenShift(w, req)

		resp := w.Result()
		require.Equal(t, http.StatusSeeOther, resp.StatusCode)
		
		// Verify shift created
		shift, err := shiftStore.GetOpenShiftByUserID(testUser.ID)
		require.NoError(t, err)
		require.NotNil(t, shift)
		require.Equal(t, 100.50, shift.StartCash)
		require.Equal(t, "open", shift.Status)
	})

	t.Run("Close Shift", func(t *testing.T) {
		form := url.Values{}
		form.Add("end_cash_declared", "150.00")
		form.Add("notes", "Closing shift")

		req := httptest.NewRequest("POST", "/shifts/close", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req = middleware.SetUser(req, testUser)
		w := httptest.NewRecorder()

		webHandler.HandleCloseShift(w, req)

		resp := w.Result()
		require.Equal(t, http.StatusSeeOther, resp.StatusCode)

		// Verify shift closed
		shift, err := shiftStore.GetOpenShiftByUserID(testUser.ID)
		require.NoError(t, err)
		require.Nil(t, shift) // Should be no open shift

		// Verify list has closed shift
		shifts, err := shiftStore.ListByUserID(testUser.ID, 1, 0)
		require.NoError(t, err)
		require.NotEmpty(t, shifts)
		lastShift := shifts[0]
		require.Equal(t, "closed", lastShift.Status)
		require.Equal(t, 150.00, *lastShift.EndCashDeclared)
	})
}
