package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	chi "github.com/go-chi/chi/v5"
)

// --- Providers ---

func (h *WebHandler) HandleListProviders(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	q := r.URL.Query().Get("q")
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	providers, err := h.providerStore.SearchProvidersFTS(q, limit+1, offset)
	if err != nil {
		h.logger.Error("listing providers", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasNext := false
	if len(providers) > limit {
		hasNext = true
		providers = providers[:limit]
	}

	categories, err := h.providerStore.GetAllProviderCategories()
	if err != nil {
		h.logger.Error("getting provider categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Providers":  providers,
		"Categories": categories, // Add categories to the data
		"Query":      q,
		"Page":       page,
		"HasNext":    hasNext,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	}

	if err := h.renderer.Render(w, "providers_list.html", data); err != nil {
		h.logger.Error("rendering providers list", "error", err)
	}
}

func (h *WebHandler) HandleCreateProviderView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	categories, err := h.providerStore.GetAllProviderCategories()
	if err != nil {
		h.logger.Error("getting provider categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Provider":   store.Provider{},
		"Categories": categories,
	}

	if err := h.renderer.Render(w, "provider_form.html", data); err != nil {
		h.logger.Error("rendering provider form", "error", err)
	}
}

func (h *WebHandler) HandleCreateProvider(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)

	provider := &store.Provider{
		Name:       r.FormValue("name"),
		Address:    r.FormValue("address"),
		Phone:      r.FormValue("phone"),
		Reference:  r.FormValue("reference"),
		Email:      r.FormValue("email"),
		CUIT:       r.FormValue("cuit"),
		CategoryID: categoryID,
	}

	if err := h.providerStore.CreateProvider(provider); err != nil {
		h.logger.Error("creating provider", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/providers", http.StatusSeeOther)
}

func (h *WebHandler) HandleEditProviderView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	providerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	provider, err := h.providerStore.GetProviderByID(providerID)
	if err != nil {
		h.logger.Error("getting provider", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if provider == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	categories, err := h.providerStore.GetAllProviderCategories()
	if err != nil {
		h.logger.Error("getting provider categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Provider":   provider,
		"Categories": categories,
	}

	if err := h.renderer.Render(w, "provider_form.html", data); err != nil {
		h.logger.Error("rendering provider form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	providerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	categoryID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)

	provider := &store.Provider{
		ID:         providerID,
		Name:       r.FormValue("name"),
		Address:    r.FormValue("address"),
		Phone:      r.FormValue("phone"),
		Reference:  r.FormValue("reference"),
		Email:      r.FormValue("email"),
		CUIT:       r.FormValue("cuit"),
		CategoryID: categoryID,
	}

	if err := h.providerStore.UpdateProvider(provider); err != nil {
		h.logger.Error("updating provider", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/providers", http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	providerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.providerStore.DeleteProvider(providerID); err != nil {
		h.logger.Error("deleting provider", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
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
	}}

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
	}}


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
