package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

type CreateLocalSaleRequest struct {
	PaymentMethodID int64                          `json:"payment_method_id"`
	Items           []services.CreateLocalSaleItem `json:"items"`
}

type LocalSaleHandler struct {
	service *services.LocalSaleService
	logger  *slog.Logger
}

func NewLocalSaleHandler(s *services.LocalSaleService, l *slog.Logger) *LocalSaleHandler {
	return &LocalSaleHandler{service: s, logger: l}
}

// HandleCreateLocalSale godoc
// @Summary      Create a new local sale
// @Description  Creates a new sale, validates stock, and adjusts it transactionally.
// @Tags         local_sales
// @Accept       json
// @Produce      json
// @Param        body  body      CreateLocalSaleRequest  true  "Local Sale data"
// @Success      201   {object}  LocalSaleResponse
// @Failure      400   {object}  utils.HTTPError "Invalid input, insufficient stock, etc."
// @Failure      404   {object}  utils.HTTPError "Resource not found (e.g., product, payment method)"
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_sales [post]
func (h *LocalSaleHandler) HandleCreateLocalSale(w http.ResponseWriter, r *http.Request) {
	var req services.CreateLocalSaleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	sale, err := h.service.CreateLocalSale(req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProductNotFound), errors.Is(err, services.ErrPaymentMethodNotFound):
			utils.Error(w, http.StatusNotFound, err.Error())
		case errors.Is(err, services.ErrInsufficientStock):
			utils.Error(w, http.StatusBadRequest, err.Error())
		case err.Error() == "sale must have at least one item":
			utils.Error(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("creating local sale", "error", err)
			utils.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"local_sale": sale}, "", nil)
}

// HandleGetLocalSale godoc
// @Summary      Get a single local sale
// @Description  Retrieves the details of a single local sale by its ID.
// @Tags         local_sales
// @Produce      json
// @Param        id   path      int      true  "Sale ID"
// @Success      200  {object}  LocalSaleResponse
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_sales/{id} [get]
func (h *LocalSaleHandler) HandleGetLocalSale(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid sale ID")
		return
	}
	sale, err := h.service.GetSale(id)
	if err != nil {
		h.logger.Error("getting local sale", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if sale == nil {
		utils.Error(w, http.StatusNotFound, "sale not found")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"local_sale": sale}, "", nil)
}

// HandleListLocalSales godoc
// @Summary      List all local sales
// @Description  Responds with a list of all local sales.
// @Tags         local_sales
// @Produce      json
// @Success      200  {object}  LocalSalesResponse
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_sales [get]
func (h *LocalSaleHandler) HandleListLocalSales(w http.ResponseWriter, r *http.Request) {
	sales, err := h.service.ListSales()
	if err != nil {
		h.logger.Error("listing local sales", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"local_sales": sales}, "", nil)
}
