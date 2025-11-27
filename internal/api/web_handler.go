package api

import (
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type WebHandler struct {
	userStore       store.UserStore
	tokenStore      store.TokenStore
	productStore    store.ProductStore
	categoryStore   store.CategoryStore
	ingredientStore store.IngredientStore
	clientStore     store.ClientStore
	renderer        *views.Renderer
	logger          *slog.Logger
}

func NewWebHandler(
	userStore store.UserStore,
	tokenStore store.TokenStore,
	productStore store.ProductStore,
	categoryStore store.CategoryStore,
	ingredientStore store.IngredientStore,
	clientStore store.ClientStore,
	logger *slog.Logger,
) *WebHandler {
	return &WebHandler{
		userStore:       userStore,
		tokenStore:      tokenStore,
		productStore:    productStore,
		categoryStore:   categoryStore,
		ingredientStore: ingredientStore,
		clientStore:     clientStore,
		renderer:        views.NewRenderer(),
		logger:          logger,
	}
}

func (h *WebHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User": user,
	}

	err := h.renderer.Render(w, "home.html", data)
	if err != nil {
		h.logger.Error("failed to render home", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})

	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *WebHandler) HandleTime(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(time.Now().Format(time.RFC1123)))
}

func (h *WebHandler) HandleShowLogin(w http.ResponseWriter, r *http.Request) {
	err := h.renderer.Render(w, "login.html", nil)
	if err != nil {
		h.logger.Error("failed to render login", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleWebLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.logger.Error("parsing form", "error", err)
		h.renderLoginError(w, "Invalid request")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.userStore.GetUserByUsername(username)
	if err != nil {
		h.logger.Error("getting user by username", "error", err)
		h.renderLoginError(w, "Credenciales incorrectas")
		return
	}

	if user == nil {
		h.renderLoginError(w, "Credenciales incorrectas")
		return
	}

	match, err := user.PasswordHash.Matches(password)
	if err != nil {
		h.logger.Error("matching password", "error", err)
		h.renderLoginError(w, "Error interno del servidor")
		return
	}

	if !match {
		h.renderLoginError(w, "Credenciales incorrectas")
		return
	}

	token, err := tokens.GenerateToken(int(user.ID), 24*time.Hour, tokens.ScopeAuth)
	if err != nil {
		h.logger.Error("generating token", "error", err)
		h.renderLoginError(w, "Internal server error")
		return
	}

	err = h.tokenStore.Insert(token)
	if err != nil {
		h.logger.Error("inserting token", "error", err)
		h.renderLoginError(w, "Internal server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token.Plaintext,
		Expires:  token.Expiry,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("HX-Redirect", "/")
}

func (h *WebHandler) renderLoginError(w http.ResponseWriter, msg string) {
	data := map[string]any{
		"Error": msg,
	}
	// When using HTMX with hx-swap="outerHTML", we want to re-render the form (login.html)
	// But if we just render "login.html", it might render with the base layout if using hx-boost or if it's a full page load.
	// However, our login.html template defines "content".
	// The renderer currently executes "base.html" which includes "content".
	// If this is an HTMX request targeting the form, we might just want the form HTML.
	// BUT, our renderer is simple and always wraps in base.
	// For now, let's just re-render the whole page. HTMX is smart enough to extract if needed,
	// OR we can just let it swap the body.
	// The login form has hx-target="body" hx-swap="outerHTML".
	// If we return the full page, the <body> of the response will replace the <body> of the current page.
	err := h.renderer.Render(w, "login.html", data)
	if err != nil {
		h.logger.Error("failed to render login error", "error", err)
	}
}

// --- Products ---

func (h *WebHandler) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	products, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("listing products", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":     user,
		"Products": products,
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

// --- Categories ---

func (h *WebHandler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	categories, err := h.categoryStore.GetAllCategories()
	if err != nil {
		h.logger.Error("listing categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Categories": categories,
	}

	if err := h.renderer.Render(w, "categories_list.html", data); err != nil {
		h.logger.Error("rendering categories list", "error", err)
	}
}

func (h *WebHandler) HandleCreateCategoryView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User":     user,
		"Category": store.Category{},
	}

	if err := h.renderer.Render(w, "category_form.html", data); err != nil {
		h.logger.Error("rendering category form", "error", err)
	}
}

func (h *WebHandler) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.Category{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	if err := h.categoryStore.CreateCategory(category); err != nil {
		h.logger.Error("creating category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *WebHandler) HandleEditCategoryView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	category, err := h.categoryStore.GetCategoryByID(categoryID)
	if err != nil {
		h.logger.Error("getting category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if category == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":     user,
		"Category": category,
	}

	if err := h.renderer.Render(w, "category_form.html", data); err != nil {
		h.logger.Error("rendering category form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.Category{
		ID:          categoryID,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	if err := h.categoryStore.UpdateCategory(category); err != nil {
		h.logger.Error("updating category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.categoryStore.DeleteCategory(categoryID); err != nil {
		h.logger.Error("deleting category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// --- Ingredients ---

func (h *WebHandler) HandleListIngredients(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	ingredients, err := h.ingredientStore.GetAllIngredients()
	if err != nil {
		h.logger.Error("listing ingredients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":        user,
		"Ingredients": ingredients,
	}

	if err := h.renderer.Render(w, "ingredients_list.html", data); err != nil {
		h.logger.Error("rendering ingredients list", "error", err)
	}
}

func (h *WebHandler) HandleCreateIngredientView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User":       user,
		"Ingredient": store.Ingredient{},
	}

	if err := h.renderer.Render(w, "ingredient_form.html", data); err != nil {
		h.logger.Error("rendering ingredient form", "error", err)
	}
}

func (h *WebHandler) HandleCreateIngredient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ingredient := &store.Ingredient{
		Name: r.FormValue("name"),
	}

	if err := h.ingredientStore.CreateIngredient(ingredient); err != nil {
		h.logger.Error("creating ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/ingredients", http.StatusSeeOther)
}

func (h *WebHandler) HandleEditIngredientView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ingredient, err := h.ingredientStore.GetIngredientByID(ingredientID)
	if err != nil {
		h.logger.Error("getting ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if ingredient == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":       user,
		"Ingredient": ingredient,
	}

	if err := h.renderer.Render(w, "ingredient_form.html", data); err != nil {
		h.logger.Error("rendering ingredient form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateIngredient(w http.ResponseWriter, r *http.Request) {
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ingredient := &store.Ingredient{
		ID:   ingredientID,
		Name: r.FormValue("name"),
	}

	if err := h.ingredientStore.UpdateIngredient(ingredient); err != nil {
		h.logger.Error("updating ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/ingredients", http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteIngredient(w http.ResponseWriter, r *http.Request) {
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.ingredientStore.DeleteIngredient(ingredientID); err != nil {
		h.logger.Error("deleting ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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

// --- Users Management (Admin) ---

func (h *WebHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r)

	users, err := h.userStore.GetAllUsers()
	if err != nil {
		h.logger.Error("listing users", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":  currentUser,
		"Users": users,
	}

	if err := h.renderer.Render(w, "users_list.html", data); err != nil {
		h.logger.Error("rendering users list", "error", err)
	}
}

func (h *WebHandler) HandleToggleUserStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r)
	targetUserID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Prevent self-disable
	if int64(currentUser.ID) == targetUserID {
		http.Error(w, "Cannot disable your own account", http.StatusBadRequest)
		return
	}

	if err := h.userStore.ToggleUserStatus(targetUserID); err != nil {
		h.logger.Error("toggling user status", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get updated user to re-render the row
	updatedUser, err := h.userStore.GetUserByID(targetUserID)
	if err != nil {
		h.logger.Error("getting updated user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := h.renderer.RenderPartial(w, "user_row.html", updatedUser); err != nil {
		h.logger.Error("rendering user row", "error", err)
	}
}

// --- Clients ---

func (h *WebHandler) HandleListClients(w http.ResponseWriter, r *http.Request) {
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
	clients, err := h.clientStore.SearchClientsFTS(q, limit+1, offset)
	if err != nil {
		h.logger.Error("listing clients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasNext := false
	if len(clients) > limit {
		hasNext = true
		clients = clients[:limit]
	}

	data := map[string]any{
		"User":     user,
		"Clients":  clients,
		"Query":    q,
		"Page":     page,
		"HasNext":  hasNext,
		"PrevPage": page - 1,
		"NextPage": page + 1,
	}

	if err := h.renderer.Render(w, "clients_list.html", data); err != nil {
		h.logger.Error("rendering clients list", "error", err)
	}
}

func (h *WebHandler) HandleCreateClientView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User":   user,
		"Client": store.Client{},
	}

	if err := h.renderer.Render(w, "client_form.html", data); err != nil {
		h.logger.Error("rendering client form", "error", err)
	}
}

func (h *WebHandler) HandleCreateClient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	client := &store.Client{
		Name:      r.FormValue("name"),
		Address:   r.FormValue("address"),
		Phone:     r.FormValue("phone"),
		Reference: r.FormValue("reference"),
		Email:     r.FormValue("email"),
		CUIT:      r.FormValue("cuit"),
		Type:      store.ClientType(r.FormValue("type")),
	}

	if err := h.clientStore.CreateClient(client); err != nil {
		h.logger.Error("creating client", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

func (h *WebHandler) HandleEditClientView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	clientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	client, err := h.clientStore.GetClientByID(clientID)
	if err != nil {
		h.logger.Error("getting client", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if client == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":   user,
		"Client": client,
	}

	if err := h.renderer.Render(w, "client_form.html", data); err != nil {
		h.logger.Error("rendering client form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateClient(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	client := &store.Client{
		ID:        clientID,
		Name:      r.FormValue("name"),
		Address:   r.FormValue("address"),
		Phone:     r.FormValue("phone"),
		Reference: r.FormValue("reference"),
		Email:     r.FormValue("email"),
		CUIT:      r.FormValue("cuit"),
		Type:      store.ClientType(r.FormValue("type")),
	}

	if err := h.clientStore.UpdateClient(client); err != nil {
		h.logger.Error("updating client", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteClient(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.clientStore.DeleteClient(clientID); err != nil {
		h.logger.Error("deleting client", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

