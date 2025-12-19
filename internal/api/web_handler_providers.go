package api

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

// --- Providers ---

func (h *WebHandler) HandleListProviders(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	q := r.URL.Query().Get("q")
	categoryIDStr := r.URL.Query().Get("category_id")
	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	var categoryID *int64
	if categoryIDStr != "" {
		cid, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err == nil {
			categoryID = &cid
		}
	}

	providers, err := h.providerStore.SearchProvidersFTS(q, categoryID, limit+1, offset)
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
		"Categories": categories,
		"CategoryID": categoryID, // Add current category filter
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

	http.Redirect(w, r, "/providers?success="+url.QueryEscape("Proveedor creado exitosamente"), http.StatusSeeOther)
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

	http.Redirect(w, r, "/providers?success="+url.QueryEscape("Proveedor actualizado correctamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	providerID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID de proveedor inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.providerStore.DeleteProvider(providerID); err != nil {
		h.logger.Error("deleting provider", "error", err)
		utils.TriggerToast(w, "Error al eliminar proveedor", "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.TriggerToast(w, "Proveedor eliminado", "success")
	w.WriteHeader(http.StatusOK)
}
