package routes

import (
	"net/http"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/app"
	mymw "github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	_ "github.com/RamunnoAJ/aesovoy-server/swagger"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	httpSwagger "github.com/swaggo/http-swagger"
)

func SetupRoutes(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(mymw.AddSecurityHeaders)
	r.Use(mymw.CSRFProtection)
	r.Use(mymw.Logging(app.Logger))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Global Rate Limiter: 100 requests per minute per IP
	r.Use(httprate.Limit(
		100,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
	))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(app.Middleware.Authenticate)
		r.Use(app.Middleware.RequireUser)

		// Local Store (Accessible by Employees and Admins)
		r.Route("/local_stock", func(r chi.Router) {
			r.Get("/", app.LocalStockHandler.HandleListLocalStock)
			r.Post("/", app.LocalStockHandler.HandleCreateInitialStock)
			r.Get("/{product_id}", app.LocalStockHandler.HandleGetLocalStock)
			r.Patch("/{product_id}/adjust", app.LocalStockHandler.HandleAdjustStock)
		})

		r.Route("/local_sales", func(r chi.Router) {
			r.Get("/", app.LocalSaleHandler.HandleListLocalSales)
			r.Post("/", app.LocalSaleHandler.HandleCreateLocalSale)
			r.Get("/{id}", app.LocalSaleHandler.HandleGetLocalSale)
		})

		// Admin Only API
		r.Group(func(r chi.Router) {
			r.Use(app.Middleware.RequireAdmin)

			r.Route("/categories", func(r chi.Router) {
				r.Get("/", app.CategoryHandler.HandleGetCategories)
				r.Get("/{id}", app.CategoryHandler.HandleGetCategoryByID)
				r.Get("/{id}/products", app.ProductHandler.HandleGetProductsByCategory)
				r.Post("/", app.CategoryHandler.HandleRegisterCategory)
				r.Patch("/{id}", app.CategoryHandler.HandleUpdateCategory)
				r.Delete("/{id}", app.CategoryHandler.HandleDeleteCategory)
			})

			r.Route("/products", func(r chi.Router) {
				r.Get("/", app.ProductHandler.HandleGetProducts)
				r.Get("/{id}", app.ProductHandler.HandleGetProductByID)
				r.Post("/", app.ProductHandler.HandleRegisterProduct)
				r.Patch("/{id}", app.ProductHandler.HandleUpdateProduct)
				r.Delete("/{id}", app.ProductHandler.HandleDeleteProduct)

				r.Post("/{productID}/ingredients", app.ProductHandler.HandleAddIngredientToProduct)
				r.Patch("/{productID}/ingredients/{ingredientID}", app.ProductHandler.HandleUpdateProductIngredient)
				r.Delete("/{productID}/ingredients/{ingredientID}", app.ProductHandler.HandleRemoveIngredientFromProduct)
			})

			r.Route("/ingredients", func(r chi.Router) {
				r.Get("/", app.IngredientHandler.HandleGetAllIngredients)
				r.Post("/", app.IngredientHandler.HandleCreateIngredient)
				r.Get("/{id}", app.IngredientHandler.HandleGetIngredientByID)
				r.Patch("/{id}", app.IngredientHandler.HandleUpdateIngredient)
				r.Delete("/{id}", app.IngredientHandler.HandleDeleteIngredient)
			})

			r.Route("/clients", func(r chi.Router) {
				r.Get("/", app.ClientHandler.HandleGetClients)
				r.Get("/{id}", app.ClientHandler.HandleGetClientByID)
				r.Post("/", app.ClientHandler.HandleRegisterClient)
				r.Patch("/{id}", app.ClientHandler.HandleUpdateClient)
			})

			r.Route("/providers", func(r chi.Router) {
				r.Get("/", app.ProviderHandler.HandleGetProviders)
				r.Get("/{id}", app.ProviderHandler.HandleGetProviderByID)
				r.Post("/", app.ProviderHandler.HandleRegisterProvider)
				r.Patch("/{id}", app.ProviderHandler.HandleUpdateProvider)
			})

			r.Route("/orders", func(r chi.Router) {
				r.Get("/", app.OrderHandler.HandleListOrders)
				r.Get("/{id}", app.OrderHandler.HandleGetOrderByID)
				r.Post("/", app.OrderHandler.HandleRegisterOrder)
				r.Patch("/{id}/state", app.OrderHandler.HandleUpdateOrderState)
			})

			r.Route("/payment_methods", func(r chi.Router) {
				r.Get("/", app.PaymentMethodHandler.HandleGetPaymentMethods)
				r.Get("/{id}", app.PaymentMethodHandler.HandleGetPaymentMethodByID)
				r.Post("/", app.PaymentMethodHandler.HandleCreatePaymentMethod)
				r.Delete("/{id}", app.PaymentMethodHandler.HandleDeletePaymentMethod)
			})

			r.Route("/expenses", func(r chi.Router) {
				r.Get("/", app.ExpenseHandler.HandleGetExpenses)
				r.Post("/", app.ExpenseHandler.HandleCreateExpense)
				r.Get("/{id}", app.ExpenseHandler.HandleGetExpenseByID)
				r.Delete("/{id}", app.ExpenseHandler.HandleDeleteExpense)
			})

			// API Tokens
			r.Post("/users", app.UserHandler.HandleRegisterUser)
			r.Post("/tokens/authentication", app.TokenHandler.HandleCreateToken)
		})
	})

	// Serve uploaded files
	r.Handle("/uploads/*", http.StripPrefix("/uploads", http.FileServer(http.Dir("uploads"))))

	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("http://localhost:8080/swagger/doc.json")))
	r.Get("/health", app.HealthCheck)

	// Public Web Views
	r.Get("/login", app.WebHandler.HandleShowLogin)
	r.With(httprate.Limit(
		10,
		1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP),
	)).Post("/login", app.WebHandler.HandleWebLogin)

	r.Get("/forgot-password", app.WebHandler.HandleShowForgotPassword)
	r.Post("/forgot-password", app.WebHandler.HandleSendPasswordResetEmail)
	r.Get("/reset-password", app.WebHandler.HandleShowResetPassword)
	r.Post("/reset-password", app.WebHandler.HandleResetPassword)

	// Protected Web Views
	r.Group(func(r chi.Router) {
		r.Use(app.Middleware.Authenticate)
		r.Use(app.Middleware.RequireUser)

		r.Get("/", app.WebHandler.HandleHome)
		r.Get("/time", app.WebHandler.HandleTime)
		r.Post("/logout", app.WebHandler.HandleLogout)

		// Invoices (Web View)
		r.Route("/invoices", func(r chi.Router) {
			r.Get("/", app.InvoiceHandler.List)
			r.Get("/download/{filename}", app.InvoiceHandler.Download)
			r.Delete("/{filename}", app.InvoiceHandler.Delete)
		})

		// Web Users Management (Admin)
		r.Group(func(r chi.Router) {
			r.Use(app.Middleware.RequireAdmin)
			r.Get("/users", app.WebHandler.HandleListUsers)
			r.Patch("/users/{id}/toggle-status", app.WebHandler.HandleToggleUserStatus)
		})

		// Products
		r.Get("/products", app.WebHandler.HandleListProducts)
		r.Get("/products/new", app.WebHandler.HandleCreateProductView)
		r.Post("/products/new", app.WebHandler.HandleCreateProduct)
		r.Get("/products/{id}/edit", app.WebHandler.HandleEditProductView)
		r.Post("/products/{id}/edit", app.WebHandler.HandleUpdateProduct)
		r.Delete("/products/{id}/delete", app.WebHandler.HandleDeleteProduct)

		// Categories
		r.Get("/categories", app.WebHandler.HandleListCategories)
		r.Get("/categories/new", app.WebHandler.HandleCreateCategoryView)
		r.Post("/categories/new", app.WebHandler.HandleCreateCategory)
		r.Post("/categories/quick", app.WebHandler.HandleQuickCreateCategory)
		r.Get("/categories/{id}/edit", app.WebHandler.HandleEditCategoryView)
		r.Post("/categories/{id}/edit", app.WebHandler.HandleUpdateCategory)
		r.Delete("/categories/{id}/delete", app.WebHandler.HandleDeleteCategory)

		// Ingredients
		r.Get("/ingredients", app.WebHandler.HandleListIngredients)
		r.Get("/ingredients/new", app.WebHandler.HandleCreateIngredientView)
		r.Post("/ingredients/new", app.WebHandler.HandleCreateIngredient)
		r.Get("/ingredients/{id}/edit", app.WebHandler.HandleEditIngredientView)
		r.Post("/ingredients/{id}/edit", app.WebHandler.HandleUpdateIngredient)
		r.Delete("/ingredients/{id}/delete", app.WebHandler.HandleDeleteIngredient)

		// Clients
		r.Get("/clients", app.WebHandler.HandleListClients)
		r.Get("/clients/new", app.WebHandler.HandleCreateClientView)
		r.Post("/clients/new", app.WebHandler.HandleCreateClient)
		r.Get("/clients/{id}/edit", app.WebHandler.HandleEditClientView)
		r.Post("/clients/{id}/edit", app.WebHandler.HandleUpdateClient)
		r.Delete("/clients/{id}/delete", app.WebHandler.HandleDeleteClient)

		// Providers
		r.Get("/providers", app.WebHandler.HandleListProviders)
		r.Get("/providers/new", app.WebHandler.HandleCreateProviderView)
		r.Post("/providers/new", app.WebHandler.HandleCreateProvider)
		r.Get("/providers/{id}/edit", app.WebHandler.HandleEditProviderView)
		r.Post("/providers/{id}/edit", app.WebHandler.HandleUpdateProvider)
		r.Delete("/providers/{id}/delete", app.WebHandler.HandleDeleteProvider)

		// Provider Categories (integrated into Providers UI)
		r.Route("/providers/categories", func(r chi.Router) {
			r.Post("/new", app.WebHandler.HandleCreateProviderCategory)
			r.Post("/quick", app.WebHandler.HandleQuickCreateProviderCategory)
			r.Delete("/{id}/delete", app.WebHandler.HandleDeleteProviderCategory)
			r.Get("/{id}/edit-form", app.WebHandler.HandleGetProviderCategoryEditForm)
			r.Put("/{id}/edit", app.WebHandler.HandleUpdateProviderCategory)
		})

		// Recipes
		r.Get("/products/{id}/recipe", app.WebHandler.HandleManageRecipeView)
		r.Get("/products/{id}/recipe-modal", app.WebHandler.HandleGetRecipeModal)
		r.Post("/products/{id}/recipe", app.WebHandler.HandleAddIngredientToRecipe)
		r.Delete("/products/{id}/ingredients/{ingredient_id}", app.WebHandler.HandleRemoveIngredientFromRecipe)

		// Payment Methods
		r.Get("/payment_methods", app.WebHandler.HandleListPaymentMethods)
		r.Get("/payment_methods/new", app.WebHandler.HandleCreatePaymentMethodView)
		r.Post("/payment_methods/new", app.WebHandler.HandleCreatePaymentMethod)
		r.Get("/payment_methods/{id}/edit", app.WebHandler.HandleEditPaymentMethodView)
		r.Post("/payment_methods/{id}/edit", app.WebHandler.HandleUpdatePaymentMethod)
		r.Delete("/payment_methods/{id}/delete", app.WebHandler.HandleDeletePaymentMethod)

		// Orders
		r.Get("/orders", app.WebHandler.HandleListOrders)
		r.Get("/orders/new", app.WebHandler.HandleCreateOrderView)
		r.Post("/orders/new", app.WebHandler.HandleCreateOrder)
		r.Get("/orders/{id}", app.WebHandler.HandleGetOrderView)
		r.Patch("/orders/{id}/state", app.WebHandler.HandleUpdateOrderState)

		// Local Stock (Admin only checked in handler)
		r.Post("/local-stock/update", app.WebHandler.HandleUpdateLocalStock)

		// Local Sales (Admin/Employee checked in handler)
		r.Get("/local-sales", app.WebHandler.HandleListLocalSales)
		r.Get("/local-sales/new", app.WebHandler.HandleCreateLocalSaleView)
		r.Post("/local-sales/new", app.WebHandler.HandleCreateLocalSale)
		r.Get("/local-sales/{id}", app.WebHandler.HandleGetLocalSaleView)
		r.Delete("/local-sales/{id}", app.WebHandler.HandleRevokeLocalSale)

		// Shift Management (Admin/Employee)
		r.Get("/shifts", app.WebHandler.HandleShiftManagement)
		r.Post("/shifts/open", app.WebHandler.HandleOpenShift)
		r.Post("/shifts/close", app.WebHandler.HandleCloseShift)

		// Production Calculator (Employee and Admin)
		r.Get("/production-calculator", app.WebHandler.HandleShowProductionCalculator)
		r.Post("/production-calculator", app.WebHandler.HandleCalculateProduction)

		// Pending Production Ingredients (Admin Only)
		r.Group(func(r chi.Router) {
			r.Use(app.Middleware.RequireAdmin)
			r.Get("/pending-production-ingredients", app.WebHandler.HandleShowPendingProductionIngredients)
		})

		// Expenses (Admin Only)
		r.Group(func(r chi.Router) {
			r.Use(app.Middleware.RequireAdmin)
			r.Route("/expenses", func(r chi.Router) {
				r.Get("/", app.WebHandler.HandleListExpenses)
				r.Get("/new", app.WebHandler.HandleCreateExpenseView)
				r.Post("/new", app.WebHandler.HandleCreateExpense)
				r.Delete("/{id}", app.WebHandler.HandleDeleteExpense)
				r.Get("/{id}/image", app.WebHandler.HandleGetExpenseImage)
				r.Post("/categories/quick", app.WebHandler.HandleQuickCreateExpenseCategory)
			})
		})

	})

	return r
}
