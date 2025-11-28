package api

import (
	"log/slog"

	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
)

type WebHandler struct {
	userStore          store.UserStore
	tokenStore         store.TokenStore
	productStore       store.ProductStore
	categoryStore      store.CategoryStore
	ingredientStore    store.IngredientStore
	clientStore        store.ClientStore
	providerStore      store.ProviderStore
	paymentMethodStore store.PaymentMethodStore
	orderStore         store.OrderStore
	localStockService  *services.LocalStockService
	localSaleService   *services.LocalSaleService
	renderer           *views.Renderer
	logger             *slog.Logger
}

func NewWebHandler(
	userStore store.UserStore,
	tokenStore store.TokenStore,
	productStore store.ProductStore,
	categoryStore store.CategoryStore,
	ingredientStore store.IngredientStore,
	clientStore store.ClientStore,
	providerStore store.ProviderStore,
	paymentMethodStore store.PaymentMethodStore,
	orderStore store.OrderStore,
	localStockService *services.LocalStockService,
	localSaleService *services.LocalSaleService,
	logger *slog.Logger,
) *WebHandler {
	return &WebHandler{
		userStore:          userStore,
		tokenStore:         tokenStore,
		productStore:       productStore,
		categoryStore:      categoryStore,
		ingredientStore:    ingredientStore,
		clientStore:        clientStore,
		providerStore:      providerStore,
		paymentMethodStore: paymentMethodStore,
		orderStore:         orderStore,
		localStockService:  localStockService,
		localSaleService:   localSaleService,
		renderer:           views.NewRenderer(),
		logger:             logger,
	}
}
