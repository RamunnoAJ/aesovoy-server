package api

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

// --- Payment Methods ---

func (h *WebHandler) HandleListPaymentMethods(w http.ResponseWriter, r *http.Request) {
	h.triggerMessages(w, r)
	user := middleware.GetUser(r)
	paymentMethods, err := h.paymentMethodStore.GetAllPaymentMethods()
	if err != nil {
		h.logger.Error("listing payment methods", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":           user,
		"PaymentMethods": paymentMethods,
	}

	if err := h.renderer.Render(w, "payment_methods_list.html", data); err != nil {
		h.logger.Error("rendering payment methods list", "error", err)
	}
}

func (h *WebHandler) HandleCreatePaymentMethodView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	data := map[string]any{
		"User":          user,
		"PaymentMethod": store.PaymentMethod{},
	}

	if err := h.renderer.Render(w, "payment_method_form.html", data); err != nil {
		h.logger.Error("rendering payment method form", "error", err)
	}
}

func (h *WebHandler) HandleCreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pm := &store.PaymentMethod{
		Name:      r.FormValue("name"),
		Reference: r.FormValue("reference"),
	}

	if err := h.paymentMethodStore.CreatePaymentMethod(pm); err != nil {
		h.logger.Error("creating payment method", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/payment_methods?success="+url.QueryEscape("Método de pago creado exitosamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleEditPaymentMethodView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	pmID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	pm, err := h.paymentMethodStore.GetPaymentMethodByID(pmID)
	if err != nil {
		h.logger.Error("getting payment method", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if pm == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":          user,
		"PaymentMethod": pm,
	}

	if err := h.renderer.Render(w, "payment_method_form.html", data); err != nil {
		h.logger.Error("rendering payment method form", "error", err)
	}
}

func (h *WebHandler) HandleUpdatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	pmID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pm := &store.PaymentMethod{
		ID:        pmID,
		Name:      r.FormValue("name"),
		Reference: r.FormValue("reference"),
	}

	if err := h.paymentMethodStore.UpdatePaymentMethod(pm); err != nil {
		h.logger.Error("updating payment method", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/payment_methods?success="+url.QueryEscape("Método de pago actualizado correctamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user.Role != "administrator" {
		utils.TriggerToast(w, "No tienes permiso para realizar esta acción", "error")
		utils.Error(w, http.StatusForbidden, "Forbidden")
		return
	}

	pmID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID inválido", "error")
		utils.Error(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	if err := h.paymentMethodStore.DeletePaymentMethod(pmID); err != nil {
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			utils.TriggerToast(w, "No se puede eliminar: el método de pago tiene ventas asociadas.", "error")
			utils.Error(w, http.StatusConflict, "No se puede eliminar: el método de pago tiene ventas asociadas.")
			return
		}
		h.logger.Error("deleting payment method", "error", err)
		utils.TriggerToast(w, "Error interno del servidor", "error")
		utils.Error(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.TriggerToast(w, "Método de pago eliminado", "success")
	w.WriteHeader(http.StatusOK)
}
