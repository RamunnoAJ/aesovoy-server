package routes

import (
	"github.com/RamunnoAJ/aesovoy-server/internal/app"
	"github.com/go-chi/chi/v5"
)

func SetupRoutes(app *app.Application) *chi.Mux {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(app.Middleware.Authenticate)
		r.Use(app.Middleware.RequireUser)

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
	})

	r.Get("/health", app.HealthCheck)
	r.Post("/users", app.UserHandler.HandleRegisterUser)
	r.Post("/tokens/authentication", app.TokenHandler.HandleCreateToken)
	return r
}
