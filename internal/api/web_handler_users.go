package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

// --- Users Management (Admin) ---

func (h *WebHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r)

	users, err := h.userStore.GetAllUsers()
	if err != nil {
		h.logger.Error("listing users", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":  currentUser,
		"Users": users,
	}

	if err := h.renderer.Render(w, "users_list.html", data); err != nil {
		h.logger.Error("rendering users list", "error", err)
	}
}

func (h *WebHandler) HandleToggleUserStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r)
	targetUserID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		utils.TriggerToast(w, "ID inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Prevent self-disable
	if int64(currentUser.ID) == targetUserID {
		utils.TriggerToast(w, "No puedes deshabilitar tu propia cuenta", "error")
		http.Error(w, "Cannot disable your own account", http.StatusBadRequest)
		return
	}

	if err := h.userStore.ToggleUserStatus(targetUserID); err != nil {
		h.logger.Error("toggling user status", "error", err)
		utils.TriggerToast(w, "Error al cambiar estado del usuario", "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.TriggerToast(w, "Estado de usuario actualizado", "success")

	// Get updated user to re-render the row
	updatedUser, err := h.userStore.GetUserByID(targetUserID)
	if err != nil {
		h.logger.Error("getting updated user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := h.renderer.RenderPartial(w, "user_row.html", updatedUser); err != nil {
		h.logger.Error("rendering user row", "error", err)
	}
}
