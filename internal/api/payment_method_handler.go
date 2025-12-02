package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type registerPaymentMethodRequest struct {
	Name      string `json:"name"`
	Reference string `json:"reference"`
}

type PaymentMethodHandler struct {
	store  store.PaymentMethodStore
	logger *slog.Logger
}

func NewPaymentMethodHandler(s store.PaymentMethodStore, l *slog.Logger) *PaymentMethodHandler {
	return &PaymentMethodHandler{store: s, logger: l}
}

func (h *PaymentMethodHandler) validateRequest(req *registerPaymentMethodRequest) error {
	var errs utils.ValidationErrors
	if strings.TrimSpace(req.Name) == "" {
		errs = append(errs, utils.FieldError{Field: "name", Message: "is required"})
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// HandleCreatePaymentMethod godoc
// @Summary      Creates a payment method
// @Description  Creates a new payment method
// @Tags         payment_methods
// @Accept       json
// @Produce      json
// @Param        body  body      registerPaymentMethodRequest  true  "Payment Method data"
// @Success      201   {object}  PaymentMethodResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/payment_methods [post]
func (h *PaymentMethodHandler) HandleCreatePaymentMethod(w http.ResponseWriter, r *http.Request) {
	var req registerPaymentMethodRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.validateRequest(&req); err != nil {
		if ve, ok := err.(utils.ValidationErrors); ok {
			utils.Fail(w, http.StatusBadRequest, "validation failed", ve)
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	pm := &store.PaymentMethod{
		Name:      req.Name,
		Reference: req.Reference,
	}

	if err := h.store.CreatePaymentMethod(pm); err != nil {
		h.logger.Error("creating payment method", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"payment_method": pm}, "", nil)
}

// HandleGetPaymentMethodByID godoc
// @Summary      Gets a payment method
// @Description  Responds with a single payment method with a given ID
// @Tags         payment_methods
// @Produce      json
// @Param        id   path      int      true  "Payment Method ID"
// @Success      200  {object}  PaymentMethodResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/payment_methods/{id} [get]
func (h *PaymentMethodHandler) HandleGetPaymentMethodByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid payment method id")
		return
	}

	pm, err := h.store.GetPaymentMethodByID(id)
	if err != nil {
		h.logger.Error("getting payment method by id", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if pm == nil {
		utils.Error(w, http.StatusNotFound, "payment method not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"payment_method": pm}, "", nil)
}

// HandleGetPaymentMethods godoc
// @Summary      Gets all payment methods
// @Description  Responds with a list of all payment methods
// @Tags         payment_methods
// @Produce      json
// @Success      200  {object}  PaymentMethodsResponse
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/payment_methods [get]
func (h *PaymentMethodHandler) HandleGetPaymentMethods(w http.ResponseWriter, r *http.Request) {
	pms, err := h.store.GetAllPaymentMethods()
	if err != nil {
		h.logger.Error("getting all payment methods", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"payment_methods": pms}, "", nil)
}

// HandleDeletePaymentMethod godoc
// @Summary      Deletes a payment method
// @Description  Deletes a payment method with a given ID
// @Tags         payment_methods
// @Param        id   path      int  true  "Payment Method ID"
// @Success      204
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/payment_methods/{id} [delete]
func (h *PaymentMethodHandler) HandleDeletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid payment method id")
		return
	}

	if err := h.store.DeletePaymentMethod(id); err != nil {
		if err.Error() == "sql: no rows in result set" {
			utils.Error(w, http.StatusNotFound, "payment method not found")
			return
		}
		if strings.Contains(err.Error(), "violates foreign key constraint") {
			utils.Error(w, http.StatusConflict, "No se puede eliminar: el método de pago tiene ventas asociadas.")
			return
		}
		h.logger.Error("deleting payment method", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
