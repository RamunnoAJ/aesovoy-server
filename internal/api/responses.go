package api

import (
	"github.com/RamunnoAJ/aesovoy-server/internal/billing"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

// acá tengo todos los tipados para el swagger

type InvoicesResponse struct {
	Data []billing.InvoiceFile `json:"data"`
	Meta InvoicesMeta          `json:"meta"`
}

type InvoicesMeta struct {
	CurrentPage  int    `json:"current_page"`
	TotalPages   int    `json:"total_pages"`
	TotalItems   int    `json:"total_items"`
	ItemsPerPage int    `json:"items_per_page"`
	DateFilter   string `json:"date_filter,omitempty"`
}

type ClientResponse struct {
	Client store.Client `json:"client"`
}

type ClientsResponse struct {
	Clients []store.Client `json:"clients"`
	Meta    utils.Meta     `json:"meta"`
}

type CategoryResponse struct {
	Category store.Category `json:"category"`
}

type CategoriesResponse struct {
	Categories []store.Category `json:"categories"`
}

type ProductResponse struct {
	Product store.Product `json:"product"`
}

type ProductsResponse struct {
	Products []store.Product `json:"products"`
}

type IngredientResponse struct {
	Ingredient store.Ingredient `json:"ingredient"`
}

type IngredientsResponse struct {
	Ingredients []store.Ingredient `json:"ingredients"`
}

type OrderResponse struct {
	Order store.Order `json:"order"`
}

type OrdersResponse struct {
	Orders []store.Order `json:"orders"`
	Meta   utils.Meta    `json:"meta"`
}

type UserResponse struct {
	User store.User `json:"user"`
}

type TokenResponse struct {
	AuthToken string `json:"auth_token"`
}

type ProductIngredientResponse struct {
	ProductIngredient store.ProductIngredient `json:"product_ingredient"`
}

type UpdateOrderStateResponse struct {
	ID    int64            `json:"id"`
	State store.OrderState `json:"state"`
}

type PaymentMethodResponse struct {
	PaymentMethod store.PaymentMethod `json:"payment_method"`
}

type PaymentMethodsResponse struct {
	PaymentMethods []store.PaymentMethod `json:"payment_methods"`
}

type ProviderResponse struct {
	Provider store.Provider `json:"provider"`
}

type LocalStockResponse struct {
	LocalStock store.LocalStock `json:"local_stock"`
}

type LocalStocksResponse struct {
	LocalStock []store.LocalStock `json:"local_stock"`
}

type LocalSaleResponse struct {
	LocalSale store.LocalSale `json:"local_sale"`
}

type LocalSalesResponse struct {
	LocalSales []store.LocalSale `json:"local_sales"`
}

type ProvidersResponse struct {
	Providers []store.Provider `json:"providers"`
	Meta      utils.Meta       `json:"meta"`
}
