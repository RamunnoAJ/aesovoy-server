package api

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

// --- Ingredients ---

func (h *WebHandler) HandleListIngredients(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	ingredients, err := h.ingredientStore.GetAllIngredients()
	if err != nil {
		h.logger.Error("listing ingredients", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":        user,
		"Ingredients": ingredients,
	}

	if err := h.renderer.Render(w, "ingredients_list.html", data); err != nil {
		h.logger.Error("rendering ingredients list", "error", err)
	}
}

func (h *WebHandler) HandleCreateIngredientView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User":       user,
		"Ingredient": store.Ingredient{},
	}

	if err := h.renderer.Render(w, "ingredient_form.html", data); err != nil {
		h.logger.Error("rendering ingredient form", "error", err)
	}
}

func (h *WebHandler) HandleCreateIngredient(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ingredient := &store.Ingredient{
		Name: r.FormValue("name"),
	}

	if err := h.ingredientStore.CreateIngredient(ingredient); err != nil {
		h.logger.Error("creating ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/ingredients?success="+url.QueryEscape("Ingrediente creado exitosamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleEditIngredientView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ingredient, err := h.ingredientStore.GetIngredientByID(ingredientID)
	if err != nil {
		h.logger.Error("getting ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if ingredient == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":       user,
		"Ingredient": ingredient,
	}

	if err := h.renderer.Render(w, "ingredient_form.html", data); err != nil {
		h.logger.Error("rendering ingredient form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateIngredient(w http.ResponseWriter, r *http.Request) {
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ingredient := &store.Ingredient{
		ID:   ingredientID,
		Name: r.FormValue("name"),
	}

	if err := h.ingredientStore.UpdateIngredient(ingredient); err != nil {
		h.logger.Error("updating ingredient", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/ingredients?success="+url.QueryEscape("Ingrediente actualizado correctamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteIngredient(w http.ResponseWriter, r *http.Request) {
	ingredientID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID de ingrediente inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.ingredientStore.DeleteIngredient(ingredientID); err != nil {
		h.logger.Error("deleting ingredient", "error", err)
		utils.TriggerToast(w, "Error al eliminar ingrediente", "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.TriggerToast(w, "Ingrediente eliminado", "success")
	w.WriteHeader(http.StatusOK)
}
