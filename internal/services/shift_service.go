package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
)

var (
	ErrShiftAlreadyOpen   = errors.New("ya hay un turno abierto para este usuario")
	ErrNoOpenShift        = errors.New("no hay un turno abierto para cerrar")
	ErrShiftAlreadyClosed = errors.New("este turno ya está cerrado")
	ErrInvalidAmount      = errors.New("el monto debe ser mayor a 0")
)

type ShiftService struct {
	shiftStore    store.ShiftStore
	saleStore     store.LocalSaleStore
	movementStore store.CashMovementStore
}

func NewShiftService(
	shiftStore store.ShiftStore,
	saleStore store.LocalSaleStore,
	movementStore store.CashMovementStore,
) *ShiftService {
	return &ShiftService{
		shiftStore:    shiftStore,
		saleStore:     saleStore,
		movementStore: movementStore,
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
	
	// 2. Get movements total
	totalIn, totalOut, err := s.movementStore.GetTotalByShiftID(shift.ID)
	if err != nil {
		return nil, fmt.Errorf("error calculating movement stats: %w", err)
	}

	// Calculate cash sales by looking for "Efectivo" (case-insensitive)
	var cashSales float64
	for method, amount := range sales.ByMethod {
		if strings.EqualFold(method, "Efectivo") {
			cashSales += amount
		}
	}
	
	expected := shift.StartCash + cashSales + totalIn - totalOut
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

func (s *ShiftService) RegisterMovement(userID int64, amount float64, typeStr string, reason string) (*store.CashMovement, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	shift, err := s.shiftStore.GetOpenShiftByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("error getting open shift: %w", err)
	}
	if shift == nil {
		return nil, ErrNoOpenShift
	}

	movement := &store.CashMovement{
		ShiftID: shift.ID,
		Amount:  amount,
		Type:    store.CashMovementType(typeStr),
		Reason:  reason,
	}

	if err := s.movementStore.Create(movement); err != nil {
		return nil, fmt.Errorf("error creating movement: %w", err)
	}

	return movement, nil
}

func (s *ShiftService) ListMovements(shiftID int64) ([]*store.CashMovement, error) {
	return s.movementStore.ListByShiftID(shiftID)
}