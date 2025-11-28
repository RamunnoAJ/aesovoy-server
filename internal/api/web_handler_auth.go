package api

import (
	"net/http"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
)

type DashboardView struct {
	Date          string
	LocalStats    *store.DailySalesStats
	OrderStats    *store.DailyOrderStats
	CombinedTotal float64
	CombinedCount int
}

func (h *WebHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	dateStr := r.URL.Query().Get("date")
	date := time.Now()
	if dateStr != "" {
		if d, err := time.Parse("2006-01-02", dateStr); err == nil {
			date = d
		}
	}

	localStats, err := h.localSaleService.GetDailyStats(date)
	if err != nil {
		h.logger.Error("getting local daily stats", "error", err)
		localStats = &store.DailySalesStats{ByMethod: make(map[string]float64)}
	}

	orderStats, err := h.orderStore.GetDailyStats(date)
	if err != nil {
		h.logger.Error("getting order daily stats", "error", err)
		orderStats = &store.DailyOrderStats{}
	}

	stats := DashboardView{
		Date:          date.Format("2006-01-02"),
		LocalStats:    localStats,
		OrderStats:    orderStats,
		CombinedTotal: localStats.TotalAmount + orderStats.TotalAmount,
		CombinedCount: localStats.TotalCount + orderStats.TotalCount,
	}

	data := map[string]any{
		"User":  user,
		"Stats": stats,
	}

	err = h.renderer.Render(w, "home.html", data)
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
