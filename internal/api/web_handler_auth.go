package api

import (
	"net/http"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/tokens"
)

type DashboardView struct {
	Date                  string // Used for input value (YYYY-MM-DD or YYYY-MM)
	ViewType              string // "day" or "month"
	LocalStats            *store.DailySalesStats
	OrderStats            *store.DailyOrderStats
	CombinedTotal         float64
	CombinedCount         int
	TopProducts           []*store.TopProduct
	TopProductsLocal      []*store.TopProduct
	TopProductsDistrib    []*store.TopProduct
}

func (h *WebHandler) HandleHome(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	viewType := r.URL.Query().Get("view")
	if viewType != "month" {
		viewType = "day"
	}

	dateStr := r.URL.Query().Get("date")
	now := time.Now()

	var start, end time.Time

	if viewType == "month" {
		// Default to current month
		targetDate := now
		if dateStr != "" {
			if d, err := time.Parse("2006-01", dateStr); err == nil {
				targetDate = d
			}
		}
		start = time.Date(targetDate.Year(), targetDate.Month(), 1, 0, 0, 0, 0, targetDate.Location())
		end = start.AddDate(0, 1, 0) // First day of next month
		// Re-format dateStr for the input
		dateStr = start.Format("2006-01")
	} else {
		// Default to today
		targetDate := now
		if dateStr != "" {
			if d, err := time.Parse("2006-01-02", dateStr); err == nil {
				targetDate = d
			}
		}
		start = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
		end = start.AddDate(0, 0, 1) // Next day
		dateStr = start.Format("2006-01-02")
	}

	// Use store directly for flexible range, assuming Service exposes a way or we bypass for dashboard
	// Since LocalSaleService currently only has GetDailyStats which takes a single date,
	// we should probably use the Store directly here OR update the service.
	// Given I updated the store interface to GetStats(start, end), let's assume we can access it via service or directly.
	// The previous step updated LocalSaleService.GetDailyStats to use store.GetStats internally but it takes a single date.
	// Let's call the store directly for the dashboard to allow custom ranges, or better, fix the service to allow ranges.
	// But to save steps, I'll cast the service's store or just call the store directly if I have access.
	// WebHandler has access to localStockService and localSaleService, but also expects to have access to stores?
	// WebHandler struct definition has `localSaleService *services.LocalSaleService`.
	// It ALSO has `localSaleStore`? No, check WebHandler struct.
	// It DOES NOT have localSaleStore in the struct based on previous `read_file` of `web_handler.go`.
	// Wait, `WebHandler` DOES NOT have `localSaleStore`. It has `localSaleService`.
	// And `localSaleService` only has `GetDailyStats(date)`.
	// I need to either update `LocalSaleService` to expose `GetStats(start, end)` OR use `GetDailyStats` strictly for days.
	// BUT I just updated `LocalSaleStore` to `GetStats(start, end)`.
	// AND I updated `LocalSaleService.GetDailyStats` to call `store.GetStats(start, end)`.
	// So I should add `GetStats(start, end)` to `LocalSaleService` as well to support the Month view properly.
	// HOWEVER, since I cannot easily edit the service again in this single turn without context switch,
	// I will assume I can call the store directly if I had it, but I don't.
	// I MUST update `LocalSaleService` to support range.
	
	// Let's try to use the `GetDailyStats` for "day" view.
	// For "month" view, I am blocked unless `LocalSaleService` supports range.
	// actually, `WebHandler` struct DOES NOT have `localSaleStore`.
	
	// WORKAROUND: I will use `localSaleService.GetDailyStats` which calls `store.GetStats`.
	// But `GetDailyStats` forces a single day range.
	// I need to fix `LocalSaleService` properly first.
	// I will initiate a `replace` on `LocalSaleService` first in the next tool call, then come back here?
	// No, I can do it in sequence.
	
	// Let's assume I will fix `LocalSaleService` to have `GetStats(start, end)` in a moment.
	// I will write the code here assuming `h.localSaleService.GetStats(start, end)` exists.

	localStats, err := h.localSaleService.GetStats(start, end)
	if err != nil {
		h.logger.Error("getting local stats", "error", err)
		localStats = &store.DailySalesStats{ByMethod: make(map[string]float64)}
	}

	var orderStats *store.DailyOrderStats
	var topProducts []*store.TopProduct
	var topProductsDistrib []*store.TopProduct

	if user.Role == "administrator" {
		orderStats, err = h.orderStore.GetStats(start, end)
		if err != nil {
			h.logger.Error("getting order stats", "error", err)
			orderStats = &store.DailyOrderStats{}
		}

		topProducts, err = h.productStore.GetTopSellingProducts(start, end)
		if err != nil {
			h.logger.Error("getting top products", "error", err)
			topProducts = []*store.TopProduct{}
		}

		topProductsDistrib, err = h.productStore.GetTopSellingProductsDistribution(start, end)
		if err != nil {
			h.logger.Error("getting top products distribution", "error", err)
			topProductsDistrib = []*store.TopProduct{}
		}
	} else {
		// Initialize empty/zero for employee
		orderStats = &store.DailyOrderStats{}
		topProducts = []*store.TopProduct{}
		topProductsDistrib = []*store.TopProduct{}
	}

	topProductsLocal, err := h.productStore.GetTopSellingProductsLocal(start, end)
	if err != nil {
		h.logger.Error("getting top products local", "error", err)
		topProductsLocal = []*store.TopProduct{}
	}

	combinedTotal := localStats.TotalAmount
	combinedCount := localStats.TotalCount

	if user.Role == "administrator" {
		combinedTotal += orderStats.TotalAmount
		combinedCount += orderStats.TotalCount
	}

	stats := DashboardView{
		Date:               dateStr,
		ViewType:           viewType,
		LocalStats:         localStats,
		OrderStats:         orderStats,
		CombinedTotal:      combinedTotal,
		CombinedCount:      combinedCount,
		TopProducts:        topProducts,
		TopProductsLocal:   topProductsLocal,
		TopProductsDistrib: topProductsDistrib,
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
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
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
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
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
