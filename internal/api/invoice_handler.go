package api

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/billing"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/views"
	chi "github.com/go-chi/chi/v5"
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
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	dateFilter := r.URL.Query().Get("date")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	files, total, err := billing.ListInvoices(page, limit, dateFilter)
	if err != nil {
		http.Error(w, "Could not list invoices", http.StatusInternalServerError)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	data := struct {
		Invoices    []billing.InvoiceFile
		User        any
		CurrentPage int
		TotalPages  int
		CurrentDate string
	}{
		Invoices:    files,
		User:        r.Context().Value(middleware.UserContextKey),
		CurrentPage: page,
		TotalPages:  totalPages,
		CurrentDate: dateFilter,
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

// HandleListInvoicesJSON godoc
// @Summary      Lists invoices
// @Description  Responds with a list of generated invoices (Excel files), with pagination and date filter
// @Tags         invoices
// @Produce      json
// @Param        page   query     int     false "Page number (default 1)"
// @Param        limit  query     int     false "Items per page (default 20)"
// @Param        date   query     string  false "Filter by date string"
// @Success      200    {object}  InvoicesResponse
// @Failure      500    {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/invoices [get]
func (h *InvoiceHandler) HandleListInvoicesJSON(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	dateFilter := r.URL.Query().Get("date")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 20
	}

	files, total, err := billing.ListInvoices(page, limit, dateFilter)
	if err != nil {
		http.Error(w, "Could not list invoices", http.StatusInternalServerError)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	response := InvoicesResponse{
		Data: files,
		Meta: InvoicesMeta{
			CurrentPage:  page,
			TotalPages:   totalPages,
			TotalItems:   total,
			ItemsPerPage: limit,
			DateFilter:   dateFilter,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
func (h *InvoiceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if err := billing.DeleteInvoice(filename); err != nil {
		http.Error(w, "Could not delete invoice", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
