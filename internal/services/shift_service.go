package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
)

var (
	ErrShiftAlreadyOpen   = errors.New("ya hay un turno abierto para este usuario")
	ErrNoOpenShift        = errors.New("no hay un turno abierto para cerrar")
	ErrShiftAlreadyClosed = errors.New("este turno ya está cerrado")
)

type ShiftService struct {
	shiftStore store.ShiftStore
	saleStore  store.LocalSaleStore
}

func NewShiftService(shiftStore store.ShiftStore, saleStore store.LocalSaleStore) *ShiftService {
	return &ShiftService{
		shiftStore: shiftStore,
		saleStore:  saleStore,
	}
}

func (s *ShiftService) OpenShift(userID int64, startCash float64, notes string) (*store.Shift, error) {
	// Check if user already has an open shift
	existing, err := s.shiftStore.GetOpenShiftByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("error checking existing shifts: %w", err)
	}
	if existing != nil {
		return nil, ErrShiftAlreadyOpen
	}

	shift := &store.Shift{
		UserID:    userID,
		StartTime: time.Now(),
		StartCash: startCash,
		Status:    "open",
		Notes:     notes,
	}

	if err := s.shiftStore.Create(shift); err != nil {
		return nil, fmt.Errorf("error creating shift: %w", err)
	}

	return shift, nil
}

func (s *ShiftService) CloseShift(userID int64, declaredCash float64, notes string) (*store.Shift, error) {
	shift, err := s.shiftStore.GetOpenShiftByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting open shift: %w", err)
	}
	if shift == nil {
		return nil, ErrNoOpenShift
	}

	now := time.Now()
	shift.EndTime = &now
	shift.EndCashDeclared = &declaredCash
	shift.Status = "closed"
	if notes != "" {
		shift.Notes = shift.Notes + "\n" + notes
	}

	// Calculate expected cash
	// 1. Get total sales during the shift
	sales, err := s.saleStore.GetStats(shift.StartTime, *shift.EndTime)
	if err != nil {
		return nil, fmt.Errorf("error calculating sales stats: %w", err)
	}

	// Assuming "Efectivo" is the payment method name for cash.
	// Ideally we should look up ID or have a constant/config.
	// For now, let's sum up everything but ideally filtering by cash.
	// Since GetStats returns map[string]float64 by method name, we can lookup "Efectivo".
	

cashSales := sales.ByMethod["Efectivo"]
	expected := shift.StartCash + cashSales
	diff := declaredCash - expected

	shift.EndCashExpected = &expected
	shift.Difference = &diff

	if err := s.shiftStore.Update(shift); err != nil {
		return nil, fmt.Errorf("error updating shift: %w", err)
	}

	return shift, nil
}

func (s *ShiftService) GetCurrentShift(userID int64) (*store.Shift, error) {
	return s.shiftStore.GetOpenShiftByUserID(userID)
}

func (s *ShiftService) ListUserShifts(userID int64, page int) ([]*store.Shift, error) {
	limit := 20
	offset := (page - 1) * limit
	return s.shiftStore.ListByUserID(userID, limit, offset)
}
