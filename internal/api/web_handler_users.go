package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	chi "github.com/go-chi/chi/v5"
)

// --- Users Management (Admin) ---

func (h *WebHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	h.triggerMessages(w, r)
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

func (h *WebHandler) HandleCreateUserView(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r)
	data := map[string]any{
		"User":       currentUser,
		"TargetUser": store.User{},
	}

	if err := h.renderer.Render(w, "user_form.html", data); err != nil {
		h.logger.Error("rendering user form", "error", err)
	}
}

func (h *WebHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	user := &store.User{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Role:     r.FormValue("role"),
		IsActive: true,
	}

	password := r.FormValue("password")
	if len(password) < 8 {
		utils.TriggerToast(w, "La contraseña debe tener al menos 8 caracteres", "error")
		h.HandleCreateUserView(w, r) // Re-render form? Or redirect with error? Redirect is better to avoid parsing issues.
		// Actually re-rendering with error context is best but for now redirect with error query param is consistent with other handlers.
		// But other handlers in this file (products) use redirect with error param.
		return
	}

	if err := user.PasswordHash.Set(password); err != nil {
		h.logger.Error("hashing password", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := h.userStore.CreateUser(user); err != nil {
		h.logger.Error("creating user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/users?success="+url.QueryEscape("Usuario creado exitosamente"), http.StatusSeeOther)
}

func (h *WebHandler) HandleEditUserView(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetUser(r)
	targetUserID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	targetUser, err := h.userStore.GetUserByID(targetUserID)
	if err != nil {
		h.logger.Error("getting user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if targetUser == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"User":       currentUser,
		"TargetUser": targetUser,
	}

	if err := h.renderer.Render(w, "user_form.html", data); err != nil {
		h.logger.Error("rendering user form", "error", err)
	}
}

func (h *WebHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	existingUser, err := h.userStore.GetUserByID(targetUserID)
	if err != nil {
		h.logger.Error("getting user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	existingUser.Username = r.FormValue("username")
	existingUser.Email = r.FormValue("email")
	existingUser.Role = r.FormValue("role")

	newPassword := r.FormValue("password")
	if newPassword != "" {
		if len(newPassword) < 8 {
			utils.TriggerToast(w, "La contraseña debe tener al menos 8 caracteres", "error")
			// Redirect back to edit with error?
			http.Redirect(w, r, fmt.Sprintf("/users/%d/edit?error=%s", targetUserID, url.QueryEscape("Contraseña muy corta")), http.StatusSeeOther)
			return
		}
		if err := existingUser.PasswordHash.Set(newPassword); err != nil {
			h.logger.Error("hashing password", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	if err := h.userStore.UpdateUser(existingUser); err != nil {
		h.logger.Error("updating user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/users?success="+url.QueryEscape("Usuario actualizado exitosamente"), http.StatusSeeOther)
}
