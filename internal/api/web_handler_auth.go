package api

import (
	"net/http"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
)

func (h *WebHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	data := map[string]any{
		"User": user,
	}

	err := h.renderer.Render(w, "home.html", data)
	if err != nil {
		h.logger.Error("failed to render home", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})

	if r.Header.Get("HX-Request") != "" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *WebHandler) HandleTime(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(time.Now().Format(time.RFC1123)))
}

func (h *WebHandler) HandleShowLogin(w http.ResponseWriter, r *http.Request) {
	err := h.renderer.Render(w, "login.html", nil)
	if err != nil {
		h.logger.Error("failed to render login", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *WebHandler) HandleWebLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.logger.Error("parsing form", "error", err)
		h.renderLoginError(w, "Invalid request")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.userStore.GetUserByUsername(username)
	if err != nil {
		h.logger.Error("getting user by username", "error", err)
		h.renderLoginError(w, "Credenciales incorrectas")
		return
	}

	if user == nil {
		h.renderLoginError(w, "Credenciales incorrectas")
		return
	}

	match, err := user.PasswordHash.Matches(password)
	if err != nil {
		h.logger.Error("matching password", "error", err)
		h.renderLoginError(w, "Error interno del servidor")
		return
	}

	if !match {
		h.renderLoginError(w, "Credenciales incorrectas")
		return
	}

	token, err := tokens.GenerateToken(int(user.ID), 24*time.Hour, tokens.ScopeAuth)
	if err != nil {
		h.logger.Error("generating token", "error", err)
		h.renderLoginError(w, "Internal server error")
		return
	}

	err = h.tokenStore.Insert(token)
	if err != nil {
		h.logger.Error("inserting token", "error", err)
		h.renderLoginError(w, "Internal server error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token.Plaintext,
		Expires:  token.Expiry,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})

	w.Header().Set("HX-Redirect", "/")
}

func (h *WebHandler) renderLoginError(w http.ResponseWriter, msg string) {
	data := map[string]any{
		"Error": msg,
	}
	// When using HTMX with hx-swap="outerHTML", we want to re-render the form (login.html)
	// But if we just render "login.html", it might render with the base layout if using hx-boost or if it's a full page load.
	// However, our login.html template defines "content".
	// The renderer currently executes "base.html" which includes "content".
	// If this is an HTMX request targeting the form, we might just want the form HTML.
	// BUT, our renderer is simple and always wraps in base.
	// For now, let's just re-render the whole page. HTMX is smart enough to extract if needed,
	// OR we can just let it swap the body.
	// The login form has hx-target="body" hx-swap="outerHTML".
	// If we return the full page, the <body> of the response will replace the <body> of the current page.
	err := h.renderer.Render(w, "login.html", data)
	if err != nil {
		h.logger.Error("failed to render login error", "error", err)
	}
}
