package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
)

var (
	ErrPaymentMethodNotFound = errors.New("payment method not found")
)

type CreateLocalSaleItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type CreateLocalSaleRequest struct {
	PaymentMethodID int64                 `json:"payment_method_id"`
	Items           []CreateLocalSaleItem `json:"items"`
}

type LocalSaleService struct {
	db                 *sql.DB
	saleStore          store.LocalSaleStore
	stockStore         store.LocalStockStore
	paymentMethodStore store.PaymentMethodStore
	productStore       store.ProductStore
}

func NewLocalSaleService(
	db *sql.DB,
	saleStore store.LocalSaleStore,
	stockStore store.LocalStockStore,
	paymentMethodStore store.PaymentMethodStore,
	productStore store.ProductStore,
) *LocalSaleService {
	return &LocalSaleService{
		db:                 db,
		saleStore:          saleStore,
		stockStore:         stockStore,
		paymentMethodStore: paymentMethodStore,
		productStore:       productStore,
	}
}

func (s *LocalSaleService) CreateLocalSale(req CreateLocalSaleRequest) (*store.LocalSale, error) {
	// --- 1. Validations and data fetching (outside transaction) ---
	if len(req.Items) == 0 {
		return nil, errors.New("sale must have at least one item")
	}

	paymentMethod, err := s.paymentMethodStore.GetPaymentMethodByID(req.PaymentMethodID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify payment method: %w", err)
	}
	// Explicitly check if the payment method was found
	if paymentMethod == nil {
		return nil, ErrPaymentMethodNotFound
	}

	var saleItems []store.LocalSaleItem
	var productIDs []int64
	for _, item := range req.Items {
		productIDs = append(productIDs, item.ProductID)
	}

	products, err := s.productStore.GetProductsByIDs(productIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	var subtotal float64 = 0.0

	for _, itemReq := range req.Items {
		product, ok := products[itemReq.ProductID]
		if !ok {
			return nil, fmt.Errorf("%w: id %d", ErrProductNotFound, itemReq.ProductID)
		}

		stock, err := s.stockStore.GetByProductID(itemReq.ProductID)
		if err != nil {
			return nil, fmt.Errorf("failed to check stock for product %d: %w", itemReq.ProductID, err)
		}
		if stock == nil || stock.Quantity < itemReq.Quantity {
			return nil, fmt.Errorf("%w: product id %d (available: %d, required: %d)", ErrInsufficientStock, itemReq.ProductID, stock.Quantity, itemReq.Quantity)
		}

		lineSubtotal := product.UnitPrice * float64(itemReq.Quantity)
		subtotal += lineSubtotal

		saleItems = append(saleItems, store.LocalSaleItem{
			ProductID:    itemReq.ProductID,
			Quantity:     itemReq.Quantity,
			UnitPrice:    strconv.FormatFloat(product.UnitPrice, 'f', 2, 64),
			LineSubtotal: strconv.FormatFloat(lineSubtotal, 'f', 2, 64),
		})
	}

	// --- 2. Transactional block ---
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	sale := &store.LocalSale{
		PaymentMethodID: req.PaymentMethodID,
		Subtotal:        strconv.FormatFloat(subtotal, 'f', 2, 64),
		Total:           strconv.FormatFloat(subtotal, 'f', 2, 64),
	}

	if err := s.saleStore.CreateInTx(tx, sale, saleItems); err != nil {
		return nil, fmt.Errorf("failed to create sale in transaction: %w", err)
	}

	for _, item := range saleItems {
		if _, err := s.stockStore.AdjustQuantityTx(tx, item.ProductID, -item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to adjust stock for product %d: %w", item.ProductID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return sale, nil
}

func (s *LocalSaleService) GetSale(id int64) (*store.LocalSale, error) {
	return s.saleStore.GetByID(id)
}

func (s *LocalSaleService) ListSales() ([]*store.LocalSale, error) {
	return s.saleStore.ListAll()
}

