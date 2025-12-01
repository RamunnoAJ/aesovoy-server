package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
)

func (h *WebHandler) HandleShowProductionCalculator(w http.ResponseWriter, r *http.Request) {
	products, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("failed to get all products", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Products": products,
		"Result":   nil,
	}

	err = h.renderer.Render(w, "production_calculator.html", data)
	if err != nil {
		h.logger.Error("failed to render production calculator", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleCalculateProduction(w http.ResponseWriter, r *http.Request) {
	productIDStr := r.FormValue("product_id")
	quantityStr := r.FormValue("quantity")

	productID, err := strconv.ParseInt(productIDStr, 10, 64)
	if err != nil {
		h.logger.Warn("invalid product ID", "product_id", productIDStr, "error", err)
		h.renderError(w, r, "ID de producto inválido", http.StatusBadRequest)
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		h.logger.Warn("invalid quantity", "quantity", quantityStr, "error", err)
		h.renderError(w, r, "Cantidad inválida. Debe ser un número positivo.", http.StatusBadRequest)
		return
	}

	product, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("failed to get product by ID", "product_id", productID, "error", err)
		h.renderError(w, r, "Producto no encontrado", http.StatusNotFound)
		return
	}
	if product == nil {
		h.renderError(w, r, "Producto no encontrado", http.StatusNotFound)
		return
	}

	// Calculate total ingredients
	calculatedIngredients := make(map[string]float64)
	ingredientUnits := make(map[string]string)

	for _, pi := range product.Recipe {
		totalIngredientQuantity := pi.Quantity * float64(quantity)
		calculatedIngredients[pi.Name] += totalIngredientQuantity
		ingredientUnits[pi.Name] = pi.Unit
	}

	resultData := map[string]any{
		"Product":           product,
		"RequestedQuantity": quantity,
		"Ingredients":       calculatedIngredients,
		"IngredientUnits":   ingredientUnits,
	}

	products, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("failed to get all products for form", "error", err)
	}

	data := map[string]any{
		"Products": products,
		"Result":   resultData,
	}

	if r.Header.Get("HX-Request") == "true" {
		err = h.renderer.Render(w, "production_calculator_results.html", data)
	} else {
		err = h.renderer.Render(w, "production_calculator.html", data)
	}

	if err != nil {
		h.logger.Error("failed to render calculation result", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) renderError(w http.ResponseWriter, r *http.Request, msg string, status int) {
	products, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("failed to get all products for error view", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Error":    msg,
		"Products": products,
	}

	// Set HTTP status before rendering the template
	w.WriteHeader(status)

	// Elegir template según si es HTMX o no
	tpl := "production_calculator.html"
	if r.Header.Get("HX-Request") == "true" {
		tpl = "production_calculator_results.html"
	}

	if err := h.renderer.Render(w, tpl, data); err != nil {
		h.logger.Error("failed to render error view", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleShowPendingProductionIngredients(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	requirements, err := h.orderStore.GetPendingProductionRequirements()
	if err != nil {
		h.logger.Error("failed to get pending production requirements", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	aggregatedIngredients := make(map[string]float64) // ingredientName -> totalQuantity
	ingredientUnits := make(map[string]string)        // ingredientName -> unit

	for _, req := range requirements {
		product, err := h.productStore.GetProductByID(req.ProductID)
		if err != nil {
			h.logger.Error("failed to get product for pending requirement", "product_id", req.ProductID, "error", err)
			// Continue, but log the error.
			continue
		}
		if product == nil {
			h.logger.Warn("product not found for pending requirement", "product_id", req.ProductID)
			continue
		}

		for _, pi := range product.Recipe {
			totalIngredientQuantity := pi.Quantity * float64(req.Quantity)
			aggregatedIngredients[pi.Name] += totalIngredientQuantity
			ingredientUnits[pi.Name] = pi.Unit // Store unit for display
		}
	}

	data := map[string]any{
		"User":                  user, // Add user to the data map
		"AggregatedIngredients": aggregatedIngredients,
		"IngredientUnits":       ingredientUnits,
		"Requirements":          requirements,
	}

	err = h.renderer.Render(w, "pending_production_ingredients.html", data)
	if err != nil {
		h.logger.Error("failed to render pending production ingredients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
