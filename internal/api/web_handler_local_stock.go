package api

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

func (h *WebHandler) HandleUpdateLocalStock(w http.ResponseWriter, r *http.Request) {
	// Used by HTMX to update stock
	user := middleware.GetUser(r)
	if user.Role != "administrator" && user.Role != "employee" {
		utils.TriggerToast(w, "No tienes permiso para esta acción", "error")
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
		utils.TriggerToast(w, "Error al obtener stock", "error")
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
		initialQty := delta 
		if initialQty < 0 {
			utils.TriggerToast(w, "No se puede reducir stock de 0", "error")
			http.Error(w, "Cannot decrease 0 stock", http.StatusBadRequest)
			return
		}
		_, err = h.localStockService.CreateInitialStock(pid, initialQty)
	} else {
		_, err = h.localStockService.AdjustStock(pid, delta)
	}

	if err != nil {
		h.logger.Error("adjusting stock", "error", err)
		utils.TriggerToast(w, "Error: "+err.Error(), "error")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Redirect based on input or default to local-stock list
	redirectTo := r.FormValue("redirect_to")
	if redirectTo == "" {
		redirectTo = "/products"
	}

	http.Redirect(w, r, redirectTo+"?success="+url.QueryEscape("Stock actualizado"), http.StatusSeeOther)
}
