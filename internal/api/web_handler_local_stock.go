package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
)

func (h *WebHandler) HandleListLocalStock(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	// Admin and Employee can see stock
	if user.Role != "administrator" && user.Role != "employee" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	stocks, err := h.localStockService.ListStock()
	if err != nil {
		h.logger.Error("listing local stock", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":   user,
		"Stocks": stocks,
	}

	if err := h.renderer.Render(w, "local_stock_list.html", data); err != nil {
		h.logger.Error("rendering local stock list", "error", err)
	}
}
func (h *WebHandler) HandleUpdateLocalStock(w http.ResponseWriter, r *http.Request) {
	// Used by HTMX to update stock
	user := middleware.GetUser(r)
	if user.Role != "administrator" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// If coming from a form or JSON? HTMX usually sends Form data, but let's support query/form for simple +/- buttons.
	// Or maybe we use a modal with a form.
	// Let's assume we receive `product_id` and `delta` in body or query.

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pidStr := r.FormValue("product_id")
	deltaStr := r.FormValue("delta")

	pid, _ := strconv.ParseInt(pidStr, 10, 64)
	delta, _ := strconv.Atoi(deltaStr)

	// If record doesn't exist, we might need CreateInitialStock logic.
	// The Service `AdjustStock` fails if record not found.
	// We should check or use a service method that "Upserts".
	// Let's check if stock exists first.
	stock, err := h.localStockService.GetStock(pid)
	if err != nil {
		h.logger.Error("getting stock", "error", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	if stock == nil {
		// Create initial if delta is positive
		if delta < 0 {
			http.Error(w, "Cannot decrease 0 stock", http.StatusBadRequest)
			return
		}
		_, err = h.localStockService.CreateInitialStock(pid, delta)
	} else {
		_, err = h.localStockService.AdjustStock(pid, delta)
	}

	if err != nil {
		h.logger.Error("adjusting stock", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the new quantity to update the UI (HTMX)
	// We can just return the number as text to swap the cell
	newStock, _ := h.localStockService.GetStock(pid) // fetch fresh
	w.Write([]byte(strconv.Itoa(newStock.Quantity)))
}
