package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/billing"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type OrderItemReq struct {
	ProductID int64       `json:"product_id"`
	Quantity  int         `json:"quantity"`
	Price     store.Money `json:"price"`
}
type RegisterOrderRequest struct {
	ClientID int64            `json:"client_id"`
	State    store.OrderState `json:"state"`
	Items    []OrderItemReq   `json:"items"`
}

type OrderHandler struct {
	orders   store.OrderStore
	clients  store.ClientStore
	products store.ProductStore
	logger   *slog.Logger
}

func NewOrderHandler(os store.OrderStore, cs store.ClientStore, ps store.ProductStore, l *slog.Logger) *OrderHandler {
	return &OrderHandler{orders: os, clients: cs, products: ps, logger: l}
}

func (h *OrderHandler) validateCreate(req *RegisterOrderRequest) []utils.FieldError {
	var errs []utils.FieldError
	if req.ClientID <= 0 {
		errs = append(errs, utils.FieldError{Field: "client_id", Message: "must be > 0"})
	}
	if len(req.Items) == 0 {
		errs = append(errs, utils.FieldError{Field: "items", Message: "must not be empty"})
	}
	for i, it := range req.Items {
		if it.ProductID <= 0 {
			errs = append(errs, utils.FieldError{Field: fieldIdx("items[%d].product_id", i), Message: "must be > 0"})
		}
		if it.Quantity <= 0 {
			errs = append(errs, utils.FieldError{Field: fieldIdx("items[%d].quantity", i), Message: "must be > 0"})
		}
		if strings.TrimSpace(it.Price) == "" {
			errs = append(errs, utils.FieldError{Field: fieldIdx("items[%d].price", i), Message: "required NUMERIC string"})
		}
	}
	return errs
}

// HandleRegisterOrder godoc
// @Summary      Creates an order
// @Description  Creates a new order for a client with a list of items
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterOrderRequest  true  "Order data"
// @Success      201   {object}  OrderResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/orders [post]
func (h *OrderHandler) HandleRegisterOrder(w http.ResponseWriter, r *http.Request) {
	var req RegisterOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	errs := h.validateCreate(&req)
	if len(errs) > 0 {
		utils.Fail(w, http.StatusBadRequest, "validation failed", errs)
		return
	}

	o := &store.Order{
		ClientID: req.ClientID,
		State:    ternState(req.State, store.OrderTodo),
	}
	items := make([]store.OrderItem, len(req.Items))
	productIDs := make([]int64, len(req.Items))
	for i, it := range req.Items {
		items[i] = store.OrderItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
			Price:     it.Price,
		}
		productIDs[i] = it.ProductID
	}
	if err := h.orders.CreateOrder(o, items); err != nil {
		h.logger.Error("create order", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Invoice generation
	go func() {
		client, err := h.clients.GetClientByID(o.ClientID)
		if err != nil {
			h.logger.Error("generating invoice: could not get client", "clientID", o.ClientID, "error", err)
			return
		}
		if client == nil {
			h.logger.Error("generating invoice: client not found", "clientID", o.ClientID)
			return
		}

		products, err := h.products.GetProductsByIDs(productIDs)
		if err != nil {
			h.logger.Error("generating invoice: could not get products", "error", err)
			return
		}

		// The order object 'o' from CreateOrder has the final items list.
		if err := billing.GenerateInvoice(o, client, products); err != nil {
			h.logger.Error("generating invoice for order", "orderID", o.ID, "error", err)
		}
	}()

	utils.OK(w, http.StatusCreated, utils.Envelope{"order": o}, "", nil)
}

// HandleUpdateOrderState godoc
// @Summary      Updates an order's state
// @Description  Updates the state of an order (e.g., "todo", "done", "delivered")
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        id    path      int     true  "Order ID"
// @Param        body  body      object{state=store.OrderState}  true  "New state"
// @Success      200   {object}  UpdateOrderStateResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      404   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/orders/{id}/state [patch]
func (h *OrderHandler) HandleUpdateOrderState(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	var req struct {
		State store.OrderState `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.State == "" {
		utils.Fail(w, http.StatusBadRequest, "validation failed", []utils.FieldError{{Field: "state", Message: "required"}})
		return
	}
	if err := h.orders.UpdateOrderState(id, req.State); err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "order not found")
			return
		}
		h.logger.Error("update order state", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"id": id, "state": req.State}, "", nil)
}

// HandleGetOrderByID godoc
// @Summary      Gets an order
// @Description  Responds with a single order with a given ID
// @Tags         orders
// @Produce      json
// @Param        id   path      int      true  "Order ID"
// @Success      200  {object}  OrderResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/orders/{id} [get]
func (h *OrderHandler) HandleGetOrderByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}
	o, err := h.orders.GetOrderByID(id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if o == nil {
		utils.Error(w, http.StatusNotFound, "order not found")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"order": o}, "", nil)
}

// HandleListOrders godoc
// @Summary      Lists orders
// @Description  Responds with a list of orders, with optional filters
// @Tags         orders
// @Produce      json
// @Param        client_id  query     int           false "Filter by client ID"
// @Param        state      query     string        false "Filter by order state"
// @Param        limit      query     int           false "Results-per-page limit"
// @Param        offset     query     int           false "Page offset for pagination"
// @Success      200        {object}  OrdersResponse
// @Failure      500        {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/orders [get]
func (h *OrderHandler) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	var (
		clientID *int64
		state    *store.OrderState
	)
	if v := r.URL.Query().Get("client_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
			clientID = &id
		}
	}
	if v := r.URL.Query().Get("state"); v != "" {
		st := store.OrderState(v)
		state = &st
	}
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	list, err := h.orders.ListOrders(store.OrderFilter{
		ClientID: clientID,
		State:    state,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"orders": list}, "", &utils.Meta{Limit: limit, Offset: offset, Total: len(list)})
}

func fieldIdx(format string, i int) string { return fmt.Sprintf(format, i) }
func ternState(v, def store.OrderState) store.OrderState {
	if v != "" {
		return v
	}
	return def
}
