package api

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/billing"
	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

// --- Orders ---

func (h *WebHandler) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	pageStr := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	filter := store.OrderFilter{
		Limit:  limit + 1,
		Offset: offset,
	}

	var stateStr string
	if s := r.URL.Query().Get("state"); s != "" {
		st := store.OrderState(s)
		filter.State = &st
		stateStr = s
	}

	q := r.URL.Query().Get("q")
	filter.ClientName = q

	startDateStr := r.URL.Query().Get("start_date")
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filter.StartDate = &t
		}
	} else {
		// Default to last 7 days if no start_date is provided
		now := time.Now()
		sevenDaysAgo := now.AddDate(0, 0, -7)
		defaultStart := time.Date(sevenDaysAgo.Year(), sevenDaysAgo.Month(), sevenDaysAgo.Day(), 0, 0, 0, 0, now.Location())
		filter.StartDate = &defaultStart
		startDateStr = defaultStart.Format("2006-01-02")
	}

	endDateStr := r.URL.Query().Get("end_date")
	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// Set to end of day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.EndDate = &t
		}
	} else {
		// Default to today's end if no end_date is provided
		now := time.Now()
		todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
		filter.EndDate = &todayEnd
		endDateStr = todayEnd.Format("2006-01-02")
	}

	orders, err := h.orderStore.ListOrders(filter)
	if err != nil {
		h.logger.Error("listing orders", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasNext := false
	if len(orders) > limit {
		hasNext = true
		orders = orders[:limit]
	}

	data := map[string]any{
		"User":      user,
		"Orders":    orders,
		"Page":      page,
		"HasNext":   hasNext,
		"PrevPage":  page - 1,
		"NextPage":  page + 1,
		"State":     stateStr,
		"Q":         q,
		"StartDate": startDateStr,
		"EndDate":   endDateStr,
	}

	if err := h.renderer.Render(w, "orders_list.html", data); err != nil {
		h.logger.Error("rendering orders list", "error", err)
	}
}

func (h *WebHandler) HandleUpdateOrderState(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID de orden inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	state := store.OrderState(r.URL.Query().Get("state"))
	if state == "" {
		utils.TriggerToast(w, "Falta el estado de la orden", "error")
		http.Error(w, "Missing state", http.StatusBadRequest)
		return
	}

	if err := h.orderStore.UpdateOrderState(orderID, state); err != nil {
		h.logger.Error("updating order state", "error", err)
		utils.TriggerToast(w, "Error al actualizar estado: "+err.Error(), "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Trigger a success toast
	utils.TriggerToast(w, "Estado de orden actualizado correctamente", "success")
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *WebHandler) HandleCreateOrderView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	clients, err := h.clientStore.GetAllClients()
	if err != nil {
		h.logger.Error("fetching clients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	products, err := h.productStore.GetAllProduct()
	if err != nil {
		h.logger.Error("fetching products", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":     user,
		"Clients":  clients,
		"Products": products,
	}

	if err := h.renderer.Render(w, "order_form.html", data); err != nil {
		h.logger.Error("rendering order form", "error", err)
	}
}

func (h *WebHandler) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	state := store.OrderState(r.FormValue("state"))

	productIDs := r.PostForm["product_ids[]"]
	quantities := r.PostForm["quantities[]"]
	prices := r.PostForm["prices[]"]

	if len(productIDs) == 0 || len(productIDs) != len(quantities) || len(productIDs) != len(prices) {
		http.Error(w, "Invalid items data", http.StatusBadRequest)
		return
	}

	var items []store.OrderItem
	var itemProductIDs []int64 // For invoice generation

	for i, pidStr := range productIDs {
		pid, _ := strconv.ParseInt(pidStr, 10, 64)
		qty, _ := strconv.Atoi(quantities[i])
		price := prices[i]

		if pid > 0 && qty > 0 {
			items = append(items, store.OrderItem{
				ProductID: pid,
				Quantity:  qty,
				Price:     price,
			})
			itemProductIDs = append(itemProductIDs, pid)
		}
	}

	order := &store.Order{
		ClientID: clientID,
		State:    state,
	}

	if err := h.orderStore.CreateOrder(order, items); err != nil {
		h.logger.Error("creating order", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	go func() {
		client, err := h.clientStore.GetClientByID(clientID)
		if err != nil || client == nil {
			h.logger.Error("invoice gen: client not found", "error", err)
			return
		}

		products, err := h.productStore.GetProductsByIDs(itemProductIDs)
		if err != nil {
			h.logger.Error("invoice gen: products error", "error", err)
			return
		}

		if err := billing.GenerateInvoice(order, client, products); err != nil {
			h.logger.Error("generating invoice", "error", err)
		}
	}()

	http.Redirect(w, r, "/orders?success="+url.QueryEscape("Orden creada exitosamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleGetOrderView(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	user := middleware.GetUser(r)
	order, err := h.orderStore.GetOrderByID(orderID)
	if err != nil {
		h.logger.Error("getting order", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if order == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":  user,
		"Order": order,
	}

	if err := h.renderer.Render(w, "order_detail.html", data); err != nil {
		h.logger.Error("rendering order detail", "error", err)
	}
}
