package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/go-chi/chi/v5"
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

	data := map[string]any{
		"User":      user,
		"Providers": providers,
		"Query":     q,
		"Page":     page,
		"HasNext":   hasNext,
		"PrevPage":  page - 1,
		"NextPage":  page + 1,
	}

	if err := h.renderer.Render(w, "providers_list.html", data); err != nil {
		h.logger.Error("rendering providers list", "error", err)
	}
}

func (h *WebHandler) HandleCreateProviderView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User":     user,
		"Provider": store.Provider{},
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

	provider := &store.Provider{
		Name:      r.FormValue("name"),
		Address:   r.FormValue("address"),
		Phone:     r.FormValue("phone"),
		Reference: r.FormValue("reference"),
		Email:     r.FormValue("email"),
		CUIT:      r.FormValue("cuit"),
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

	data := map[string]any{
		"User":     user,
		"Provider": provider,
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

	provider := &store.Provider{
		ID:        providerID,
		Name:      r.FormValue("name"),
		Address:   r.FormValue("address"),
		Phone:     r.FormValue("phone"),
		Reference: r.FormValue("reference"),
		Email:     r.FormValue("email"),
		CUIT:      r.FormValue("cuit"),
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
