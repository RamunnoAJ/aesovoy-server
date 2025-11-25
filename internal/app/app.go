package app

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/migrations"
)

type Application struct {
	Logger            *log.Logger
	UserHandler       *api.UserHandler
	TokenHandler      *api.TokenHandler
	CategoryHandler   *api.CategoryHandler
	ProductHandler    *api.ProductHandler
	ClientHandler     *api.ClientHandler
	ProviderHandler   *api.ProviderHandler
	OrderHandler      *api.OrderHandler
	IngredientHandler *api.IngredientHandler
	Middleware        middleware.UserMiddleware
	DB                *sql.DB
}

func NewApplication() (*Application, error) {
	pgDB, err := store.Open()
	if err != nil {
		return nil, err
	}

	err = store.MigrateFS(pgDB, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	// our stores will go here
	userStore := store.NewPostgresUserStore(pgDB)
	tokenStore := store.NewPostgresTokenStore(pgDB)
	categoryStore := store.NewPostgresCategoryStore(pgDB)
	productStore := store.NewPostgresProductStore(pgDB)
	clientStore := store.NewPostgresClientStore(pgDB)
	providerStore := store.NewPostgresProviderStore(pgDB)
	orderStore := store.NewPostgresOrderStore(pgDB)
	ingredientStore := store.NewPostgresIngredientStore(pgDB)

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

	app := &Application{
		Logger:            logger,
		UserHandler:       userHandler,
		TokenHandler:      tokenHandler,
		Middleware:        middlewareHandler,
		CategoryHandler:   categoryHandler,
		ProductHandler:    productHandler,
		ClientHandler:     clientHandler,
		ProviderHandler:   providerHandler,
		OrderHandler:      orderHandler,
		IngredientHandler: ingredientHandler,
		DB:                pgDB,
	}

	return app, nil
}

func (a *Application) HealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Status is available\n")
}
