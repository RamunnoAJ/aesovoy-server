package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type registerCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CategoryHandler struct {
	categoryStore store.CategoryStore
	logger        *slog.Logger
}

func NewCategoryHandler(categoryStore store.CategoryStore, logger *slog.Logger) *CategoryHandler {
	return &CategoryHandler{
		categoryStore: categoryStore,
		logger:        logger,
	}
}

func (h *CategoryHandler) validateRegisterRequest(req *registerCategoryRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	return nil
}

// HandleRegisterCategory godoc
// @Summary      Creates a category
// @Description  Creates a new category with a name and a description
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        body  body      registerCategoryRequest  true  "Category data"
// @Success      201   {object}  CategoryResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /categories [post]
func (h *CategoryHandler) HandleRegisterCategory(w http.ResponseWriter, r *http.Request) {
	var req registerCategoryRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.logger.Error("decoding register request", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	err = h.validateRegisterRequest(&req)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	category := &store.Category{
		Name:        req.Name,
		Description: req.Description,
	}

	err = h.categoryStore.CreateCategory(category)
	if err != nil {
		h.logger.Error("registering category", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"category": category}, "", nil)
}

// HandleUpdateCategory godoc
// @Summary      Updates a category
// @Description  Updates a category's name or description
// @Tags         categories
// @Accept       json
// @Produce      json
// @Param        id    path      int                      true  "Category ID"
// @Param        body  body      registerCategoryRequest  true  "Category data"
// @Success      200   {object}  CategoryResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /categories/{id} [patch]
func (h *CategoryHandler) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	category, err := h.categoryStore.GetCategoryByID(categoryID)
	if err != nil {
		h.logger.Error("getCategoryByID", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	var updateCategoryRequest struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	err = json.NewDecoder(r.Body).Decode(&updateCategoryRequest)
	if err != nil {
		h.logger.Error("decodingUpdateRequest", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if updateCategoryRequest.Name != nil {
		category.Name = *updateCategoryRequest.Name
	}
	if updateCategoryRequest.Description != nil {
		category.Description = *updateCategoryRequest.Description
	}

	err = h.categoryStore.UpdateCategory(category)
	if err != nil {
		h.logger.Error("updatingCategory", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"category": category}, "", nil)
}

// HandleGetCategoryByID godoc
// @Summary      Gets a category
// @Description  Responds with a single category with a given ID
// @Tags         categories
// @Produce      json
// @Param        id   path      int      true  "Category ID"
// @Success      200  {object}  CategoryResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /categories/{id} [get]
func (h *CategoryHandler) HandleGetCategoryByID(w http.ResponseWriter, r *http.Request) {
	categoryID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam:", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	category, err := h.categoryStore.GetCategoryByID(categoryID)
	if err != nil {
		h.logger.Error("getCategoryByID", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"category": category}, "", nil)
}

// HandleGetCategories godoc
// @Summary      Gets all categories
// @Description  Responds with a list of all categories
// @Tags         categories
// @Produce      json
// @Success      200  {object}  CategoriesResponse
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /categories [get]
func (h *CategoryHandler) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categoryStore.GetAllCategories()
	if err != nil {
		h.logger.Error("getAllCategories", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"categories": categories}, "", nil)
}

// HandleDeleteCategory godoc
// @Summary      Deletes a category
// @Description  Deletes a category with a given ID
// @Tags         categories
// @Param        id   path      int  true  "Category ID"
// @Success      204
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /categories/{id} [delete]
func (h *CategoryHandler) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("readIDParam", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	err = h.categoryStore.DeleteCategory(categoryID)
	if err == sql.ErrNoRows {
		http.Error(w, "category not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "error deleting category", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
