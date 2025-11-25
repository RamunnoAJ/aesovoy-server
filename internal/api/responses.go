package api

import (
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

// acá tengo todos los tipados para el swagger

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
	ID    int64           `json:"id"`
	State store.OrderState `json:"state"`
}

// PaymentMethodResponse defines the response structure for a single payment method.
type PaymentMethodResponse struct {
	PaymentMethod store.PaymentMethod `json:"payment_method"`
}

// PaymentMethodsResponse defines the response structure for a list of payment methods.
type PaymentMethodsResponse struct {
	PaymentMethods []store.PaymentMethod `json:"payment_methods"`
}

// ProviderResponse defines the response structure for a single provider.
type ProviderResponse struct {
	Provider store.Provider `json:"provider"`
}

type ProvidersResponse struct {
	Providers []store.Provider `json:"providers"`
	Meta      utils.Meta       `json:"meta"`
}
