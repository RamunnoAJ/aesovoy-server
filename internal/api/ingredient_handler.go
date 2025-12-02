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

type ingredientRequest struct {
	Name string `json:"name"`
}

type IngredientHandler struct {
	ingredientStore store.IngredientStore
	logger          *slog.Logger
}

func NewIngredientHandler(ingredientStore store.IngredientStore, logger *slog.Logger) *IngredientHandler {
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

// HandleCreateIngredient godoc
// @Summary      Creates an ingredient
// @Description  Creates a new ingredient with a name
// @Tags         ingredients
// @Accept       json
// @Produce      json
// @Param        body  body      ingredientRequest  true  "Ingredient data"
// @Success      201   {object}  IngredientResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/ingredients [post]
func (h *IngredientHandler) HandleCreateIngredient(w http.ResponseWriter, r *http.Request) {
	var req ingredientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding create ingredient request", "error", err)
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
		h.logger.Error("creating ingredient", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"ingredient": ingredient}, "", nil)
}

// HandleUpdateIngredient godoc
// @Summary      Updates an ingredient
// @Description  Updates an ingredient's name
// @Tags         ingredients
// @Accept       json
// @Produce      json
// @Param        id    path      int                true  "Ingredient ID"
// @Param        body  body      ingredientRequest  true  "Ingredient data"
// @Success      200   {object}  IngredientResponse
// @Failure      400   {object}  utils.HTTPError
// @Failure      404   {object}  utils.HTTPError
// @Failure      500   {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/ingredients/{id} [patch]
func (h *IngredientHandler) HandleUpdateIngredient(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("reading id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	ingredient, err := h.ingredientStore.GetIngredientByID(id)
	if err != nil {
		h.logger.Error("getting ingredient by id", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if ingredient == nil {
		utils.Error(w, http.StatusNotFound, "ingredient not found")
		return
	}

	var req ingredientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("decoding update ingredient request", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := h.validateRequest(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	ingredient.Name = req.Name
	if err := h.ingredientStore.UpdateIngredient(ingredient); err != nil {
		h.logger.Error("updating ingredient", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"ingredient": ingredient}, "", nil)
}

// HandleGetIngredientByID godoc
// @Summary      Gets an ingredient
// @Description  Responds with a single ingredient with a given ID
// @Tags         ingredients
// @Produce      json
// @Param        id   path      int      true  "Ingredient ID"
// @Success      200  {object}  IngredientResponse
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/ingredients/{id} [get]
func (h *IngredientHandler) HandleGetIngredientByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("reading id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	ingredient, err := h.ingredientStore.GetIngredientByID(id)
	if err != nil {
		h.logger.Error("getting ingredient by id", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if ingredient == nil {
		utils.Error(w, http.StatusNotFound, "ingredient not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"ingredient": ingredient}, "", nil)
}

// HandleGetAllIngredients godoc
// @Summary      Gets all ingredients
// @Description  Responds with a list of all ingredients
// @Tags         ingredients
// @Produce      json
// @Success      200  {object}  IngredientsResponse
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/ingredients [get]
func (h *IngredientHandler) HandleGetAllIngredients(w http.ResponseWriter, r *http.Request) {
	ingredients, err := h.ingredientStore.GetAllIngredients()
	if err != nil {
		h.logger.Error("getting all ingredients", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"ingredients": ingredients}, "", nil)
}

// HandleDeleteIngredient godoc
// @Summary      Deletes an ingredient
// @Description  Deletes an ingredient with a given ID
// @Tags         ingredients
// @Param        id   path      int  true  "Ingredient ID"
// @Success      204
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/ingredients/{id} [delete]
func (h *IngredientHandler) HandleDeleteIngredient(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		h.logger.Error("reading id param", "error", err)
		utils.Error(w, http.StatusBadRequest, "invalid ingredient id")
		return
	}

	err = h.ingredientStore.DeleteIngredient(id)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "ingredient not found")
		return
	}
	if err != nil {
		h.logger.Error("deleting ingredient", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
