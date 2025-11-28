package api

import (
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
