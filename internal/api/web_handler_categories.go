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

// --- Categories ---

func (h *WebHandler) HandleListCategories(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	categories, err := h.categoryStore.GetAllCategories()
	if err != nil {
		h.logger.Error("listing categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Categories": categories,
	}

	if err := h.renderer.Render(w, "categories_list.html", data); err != nil {
		h.logger.Error("rendering categories list", "error", err)
	}
}

func (h *WebHandler) HandleCreateCategoryView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User":     user,
		"Category": store.Category{},
	}

	if err := h.renderer.Render(w, "category_form.html", data); err != nil {
		h.logger.Error("rendering category form", "error", err)
	}
}

func (h *WebHandler) HandleCreateCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.Category{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	if err := h.categoryStore.CreateCategory(category); err != nil {
		h.logger.Error("creating category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/categories?success="+url.QueryEscape("Categoría creada exitosamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleQuickCreateCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.Category{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	if err := h.categoryStore.CreateCategory(category); err != nil {
		h.logger.Error("creating category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	// Return the new option tag
	// We assume the ID is populated by the store (pointer)
	// If the store doesn't populate ID on create, we might need to fetch it or ensure the store does.
	// Checking postgres_category_store.go might be needed.
	// Assuming Postgres RETURNING id works.
	
	// Create a simple template or just write string
	html := "<option value=\"" + strconv.FormatInt(category.ID, 10) + "\" selected>" + category.Name + "</option>"
	w.Write([]byte(html))
}

func (h *WebHandler) HandleEditCategoryView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	category, err := h.categoryStore.GetCategoryByID(categoryID)
	if err != nil {
		h.logger.Error("getting category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if category == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":     user,
		"Category": category,
	}

	if err := h.renderer.Render(w, "category_form.html", data); err != nil {
		h.logger.Error("rendering category form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	category := &store.Category{
		ID:          categoryID,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
	}

	if err := h.categoryStore.UpdateCategory(category); err != nil {
		h.logger.Error("updating category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/categories?success="+url.QueryEscape("Categoría actualizada correctamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID de categoría inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.categoryStore.DeleteCategory(categoryID); err != nil {
		h.logger.Error("deleting category", "error", err)
		utils.TriggerToast(w, "Error al eliminar categoría", "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.TriggerToast(w, "Categoría eliminada", "success")
	w.WriteHeader(http.StatusOK)
}
