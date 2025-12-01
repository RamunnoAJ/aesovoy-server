package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/go-chi/chi/v5"
)

// --- Products ---

func (h *WebHandler) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	q := r.URL.Query().Get("q")
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	// Fetch one extra to determine if there is a next page
	products, err := h.productStore.SearchProductsFTS(q, limit+1, offset)
	if err != nil {
		h.logger.Error("listing products", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasNext := false
	if len(products) > limit {
		hasNext = true
		products = products[:limit]
	}

	data := map[string]any{
		"User":     user,
		"Products": products,
		"Query":    q,
		"Page":     page,
		"HasNext":  hasNext,
		"PrevPage": page - 1,
		"NextPage": page + 1,
	}

	if err := h.renderer.Render(w, "products_list.html", data); err != nil {
		h.logger.Error("rendering products list", "error", err)
	}
}

func (h *WebHandler) HandleCreateProductView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	categories, err := h.categoryStore.GetAllCategories()
	if err != nil {
		h.logger.Error("fetching categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Categories": categories,
		"Product":    store.Product{},
	}

	if err := h.renderer.Render(w, "product_form.html", data); err != nil {
		h.logger.Error("rendering product form", "error", err)
	}
}

func (h *WebHandler) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	unitPrice, _ := strconv.ParseFloat(r.FormValue("unit_price"), 64)
	distPrice, _ := strconv.ParseFloat(r.FormValue("distribution_price"), 64)

	product := &store.Product{
		Name:              r.FormValue("name"),
		Description:       r.FormValue("description"),
		CategoryID:        categoryID,
		UnitPrice:         unitPrice,
		DistributionPrice: distPrice,
	}

	if err := h.productStore.CreateProduct(product); err != nil {
		h.logger.Error("creating product", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *WebHandler) HandleEditProductView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	product, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("getting product", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if product == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	categories, err := h.categoryStore.GetAllCategories()
	if err != nil {
		h.logger.Error("fetching categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Categories": categories,
		"Product":    product,
	}

	if err := h.renderer.Render(w, "product_form.html", data); err != nil {
		h.logger.Error("rendering product form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	unitPrice, _ := strconv.ParseFloat(r.FormValue("unit_price"), 64)
	distPrice, _ := strconv.ParseFloat(r.FormValue("distribution_price"), 64)

	product := &store.Product{
		ID:                productID,
		Name:              r.FormValue("name"),
		Description:       r.FormValue("description"),
		CategoryID:        categoryID,
		UnitPrice:         unitPrice,
		DistributionPrice: distPrice,
	}

	if err := h.productStore.UpdateProduct(product); err != nil {
		h.logger.Error("updating product", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/products", http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.productStore.DeleteProduct(productID); err != nil {
		h.logger.Error("deleting product", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) // HTMX will remove the element
}

// --- Recipes ---

func (h *WebHandler) HandleManageRecipeView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	product, err := h.productStore.GetProductByID(productID)
	if err != nil {
		h.logger.Error("getting product", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if product == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	allIngredients, err := h.ingredientStore.GetAllIngredients()
	if err != nil {
		h.logger.Error("getting all ingredients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":           user,
		"Product":        product,
		"AllIngredients": allIngredients,
	}

	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		w.Header().Set("HX-Trigger", fmt.Sprintf(`{"showToast": {"message": "%s", "type": "error"}}`, errMsg))
	}

	if err := h.renderer.Render(w, "product_recipe.html", data); err != nil {
		h.logger.Error("rendering product recipe", "error", err)
	}
}

func (h *WebHandler) HandleAddIngredientToRecipe(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid Product ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ingredientID, _ := strconv.ParseInt(r.FormValue("ingredient_id"), 10, 64)
	quantity, _ := strconv.ParseFloat(r.FormValue("quantity"), 64)
	unit := r.FormValue("unit")

	_, err = h.productStore.AddIngredientToProduct(productID, ingredientID, quantity, unit)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			http.Redirect(w, r, "/products/"+strconv.FormatInt(productID, 10)+"/recipe?error=El ingrediente ya existe en la receta", http.StatusSeeOther)
			return
		}
		h.logger.Error("adding ingredient to recipe", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/products/"+strconv.FormatInt(productID, 10)+"/recipe", http.StatusSeeOther)
}

func (h *WebHandler) HandleRemoveIngredientFromRecipe(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid Product ID", http.StatusBadRequest)
		return
	}

	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "ingredient_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid Ingredient ID", http.StatusBadRequest)
		return
	}

	err = h.productStore.RemoveIngredientFromProduct(productID, ingredientID)
	if err != nil {
		h.logger.Error("removing ingredient from recipe", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebHandler) HandleGetRecipeModal(w http.ResponseWriter, r *http.Request) {
	productID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	product, err := h.productStore.GetProductByID(productID)
	if err != nil || product == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<div class="p-6">
		<div class="flex justify-between items-center mb-4">
			<h3 class="text-xl font-bold text-gray-900">%s</h3>
			<button @click="openRecipe = false" class="text-gray-400 hover:text-gray-600">
				<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
			</button>
		</div>
		<h4 class="text-sm font-medium text-gray-500 uppercase tracking-wider mb-2">Ingredientes</h4>
		<ul class="divide-y divide-gray-200 border-t border-b border-gray-200">`, product.Name)

	for _, ing := range product.Recipe {
		qtyFormat := "%.2f"
		if ing.Unit == "g" || ing.Unit == "ml" {
			qtyFormat = "%.0f"
		}
		fmt.Fprintf(w, `<li class="py-3 flex justify-between items-center">
			<span class="text-gray-700">%s</span>
			<span class="font-mono font-medium text-gray-900">`+qtyFormat+` %s</span>
		</li>`, ing.Name, ing.Quantity, ing.Unit)
	}

	if len(product.Recipe) == 0 {
		fmt.Fprint(w, `<li class="py-3 text-center text-gray-500 italic">Este producto no tiene receta definida.</li>`)
	}

	fmt.Fprint(w, `</ul>
		<div class="mt-6 flex justify-end">
			<a href="/products/`+strconv.FormatInt(product.ID, 10)+`/recipe" class="text-blue-600 hover:text-blue-800 text-sm font-medium">Editar Receta &rarr;</a>
		</div>
	</div>`)
}
