package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
)

func (h *WebHandler) HandleShowForgotPassword(w http.ResponseWriter, r *http.Request) {
	err := h.renderer.Render(w, "forgot_password.html", nil)
	if err != nil {
		h.logger.Error("failed to render forgot password", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleSendPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	user, err := h.userStore.GetUserByEmail(email)
	if err != nil {
		h.logger.Error("getting user by email", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If user not found, we pretend we sent it to avoid enumeration attacks
	if user == nil {
		data := map[string]any{
			"Success": "Si el correo electrónico existe, recibirás un enlace para restablecer tu contraseña.",
		}
		h.renderer.Render(w, "forgot_password.html", data)
		return
	}

	token, err := tokens.GenerateToken(int(user.ID), 1*time.Hour, tokens.ScopePasswordReset)
	if err != nil {
		h.logger.Error("generating token", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.tokenStore.Insert(token)
	if err != nil {
		h.logger.Error("inserting token", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Send email
	// Construct link. Assuming localhost:8080 for now, but should be config.
	// Ideally, we should get the base URL from config or request.
	host := r.Host
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	// For development, if host is localhost:8080, scheme is http.
	// In production, it might be behind a proxy.
	link := fmt.Sprintf("%s://%s/reset-password?token=%s", scheme, host, token.Plaintext)

	data := struct {
		Link string
	}{
		Link: link,
	}

	// Run in background to not block response
	go func() {
		err = h.mailer.Send(user.Email, "password_reset.tmpl", data)
		if err != nil {
			h.logger.Error("sending password reset email", "error", err)
		}
	}()

	renderData := map[string]any{
		"Success": "Si el correo electrónico existe, recibirás un enlace para restablecer tu contraseña.",
	}
	h.renderer.Render(w, "forgot_password.html", renderData)
}

func (h *WebHandler) HandleShowResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	data := map[string]any{
		"Token": token,
	}

	h.renderer.Render(w, "reset_password.html", data)
}

func (h *WebHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	tokenPlaintext := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password != confirmPassword {
		data := map[string]any{
			"Error": "Las contraseñas no coinciden",
			"Token": tokenPlaintext,
		}
		h.renderer.Render(w, "reset_password.html", data)
		return
	}

	if len(password) < 8 {
		data := map[string]any{
			"Error": "La contraseña debe tener al menos 8 caracteres",
			"Token": tokenPlaintext,
		}
		h.renderer.Render(w, "reset_password.html", data)
		return
	}

	user, err := h.userStore.GetUserToken(tokens.ScopePasswordReset, tokenPlaintext)
	if err != nil {
		h.logger.Error("getting user by token", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		data := map[string]any{
			"Error": "El enlace es inválido o ha expirado",
			"Token": tokenPlaintext,
		}
		h.renderer.Render(w, "reset_password.html", data)
		return
	}

	err = user.PasswordHash.Set(password)
	if err != nil {
		h.logger.Error("setting password", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = h.userStore.UpdateUser(user)
	if err != nil {
		h.logger.Error("updating user", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delete all password reset tokens for this user to prevent reuse?
	// Or just this one? The store `GetUserToken` checks expiry.
	// Ideally we should consume the token (delete it).
	err = h.tokenStore.DeleteAllTokensForUser(int(user.ID), tokens.ScopePasswordReset)
	if err != nil {
		h.logger.Error("deleting tokens", "error", err)
		// Non-critical error
	}

	// Render success or redirect to login
	// Since we are using HTMX for form submission, we can redirect using HX-Redirect
	w.Header().Set("HX-Redirect", "/login?success=Clave actualizada exitosamente")
}
