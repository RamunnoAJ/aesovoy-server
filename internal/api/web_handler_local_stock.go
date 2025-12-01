package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
)

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

	// Redirect based on input or default to local-stock list
	redirectTo := r.FormValue("redirect_to")
	if redirectTo == "" {
		redirectTo = "/local-stock"
	}

	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}
