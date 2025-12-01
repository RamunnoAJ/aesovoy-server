package api

import (
	"net/http"

	"github.com/RamunnoAJ/aesovoy-server/internal/billing"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
	"github.com/go-chi/chi/v5"
)

type InvoiceHandler struct {
	renderer *views.Renderer
}

func NewInvoiceHandler(renderer *views.Renderer) *InvoiceHandler {
	return &InvoiceHandler{
		renderer: renderer,
	}
}

func (h *InvoiceHandler) List(w http.ResponseWriter, r *http.Request) {
	files, err := billing.ListInvoices()
	if err != nil {
		http.Error(w, "Could not list invoices", http.StatusInternalServerError)
		return
	}

	data := struct {
		Invoices []billing.InvoiceFile
		User     interface{}
	}{
		Invoices: files,
		User:     r.Context().Value(middleware.UserContextKey),
	}

	h.renderer.Render(w, "invoices_list.html", data)
}

func (h *InvoiceHandler) Download(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	path, err := billing.GetInvoicePath(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	http.ServeFile(w, r, path)
}
