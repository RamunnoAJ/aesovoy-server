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

func (h *CategoryHandler) HandleGetCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.categoryStore.GetAllCategories()
	if err != nil {
		h.logger.Error("getAllCategories", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"categories": categories}, "", nil)
}

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
