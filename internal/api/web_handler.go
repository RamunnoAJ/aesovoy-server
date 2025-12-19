package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/mailer"
	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
	chi "github.com/go-chi/chi/v5"
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
	expenseStore       store.ExpenseStore
	localStockService  *services.LocalStockService
	localSaleService   *services.LocalSaleService
	shiftService       *services.ShiftService
	mailer             *mailer.Mailer
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
	expenseStore store.ExpenseStore,
	localStockService *services.LocalStockService,
	localSaleService *services.LocalSaleService,
	shiftService *services.ShiftService,
	mailer *mailer.Mailer,
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
		expenseStore:       expenseStore,
		localStockService:  localStockService,
		localSaleService:   localSaleService,
		shiftService:       shiftService,
		mailer:             mailer,
		renderer:           views.NewRenderer(),
		logger:             logger,
	}
}

// --- Provider Categories (Integrated into Providers Management) ---

func (h *WebHandler) HandleCreateProviderCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.ProviderCategory{
		Name: r.FormValue("name"),
	}

	if err := h.providerStore.CreateProviderCategory(category); err != nil {
		h.logger.Error("creating provider category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Category": category,
	}

	if err := h.renderer.Render(w, "provider_category_row.html", data); err != nil {
		h.logger.Error("rendering new provider category row", "error", err)
	}
}

func (h *WebHandler) HandleQuickCreateProviderCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.ProviderCategory{
		Name: r.FormValue("name"),
	}

	if err := h.providerStore.CreateProviderCategory(category); err != nil {
		h.logger.Error("creating provider category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	html := "<option value=\"" + strconv.FormatInt(category.ID, 10) + "\" selected>" + category.Name + "</option>"
	w.Write([]byte(html))
}

func (h *WebHandler) HandleUpdateProviderCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.ProviderCategory{
		ID:   id,
		Name: r.FormValue("name"),
	}

	if err := h.providerStore.UpdateProviderCategory(category); err != nil {
		h.logger.Error("updating provider category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Category": category,
	}

	if err := h.renderer.Render(w, "provider_category_row.html", data); err != nil {
		h.logger.Error("rendering updated provider category row", "error", err)
	}
}

func (h *WebHandler) HandleDeleteProviderCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.providerStore.DeleteProviderCategory(id); err != nil {
		h.logger.Error("deleting provider category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebHandler) HandleGetProviderCategoryEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	category, err := h.providerStore.GetProviderCategoryByID(id)
	if err != nil {
		h.logger.Error("getting provider category for edit form", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if category == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"Category": category,
	}

	if err := h.renderer.Render(w, "provider_category_edit_row.html", data); err != nil {
		h.logger.Error("rendering provider category edit form", "error", err)
	}
}

