package app

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/RamunnoAJ/aesovoy-server/internal/api"
	"github.com/RamunnoAJ/aesovoy-server/internal/mailer"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
	"github.com/RamunnoAJ/aesovoy-server/migrations"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
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
	LocalStockHandler    *api.LocalStockHandler
	LocalSaleHandler     *api.LocalSaleHandler
	InvoiceHandler       *api.InvoiceHandler
	ExpenseHandler       *api.ExpenseHandler
	WebHandler           *api.WebHandler
	Middleware           middleware.UserMiddleware
	DB                   *sql.DB
}

func NewApplication() (*Application, error) {
	LOG_FILE := os.Getenv("LOG_FILE")

	pgDB, err := store.Open()
	if err != nil {
		return nil, err
	}

	err = store.MigrateFS(pgDB, migrations.FS, ".")
	if err != nil {
		panic(err)
	}

	logRotator := &lumberjack.Logger{
		Filename:   LOG_FILE,
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	w := io.MultiWriter(os.Stdout, logRotator)
	handlerOpts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	logger := slog.New(slog.NewJSONHandler(w, handlerOpts))
	slog.SetDefault(logger)

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
	localStockStore := store.NewPostgresLocalStockStore(pgDB)
	localSaleStore := store.NewPostgresLocalSaleStore(pgDB)
	expenseStore := store.NewPostgresExpenseStore(pgDB)
	shiftStore := store.NewPostgresShiftStore(pgDB)

	// our services will go here
	localStockService := services.NewLocalStockService(localStockStore, productStore)
	localSaleService := services.NewLocalSaleService(pgDB, localSaleStore, localStockStore, paymentMethodStore, productStore)
	shiftService := services.NewShiftService(shiftStore, localSaleStore)

	mailer := mailer.New(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USERNAME"),
		os.Getenv("SMTP_PASSWORD"),
		os.Getenv("SMTP_FROM"),
	)

	// our handlers will go here
	renderer := views.NewRenderer()
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
	localStockHandler := api.NewLocalStockHandler(localStockService, logger)
	localSaleHandler := api.NewLocalSaleHandler(localSaleService, logger)
	invoiceHandler := api.NewInvoiceHandler(renderer)
	expenseHandler := api.NewExpenseHandler(expenseStore, logger)
	webHandler := api.NewWebHandler(
		userStore, tokenStore, productStore, categoryStore, ingredientStore,
		clientStore, providerStore, paymentMethodStore, orderStore, expenseStore,
		localStockService, localSaleService, shiftService, mailer, logger,
	)

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
		LocalStockHandler:    localStockHandler,
		LocalSaleHandler:     localSaleHandler,
		InvoiceHandler:       invoiceHandler,
		ExpenseHandler:       expenseHandler,
		WebHandler:           webHandler,
		DB:                   pgDB,
	}

	return app, nil
}

func (a *Application) HealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Status is available\n")
}
