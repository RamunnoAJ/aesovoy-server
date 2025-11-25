package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
)

type ingredientRequest struct {
	Name string `json:"name"`
}

type IngredientHandler struct {
	ingredientStore store.IngredientStore
	logger          *log.Logger
}

func NewIngredientHandler(ingredientStore store.IngredientStore, logger *log.Logger) *IngredientHandler {
	return &IngredientHandler{
		ingredientStore: ingredientStore,
		logger:          logger,
	}
}

func (h *IngredientHandler) validateRequest(req *ingredientRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *IngredientHandler) HandleCreateIngredient(w http.ResponseWriter, r *http.Request) {
	var req ingredientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("ERROR: decoding create ingredient request: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.validateRequest(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ingredient := &store.Ingredient{
		Name: req.Name,
	}

	if err := h.ingredientStore.CreateIngredient(ingredient); err != nil {
		h.logger.Printf("ERROR: creating ingredient: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"ingredient": ingredient}, "", nil)
}

func (h *IngredientHandler) HandleUpdateIngredient(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Printf("ERROR: reading id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	ingredient, err := h.ingredientStore.GetIngredientByID(id)
	if err != nil {
		h.logger.Printf("ERROR: getting ingredient by id: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if ingredient == nil {
		utils.Error(w, http.StatusNotFound, "ingredient not found")
		return
	}

	var req ingredientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Printf("ERROR: decoding update ingredient request: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.validateRequest(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ingredient.Name = req.Name
	if err := h.ingredientStore.UpdateIngredient(ingredient); err != nil {
		h.logger.Printf("ERROR: updating ingredient: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"ingredient": ingredient}, "", nil)
}

func (h *IngredientHandler) HandleGetIngredientByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Printf("ERROR: reading id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	ingredient, err := h.ingredientStore.GetIngredientByID(id)
	if err != nil {
		h.logger.Printf("ERROR: getting ingredient by id: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if ingredient == nil {
		utils.Error(w, http.StatusNotFound, "ingredient not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"ingredient": ingredient}, "", nil)
}

func (h *IngredientHandler) HandleGetAllIngredients(w http.ResponseWriter, r *http.Request) {
	ingredients, err := h.ingredientStore.GetAllIngredients()
	if err != nil {
		h.logger.Printf("ERROR: getting all ingredients: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"ingredients": ingredients}, "", nil)
}

func (h *IngredientHandler) HandleDeleteIngredient(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Printf("ERROR: reading id param: %v", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	err = h.ingredientStore.DeleteIngredient(id)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "ingredient not found")
		return
	}
	if err != nil {
		h.logger.Printf("ERROR: deleting ingredient: %v", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
