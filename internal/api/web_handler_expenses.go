package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/middleware"
	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *WebHandler) HandleListExpenses(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	typeStr := r.URL.Query().Get("type")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	filter := store.ExpenseFilter{
		Limit:  limit,
		Offset: offset,
	}

	if typeStr != "" {
		t := store.ExpenseType(typeStr)
		filter.Type = &t
	}
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			filter.StartDate = &t
		}
	}
	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			filter.EndDate = &t
		}
	}

	expenses, err := h.expenseStore.ListExpenses(filter)
	if err != nil {
		h.logger.Error("listing expenses", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":      user,
		"Expenses":  expenses,
		"Filter":    filter,
		"TypeParam": typeStr,
		"Page":      page,
		"NextPage":  page + 1,
		"PrevPage":  page - 1,
	}

	if err := h.renderer.Render(w, "expenses_list.html", data); err != nil {
		h.logger.Error("rendering expenses list", "error", err)
	}
}

func (h *WebHandler) HandleCreateExpenseView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	providers, err := h.providerStore.GetAllProviders()
	if err != nil {
		h.logger.Error("getting providers for expense form", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":      user,
		"Providers": providers,
		"Today":     time.Now().Format("2006-01-02"),
	}

	if err := h.renderer.Render(w, "expense_form.html", data); err != nil {
		h.logger.Error("rendering expense form", "error", err)
	}
}

func (h *WebHandler) HandleCreateExpense(w http.ResponseWriter, r *http.Request) {
	// Limit upload size (e.g. 10MB)
	r.ParseMultipartForm(10 << 20)

	amountStr := r.FormValue("amount")
	category := r.FormValue("category")
	typeStr := r.FormValue("type")
	dateStr := r.FormValue("date")
	providerIDStr := r.FormValue("provider_id")

	// Validation
	if amountStr == "" || category == "" || typeStr == "" || dateStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	var providerID *int64
	if providerIDStr != "" {
		pid, err := strconv.ParseInt(providerIDStr, 10, 64)
		if err == nil && pid != 0 {
			providerID = &pid
		}
	}

	// Handle File Upload
	var imagePath string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		
		uploadDir := "uploads/expenses"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			h.logger.Error("failed to create upload directory", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		ext := filepath.Ext(header.Filename)
		filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
		destPath := filepath.Join(uploadDir, filename)
		
		dst, err := os.Create(destPath)
		if err != nil {
			h.logger.Error("failed to create file", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			h.logger.Error("failed to save file", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		imagePath = destPath
	} else if err != http.ErrMissingFile {
		h.logger.Error("file upload error", "error", err)
		http.Error(w, "File upload error", http.StatusBadRequest)
		return
	}

	expense := &store.Expense{
		Amount:     amountStr,
		Category:   category,
		Type:       store.ExpenseType(typeStr),
		Date:       date,
		ProviderID: providerID,
		ImagePath:  imagePath,
	}

	if err := h.expenseStore.CreateExpense(expense); err != nil {
		h.logger.Error("creating expense", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Redirect back to expenses list (preserving the type filter if possible)
	http.Redirect(w, r, fmt.Sprintf("/expenses?type=%s", typeStr), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.expenseStore.DeleteExpense(id); err != nil {
		h.logger.Error("deleting expense", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK) // HTMX will remove the element
}

func (h *WebHandler) HandleGetExpenseImage(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	expense, err := h.expenseStore.GetExpenseByID(id)
	if err != nil {
		h.logger.Error("getting expense", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if expense == nil || expense.ImagePath == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, expense.ImagePath)
}
