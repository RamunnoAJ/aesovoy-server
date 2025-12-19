package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type registerProviderRequest struct {
	Name       string `json:"name"`
	Address    string `json:"address"`
	Phone      string `json:"phone"`
	Reference  string `json:"reference"`
	Email      string `json:"email"`
	CUIT       string `json:"cuit"`
	CategoryID int64  `json:"category_id"`
}

type registerProviderCategoryRequest struct {
	Name string `json:"name"`
}

type ProviderHandler struct {
	providerStore store.ProviderStore
	logger        *slog.Logger
}

func NewProviderHandler(s store.ProviderStore, l *slog.Logger) *ProviderHandler {
	return &ProviderHandler{providerStore: s, logger: l}
}

func (h *ProviderHandler) validateRegister(req *registerProviderRequest) error {
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
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// HandleRegisterProvider godoc
// @Summary      Creates a provider
// @Description  Creates a new provider
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        body  body      registerProviderRequest  true  "Provider data"
// @Success      201   {object}  ProviderResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/providers [post]
func (h *ProviderHandler) HandleRegisterProvider(w http.ResponseWriter, r *http.Request) {
	var req registerProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if err := h.validateRegister(&req); err != nil {
		if ve, ok := err.(utils.ValidationErrors); ok {
			utils.Fail(w, http.StatusBadRequest, "validation failed", ve)
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	p := &store.Provider{
		Name: req.Name, Address: req.Address, Phone: req.Phone,
		Reference: req.Reference, Email: req.Email, CUIT: req.CUIT,
		CategoryID: req.CategoryID,
	}
	if err := h.providerStore.CreateProvider(p); err != nil {
		h.logger.Error("creating provider", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusCreated, utils.Envelope{"provider": p}, "", nil)
}

// HandleUpdateProvider godoc
// @Summary      Updates a provider
// @Description  Updates a provider's details
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        id    path      int                      true  "Provider ID"
// @Param        body  body      registerProviderRequest  true  "Provider data"
// @Success      200   {object}  ProviderResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      404   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/providers/{id} [patch]
func (h *ProviderHandler) HandleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	p, err := h.providerStore.GetProviderByID(id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if p == nil {
		utils.Error(w, http.StatusNotFound, "provider not found")
		return
	}

	var req struct {
		Name, Address, Phone, Reference, Email, CUIT *string
		CategoryID                                   *int64 `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Address != nil {
		p.Address = *req.Address
	}
	if req.Phone != nil {
		p.Phone = *req.Phone
	}
	if req.Reference != nil {
		p.Reference = *req.Reference
	}
	if req.Email != nil {
		p.Email = *req.Email
	}
	if req.CUIT != nil {
		p.CUIT = *req.CUIT
	}
	if req.CategoryID != nil {
		p.CategoryID = *req.CategoryID
	}

	if err := h.providerStore.UpdateProvider(p); err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "provider not found")
			return
		}
		h.logger.Error("updating provider", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"provider": p}, "", nil)
}

// HandleGetProviderByID godoc
// @Summary      Gets a provider
// @Description  Responds with a single provider with a given ID
// @Tags         providers
// @Produce      json
// @Param        id   path      int      true  "Provider ID"
// @Success      200  {object}  ProviderResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/providers/{id} [get]
func (h *ProviderHandler) HandleGetProviderByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid provider id")
		return
	}
	p, err := h.providerStore.GetProviderByID(id)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if p == nil {
		utils.Error(w, http.StatusNotFound, "provider not found")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"provider": p}, "", nil)
}

// HandleGetProviders godoc
// @Summary      Gets all providers, or searches them
// @Description  Responds with a list of all providers. Can be filtered using a
// @Description  full-text search query, and paginated using limit and offset.
// @Tags         providers
// @Accept       json
// @Produce      json
// @Param        q            query     string        false "Full-text search query"
// @Param        category_id  query     int           false "Category ID to filter by"
// @Param        limit        query     int           false "Results-per-page limit"
// @Param        offset       query     int           false "Page offset for pagination"
// @Success      200          {object}  ProvidersResponse
// @Failure      500          {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/providers [get]
func (h *ProviderHandler) HandleGetProviders(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)
	
	var categoryID *int64
	if cidStr := r.URL.Query().Get("category_id"); cidStr != "" {
		if cid, err := strconv.ParseInt(cidStr, 10, 64); err == nil {
			categoryID = &cid
		}
	}

	// Always use SearchProvidersFTS to handle pagination and filters consistently,
	// even if query is empty (store handles empty query optimization).
	// However, original code used GetAllProviders if q == "".
	// GetAllProviders doesn't support pagination or filtering by category easily without refactoring.
	// SearchProvidersFTS in our new implementation handles empty Q and filters.
	
	list, err := h.providerStore.SearchProvidersFTS(q, categoryID, limit, offset)

	if err != nil {
		h.logger.Error("list/search providers", "error", err)
		utils.Error(w, 500, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{
		"providers": list,
	}, "", &utils.Meta{
		Limit:  limit,
		Offset: offset,
		Total:  len(list),
	})
}

// Category Handlers

func (h *ProviderHandler) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req registerProviderCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		utils.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	pc := &store.ProviderCategory{Name: req.Name}
	if err := h.providerStore.CreateProviderCategory(pc); err != nil {
		h.logger.Error("creating provider category", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusCreated, utils.Envelope{"category": pc}, "", nil)
}

func (h *ProviderHandler) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.providerStore.GetAllProviderCategories()
	if err != nil {
		h.logger.Error("listing provider categories", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"categories": list}, "", nil)
}

func (h *ProviderHandler) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}
	var req registerProviderCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		utils.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	pc := &store.ProviderCategory{ID: id, Name: req.Name}
	if err := h.providerStore.UpdateProviderCategory(pc); err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "category not found")
			return
		}
		h.logger.Error("updating provider category", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"category": pc}, "", nil)
}

func (h *ProviderHandler) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}
	if err := h.providerStore.DeleteProviderCategory(id); err != nil {
		if err == sql.ErrNoRows {
			utils.Error(w, http.StatusNotFound, "category not found")
			return
		}
		// Check for FK violation
		if strings.Contains(err.Error(), "foreign key constraint") {
			utils.Error(w, http.StatusConflict, "cannot delete category used by providers")
			return
		}
		h.logger.Error("deleting provider category", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.OK(w, http.StatusOK, utils.Envelope{"message": "category deleted"}, "", nil)
}