package app

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/migrations"
)

type Application struct {
	Logger               *slog.Logger
	UserHandler          *api.UserHandler
	TokenHandler         *api.TokenHandler
	CategoryHandler      *api.CategoryHandler
	ProductHandler       *api.ProductHandler
	ClientHandler        *api.ClientHandler
	ProviderHandler      *api.ProviderHandler
	OrderHandler         *api.OrderHandler
	IngredientHandler    *api.IngredientHandler
	PaymentMethodHandler *api.PaymentMethodHandler
	Middleware           middleware.UserMiddleware
	DB                   *sql.DB
}

func NewApplication() (*Application, error) {
	LOG_FILE := os.Getenv("LOG_FILE")
	file, err := os.OpenFile(LOG_FILE, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	pgDB, err := store.Open()
	if err != nil {
		return nil, err
	}

	err = store.MigrateFS(pgDB, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	w := io.MultiWriter(os.Stderr, file)
	handlerOpts := &slog.HandlerOptions{
		AddSource: true,
	}
	logger := slog.New(slog.NewTextHandler(w, handlerOpts))

	// our stores will go here
	userStore := store.NewPostgresUserStore(pgDB)
	tokenStore := store.NewPostgresTokenStore(pgDB)
	categoryStore := store.NewPostgresCategoryStore(pgDB)
	productStore := store.NewPostgresProductStore(pgDB)
	clientStore := store.NewPostgresClientStore(pgDB)
	providerStore := store.NewPostgresProviderStore(pgDB)
	orderStore := store.NewPostgresOrderStore(pgDB)
	ingredientStore := store.NewPostgresIngredientStore(pgDB)
	paymentMethodStore := store.NewPostgresPaymentMethodStore(pgDB)

	// our handlers will go here
	userHandler := api.NewUserHandler(userStore, logger)
	tokenHandler := api.NewTokenHandler(tokenStore, userStore, logger)
	middlewareHandler := middleware.UserMiddleware{UserStore: userStore}
	categoryHandler := api.NewCategoryHandler(categoryStore, logger)
	productHandler := api.NewProductHandler(productStore, logger)
	clientHandler := api.NewClientHandler(clientStore, logger)
	providerHandler := api.NewProviderHandler(providerStore, logger)
	orderHandler := api.NewOrderHandler(orderStore, clientStore, productStore, logger)
	ingredientHandler := api.NewIngredientHandler(ingredientStore, logger)
	paymentMethodHandler := api.NewPaymentMethodHandler(paymentMethodStore, logger)

	app := &Application{
		Logger:               logger,
		UserHandler:          userHandler,
		TokenHandler:         tokenHandler,
		Middleware:           middlewareHandler,
		CategoryHandler:      categoryHandler,
		ProductHandler:       productHandler,
		ClientHandler:        clientHandler,
		ProviderHandler:      providerHandler,
		OrderHandler:         orderHandler,
		IngredientHandler:    ingredientHandler,
		PaymentMethodHandler: paymentMethodHandler,
		DB:                   pgDB,
	}

	return app, nil
}

func (a *Application) HealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Status is available\n")
}
