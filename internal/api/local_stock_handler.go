package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/services"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	"github.com/go-chi/chi/v5"
)

// --- DTOs for Requests ---

type CreateInitialStockRequest struct {
	ProductID      int64 `json:"product_id"`
	InitialQuantity int   `json:"initial_quantity"`
}

type AdjustStockRequest struct {
	Delta int `json:"delta"`
}

// --- Handler ---

type LocalStockHandler struct {
	service *services.LocalStockService
	logger  *slog.Logger
}

func NewLocalStockHandler(s *services.LocalStockService, l *slog.Logger) *LocalStockHandler {
	return &LocalStockHandler{service: s, logger: l}
}

// --- Endpoints ---

// HandleGetLocalStock godoc
// @Summary      Get stock for a single product
// @Description  Responds with the local stock quantity for a given product ID
// @Tags         local_stock
// @Produce      json
// @Param        product_id   path      int      true  "Product ID"
// @Success      200  {object}  LocalStockResponse
// @Failure      404  {object}  utils.HTTPError "Stock record not found"
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_stock/{product_id} [get]
func (h *LocalStockHandler) HandleGetLocalStock(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	stock, err := h.service.GetStock(productID)
	if err != nil {
		h.logger.Error("getting local stock", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if stock == nil {
		utils.Error(w, http.StatusNotFound, "stock record not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"local_stock": stock}, "", nil)
}

// HandleListLocalStock godoc
// @Summary      List all local stock
// @Description  Responds with a list of all local stock records
// @Tags         local_stock
// @Produce      json
// @Success      200  {object}  LocalStocksResponse
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_stock [get]
func (h *LocalStockHandler) HandleListLocalStock(w http.ResponseWriter, r *http.Request) {
	stocks, err := h.service.ListStock()
	if err != nil {
		h.logger.Error("listing local stock", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"local_stock": stocks}, "", nil)
}

// HandleCreateInitialStock godoc
// @Summary      Create initial stock for a product
// @Description  Creates the first stock record for a product. Fails if a record already exists.
// @Tags         local_stock
// @Accept       json
// @Produce      json
// @Param        body  body      CreateInitialStockRequest  true  "Initial stock data"
// @Success      201   {object}  LocalStockResponse
// @Failure      400   {object}  utils.HTTPError "Invalid input"
// @Failure      404   {object}  utils.HTTPError "Product not found"
// @Failure      409   {object}  utils.HTTPError "Stock record already exists"
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_stock [post]
func (h *LocalStockHandler) HandleCreateInitialStock(w http.ResponseWriter, r *http.Request) {
	var req CreateInitialStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	stock, err := h.service.CreateInitialStock(req.ProductID, req.InitialQuantity)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrProductNotFound):
			utils.Error(w, http.StatusNotFound, err.Error())
		case errors.Is(err, services.ErrStockRecordExists):
			utils.Error(w, http.StatusConflict, err.Error())
		case errors.Is(err, services.ErrInitialQuantityInvalid):
			utils.Error(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("creating initial stock", "error", err)
			utils.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"local_stock": stock}, "", nil)
}

// HandleAdjustStock godoc
// @Summary      Adjust stock for a product
// @Description  Adjusts a product's stock quantity by a delta (can be positive or negative).
// @Tags         local_stock
// @Accept       json
// @Produce      json
// @Param        product_id   path      int                 true  "Product ID"
// @Param        body         body      AdjustStockRequest  true  "Adjustment data"
// @Success      200          {object}  LocalStockResponse
// @Failure      400          {object}  utils.HTTPError "Invalid input or insufficient stock"
// @Failure      500          {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/local_stock/{product_id}/adjust [patch]
func (h *LocalStockHandler) HandleAdjustStock(w http.ResponseWriter, r *http.Request) {
	productID, err := strconv.ParseInt(chi.URLParam(r, "product_id"), 10, 64)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid product ID")
		return
	}

	var req AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	stock, err := h.service.AdjustStock(productID, req.Delta)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInsufficientStock):
			utils.Error(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, services.ErrStockRecordNotFound):
             // This case may not be hit if the service auto-creates the record.
			utils.Error(w, http.StatusNotFound, err.Error())
		default:
			h.logger.Error("adjusting stock", "error", err)
			utils.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"local_stock": stock}, "", nil)
}
