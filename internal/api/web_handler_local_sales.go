package api

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/go-chi/chi/v5"
)

func (h *WebHandler) HandleListLocalSales(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	// Employee or Admin
	if user.Role != "administrator" && user.Role != "employee" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	dateStr := r.URL.Query().Get("date")
	targetDate := time.Now()
	if dateStr != "" {
		if d, err := time.Parse("2006-01-02", dateStr); err == nil {
			targetDate = d
		}
	} else {
		dateStr = targetDate.Format("2006-01-02")
	}

	sales, err := h.localSaleService.ListSalesByDate(targetDate)
	if err != nil {
		h.logger.Error("listing local sales", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Enhance with Payment Method Name?
	// Sales struct has ID. We might want to show method name.
	pMethods, _ := h.paymentMethodStore.GetAllPaymentMethods()
	pmMap := make(map[int64]string)
	for _, pm := range pMethods {
		pmMap[pm.ID] = pm.Name
	}

	type SaleView struct {
		ID            int64
		PaymentMethod string
		Total         string
		Date          string
	}

	var saleViews []SaleView
	for _, s := range sales {
		saleViews = append(saleViews, SaleView{
			ID:            s.ID,
			PaymentMethod: pmMap[s.PaymentMethodID],
			Total:         s.Total,
			Date:          s.CreatedAt.Format("02/01/2006 15:04"),
		})
	}

	data := map[string]any{
		"User":        user,
		"Sales":       saleViews,
		"CurrentDate": dateStr,
	}

	if err := h.renderer.Render(w, "local_sales_list.html", data); err != nil {
		h.logger.Error("rendering local sales list", "error", err)
	}
}

func (h *WebHandler) HandleCreateLocalSaleView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" && user.Role != "employee" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	products, _ := h.productStore.GetAllProduct()
	pMethods, _ := h.paymentMethodStore.GetAllPaymentMethods()

	data := map[string]any{
		"User":           user,
		"Products":       products,
		"PaymentMethods": pMethods,
	}

	if err := h.renderer.Render(w, "local_sale_form.html", data); err != nil {
		h.logger.Error("rendering local sale form", "error", err)
	}
}

func (h *WebHandler) HandleCreateLocalSale(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" && user.Role != "employee" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pmID, _ := strconv.ParseInt(r.FormValue("payment_method_id"), 10, 64)

	productIDs := r.PostForm["product_ids[]"]
	quantities := r.PostForm["quantities[]"]

	var items []services.CreateLocalSaleItem
	for i, pidStr := range productIDs {
		pid, _ := strconv.ParseInt(pidStr, 10, 64)
		qty, _ := strconv.Atoi(quantities[i])
		if pid > 0 && qty > 0 {
			items = append(items, services.CreateLocalSaleItem{
				ProductID: pid,
				Quantity:  qty,
			})
		}
	}

	req := services.CreateLocalSaleRequest{
		PaymentMethodID: pmID,
		Items:           items,
	}

	_, err := h.localSaleService.CreateLocalSale(req)
	if err != nil {
		h.logger.Error("creating local sale", "error", err)
		msg := err.Error()
		// Customize message for insufficient stock to be user friendly if needed,
		// but service provides details.
		http.Redirect(w, r, "/local-sales/new?error="+url.QueryEscape(msg), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/local-sales", http.StatusSeeOther)
}

func (h *WebHandler) HandleGetLocalSaleView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" && user.Role != "employee" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	sale, err := h.localSaleService.GetSale(id)
	if err != nil {
		h.logger.Error("getting local sale", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if sale == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	pm, err := h.paymentMethodStore.GetPaymentMethodByID(sale.PaymentMethodID)
	if err != nil {
		h.logger.Error("getting payment method", "error", err)
	}
	pmName := "Unknown"
	if pm != nil {
		pmName = pm.Name
	}

	var productIDs []int64
	for _, item := range sale.Items {
		productIDs = append(productIDs, item.ProductID)
	}
	products, err := h.productStore.GetProductsByIDs(productIDs)
	if err != nil {
		h.logger.Error("getting products", "error", err)
	}

	type ItemView struct {
		ProductName  string
		Quantity     int
		UnitPrice    string
		LineSubtotal string
	}

	var itemViews []ItemView
	for _, item := range sale.Items {
		pName := "Unknown Product"
		if p, ok := products[item.ProductID]; ok {
			pName = p.Name
		}
		itemViews = append(itemViews, ItemView{
			ProductName:  pName,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			LineSubtotal: item.LineSubtotal,
		})
	}

	type SaleView struct {
		ID    int64
		Total string
		Date  string
		Items []ItemView
	}

	saleView := SaleView{
		ID:    sale.ID,
		Total: sale.Total,
		Date:  sale.CreatedAt.Format("02/01/2006 15:04"),
		Items: itemViews,
	}

	backDate := r.URL.Query().Get("date")
	if backDate == "" {
		backDate = time.Now().Format("2006-01-02")
	}

	data := map[string]any{
		"User":              user,
		"Sale":              saleView,
		"PaymentMethodName": pmName,
		"BackDate":          backDate,
	}

	if err := h.renderer.Render(w, "local_sale_detail.html", data); err != nil {
		h.logger.Error("rendering local sale detail", "error", err)
	}
}
