package services

import (
	"errors"
	"fmt"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
)

var (
	ErrStockRecordExists      = errors.New("ya existe un stock asignado para este producto")
	ErrStockRecordNotFound    = errors.New("no se encontró stock para este producto")
	ErrProductNotFound        = errors.New("producto no encontrado")
	ErrInsufficientStock      = errors.New("no hay stock suficiente")
	ErrInitialQuantityInvalid = errors.New("cantidad inicial debe ser 0 o mayor")
)

type LocalStockService struct {
	stockStore   store.LocalStockStore
	productStore store.ProductStore
}

func NewLocalStockService(stockStore store.LocalStockStore, productStore store.ProductStore) *LocalStockService {
	return &LocalStockService{
		stockStore:   stockStore,
		productStore: productStore,
	}
}

func (s *LocalStockService) GetStock(productID int64) (*store.LocalStock, error) {
	return s.stockStore.GetByProductID(productID)
}

func (s *LocalStockService) ListStock() ([]*store.ProductStock, error) {
	return s.stockStore.ListStockWithProductDetails()
}

func (s *LocalStockService) CreateInitialStock(productID int64, initialQuantity int) (*store.LocalStock, error) {
	if initialQuantity < 0 {
		return nil, ErrInitialQuantityInvalid
	}

	product, err := s.productStore.GetProductByID(productID)
	if err != nil {
		return nil, fmt.Errorf("error checking product existence: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	existing, err := s.stockStore.GetByProductID(productID)
	if err != nil {
		return nil, fmt.Errorf("error checking existing stock: %w", err)
	}
	if existing != nil {
		return nil, ErrStockRecordExists
	}

	return s.stockStore.Create(productID, initialQuantity)
}

func (s *LocalStockService) AdjustStock(productID int64, delta int) (*store.LocalStock, error) {
	stock, err := s.stockStore.GetByProductID(productID)
	if err != nil {
		return nil, fmt.Errorf("error getting current stock: %w", err)
	}
	if stock == nil {
		return nil, ErrStockRecordNotFound
	}

	if stock.Quantity+delta < 0 {
		return nil, ErrInsufficientStock
	}

	return s.stockStore.AdjustQuantity(productID, delta)
}

// HasSufficientStock is for future integration with sales flow.
func (s *LocalStockService) HasSufficientStock(productID int64, quantityNeeded int) (bool, error) {
	stock, err := s.stockStore.GetByProductID(productID)
	if err != nil {
		return false, err
	}
	if stock == nil {
		return false, nil // No stock record means 0 stock
	}
	return stock.Quantity >= quantityNeeded, nil
}

func (s *LocalStockService) GetLowStockAlerts(threshold int) ([]*store.ProductStock, error) {
	return s.stockStore.GetLowStockAlerts(threshold)
}
