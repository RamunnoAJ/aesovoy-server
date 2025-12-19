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

// --- Clients ---

func (h *WebHandler) HandleListClients(w http.ResponseWriter, r *http.Request) {
	h.triggerMessages(w, r)
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

	http.Redirect(w, r, "/clients?success="+url.QueryEscape("Cliente creado exitosamente"), http.StatusSeeOther)
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

	http.Redirect(w, r, "/clients?success="+url.QueryEscape("Cliente actualizado correctamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteClient(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID de cliente inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.clientStore.DeleteClient(clientID); err != nil {
		h.logger.Error("deleting client", "error", err)
		utils.TriggerToast(w, "Error al eliminar cliente", "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.TriggerToast(w, "Cliente eliminado", "success")
	w.WriteHeader(http.StatusOK)
}
