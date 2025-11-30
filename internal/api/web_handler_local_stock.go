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
	if user.Role != "administrator" && user.Role != "employee" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pidStr := r.FormValue("product_id")
	pid, _ := strconv.ParseInt(pidStr, 10, 64)

	// Check if we are setting absolute quantity or delta
	newQtyStr := r.FormValue("new_quantity")
	
	stock, err := h.localStockService.GetStock(pid)
	if err != nil {
		h.logger.Error("getting stock", "error", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	var delta int

	if newQtyStr != "" {
		// Absolute Set
		newQty, _ := strconv.Atoi(newQtyStr)
		currentQty := 0
		if stock != nil {
			currentQty = stock.Quantity
		}
		delta = newQty - currentQty
	} else {
		// Relative Delta
		deltaStr := r.FormValue("delta")
		delta, _ = strconv.Atoi(deltaStr)
	}

	if stock == nil {
		// Create initial if needed
		// If delta makes it negative, create fails inside service usually? 
		// Service CreateInitial takes absolute.
		// If we are here, stock is nil.
		// If we have absolute newQty, create with that.
		// If we have delta, assume start 0 + delta.
		
		initialQty := delta 
		if initialQty < 0 {
			http.Error(w, "Cannot decrease 0 stock", http.StatusBadRequest)
			return
		}
		_, err = h.localStockService.CreateInitialStock(pid, initialQty)
	} else {
		_, err = h.localStockService.AdjustStock(pid, delta)
	}

	if err != nil {
		h.logger.Error("adjusting stock", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If request came from modal (redirect usually) or HTMX (partial)?
	// If form submit, redirect. If HTMX, redirect or refresh.
	// Simplest for Modal: Redirect to list.
	http.Redirect(w, r, "/local-stock", http.StatusSeeOther)
}
