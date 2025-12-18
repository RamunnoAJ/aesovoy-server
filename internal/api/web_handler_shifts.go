package api

import (
	"net/http"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
)

func (h *WebHandler) HandleShiftManagement(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	
	currentShift, err := h.shiftService.GetCurrentShift(user.ID)
	if err != nil {
		h.logger.Error("getting current shift", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	shifts, err := h.shiftService.ListUserShifts(user.ID, page)
	if err != nil {
		h.logger.Error("listing user shifts", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":         user,
		"CurrentShift": currentShift,
		"Shifts":       shifts,
		"Page":         page,
		"NextPage":     page + 1,
		"PrevPage":     page - 1,
	}

	if err := h.renderer.Render(w, "shifts.html", data); err != nil {
		h.logger.Error("rendering shifts view", "error", err)
	}
}

func (h *WebHandler) HandleOpenShift(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	startCash, _ := strconv.ParseFloat(r.FormValue("start_cash"), 64)
	notes := r.FormValue("notes")

	_, err := h.shiftService.OpenShift(user.ID, startCash, notes)
	if err != nil {
		h.logger.Error("opening shift", "error", err)
		// Return error as toast via HX-Trigger if HTMX, or redirect with error param
		http.Redirect(w, r, "/shifts?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/shifts", http.StatusSeeOther)
}

func (h *WebHandler) HandleCloseShift(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	declaredCash, _ := strconv.ParseFloat(r.FormValue("end_cash_declared"), 64)
	notes := r.FormValue("notes")

	_, err := h.shiftService.CloseShift(user.ID, declaredCash, notes)
	if err != nil {
		h.logger.Error("closing shift", "error", err)
		http.Redirect(w, r, "/shifts?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/shifts", http.StatusSeeOther)
}
