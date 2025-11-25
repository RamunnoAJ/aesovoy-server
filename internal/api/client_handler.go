package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type registerClientRequest struct {
	Name      string           `json:"name"`
	Address   string           `json:"address"`
	Phone     string           `json:"phone"`
	Reference string           `json:"reference"`
	Email     string           `json:"email"`
	CUIT      string           `json:"cuit"`
	Type      store.ClientType `json:"type"`
}

type ClientHandler struct {
	clientStore store.ClientStore
	logger      *slog.Logger
}

func NewClientHandler(clientStore store.ClientStore, logger *slog.Logger) *ClientHandler {
	return &ClientHandler{clientStore: clientStore, logger: logger}
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func (h *ClientHandler) validateRegisterRequest(req *registerClientRequest) error {
	var errs utils.ValidationErrors
	if strings.TrimSpace(req.Name) == "" {
		errs = append(errs, utils.FieldError{Field: "name", Message: "is required"})
	}
	if strings.TrimSpace(req.Reference) == "" {
		errs = append(errs, utils.FieldError{Field: "reference", Message: "is required"})
	}
	if strings.TrimSpace(req.CUIT) == "" {
		errs = append(errs, utils.FieldError{Field: "cuit", Message: "is required"})
	}
	if req.Email != "" && !emailRe.MatchString(req.Email) {
		errs = append(errs, utils.FieldError{Field: "email", Message: "invalid format"})
	}
	switch req.Type {
	case store.ClientTypeDistributer, store.ClientTypeIndividual, "":
	default:
		errs = append(errs, utils.FieldError{Field: "type", Message: "must be 'distributer' or 'individual'"})
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// HandleRegisterClient godoc
// @Summary      Creates a client
// @Description  Creates a new client
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        body  body      registerClientRequest  true  "Client data"
// @Success      201   {object}  ClientResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /clients [post]
func (h *ClientHandler) HandleRegisterClient(w http.ResponseWriter, r *http.Request) {
	var req registerClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if err := h.validateRegisterRequest(&req); err != nil {
		if ve, ok := err.(utils.ValidationErrors); ok {
			utils.Fail(w, http.StatusBadRequest, "validation failed", ve)
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	c := &store.Client{
		Name: req.Name, Address: req.Address, Phone: req.Phone,
		Reference: req.Reference, Email: req.Email, CUIT: req.CUIT,
		Type: req.Type,
	}
	if c.Type == "" {
		c.Type = store.ClientTypeIndividual
	}

	if err := h.clientStore.CreateClient(c); err != nil {
		h.logger.Error("creating client", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusCreated, utils.Envelope{"client": c}, "", nil)
}

// HandleUpdateClient godoc
// @Summary      Updates a client
// @Description  Updates a client's details
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        id    path      int                    true  "Client ID"
// @Param        body  body      registerClientRequest  true  "Client data"
// @Success      200   {object}  ClientResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      404   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /clients/{id} [patch]
func (h *ClientHandler) HandleUpdateClient(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid client id")
		return
	}
	cl, err := h.clientStore.GetClientByID(id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if cl == nil {
		utils.Error(w, http.StatusNotFound, "client not found")
		return
	}

	var req struct {
		Name      *string           `json:"name"`
		Address   *string           `json:"address"`
		Phone     *string           `json:"phone"`
		Reference *string           `json:"reference"`
		Email     *string           `json:"email"`
		CUIT      *string           `json:"cuit"`
		Type      *store.ClientType `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Name != nil {
		cl.Name = *req.Name
	}
	if req.Address != nil {
		cl.Address = *req.Address
	}
	if req.Phone != nil {
		cl.Phone = *req.Phone
	}
	if req.Reference != nil {
		cl.Reference = *req.Reference
	}
	if req.Email != nil {
		cl.Email = *req.Email
	}
	if req.CUIT != nil {
		cl.CUIT = *req.CUIT
	}
	if req.Type != nil {
		cl.Type = *req.Type
	}

	if err := h.clientStore.UpdateClient(cl); err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "client not found")
			return
		}
		h.logger.Error("updating client", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"client": cl}, "", nil)
}

// HandleGetClientByID godoc
// @Summary      Gets a client
// @Description  Responds with a single client with a given ID
// @Tags         clients
// @Produce      json
// @Param        id   path      int      true  "Client ID"
// @Success      200  {object}  ClientResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /clients/{id} [get]
func (h *ClientHandler) HandleGetClientByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid client id")
		return
	}
	cl, err := h.clientStore.GetClientByID(id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if cl == nil {
		utils.Error(w, http.StatusNotFound, "client not found")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"client": cl}, "", nil)
}

// HandleGetClients godoc
// @Summary      Gets all clients, or searches them
// @Description  Responds with a list of all clients. Can be filtered using a
// @Description  full-text search query, and paginated using limit and offset.
// @Tags         clients
// @Accept       json
// @Produce      json
// @Param        q      query     string        false "Full-text search query"
// @Param        limit  query     int           false "Results-per-page limit"
// @Param        offset query     int           false "Page offset for pagination"
// @Success      200    {object}  ClientsResponse
// @Failure      500    {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /clients [get]
func (h *ClientHandler) HandleGetClients(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	var (
		list []*store.Client
		err  error
	)
	if q == "" {
		list, err = h.clientStore.GetAllClients()
	} else {
		list, err = h.clientStore.SearchClientsFTS(q, limit, offset)
	}
	if err != nil {
		h.logger.Error("list/search clients", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{
		"clients": list,
	}, "", &utils.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  len(list),
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil && v >= 0 {
		return v
	}
	return def
}
