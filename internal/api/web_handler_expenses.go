package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	h.triggerMessages(w, r)
	user := middleware.GetUser(r)
	typeStr := r.URL.Query().Get("type")
	categoryIDStr := r.URL.Query().Get("category_id")
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
	if categoryIDStr != "" {
		cid, err := strconv.ParseInt(categoryIDStr, 10, 64)
		if err == nil {
			filter.CategoryID = &cid
		}
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

	// Fetch categories for filter dropdown
	categories, err := h.expenseStore.GetAllExpenseCategories()
	if err != nil {
		h.logger.Error("fetching expense categories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	hasNext := false
	// We could optimize by fetching limit+1 in store, but currently store returns exactly limit.
	// For pagination to work perfectly we might want to count total or fetch one more.
	// Current store implementation takes limit/offset.
	// If expenses len == limit, we *might* have next page. Simple heuristic:
	if len(expenses) == limit {
		hasNext = true
	}

	data := map[string]any{
		"User":       user,
		"Expenses":   expenses,
		"Categories": categories,
		"Filter":     filter,
		"TypeParam":  typeStr,
		"Page":       page,
		"HasNext":    hasNext,
		"NextPage":   page + 1,
		"PrevPage":   page - 1,
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

	categories, err := h.expenseStore.GetAllExpenseCategories()
	if err != nil {
		h.logger.Error("getting categories for expense form", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"User":       user,
		"Providers":  providers,
		"Categories": categories,
		"Today":      time.Now().Format("2006-01-02"),
	}

	if err := h.renderer.Render(w, "expense_form.html", data); err != nil {
		h.logger.Error("rendering expense form", "error", err)
	}
}

func (h *WebHandler) HandleCreateExpense(w http.ResponseWriter, r *http.Request) {
	// Limit upload size (e.g. 10MB)
	r.ParseMultipartForm(10 << 20)

	amountStr := r.FormValue("amount")
	categoryIDStr := r.FormValue("category_id")
	typeStr := r.FormValue("type")
	dateStr := r.FormValue("date")
	providerIDStr := r.FormValue("provider_id")

	// Validation
	if amountStr == "" || categoryIDStr == "" || typeStr == "" || dateStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
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
		CategoryID: categoryID,
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
	http.Redirect(w, r, fmt.Sprintf("/expenses?type=%s&success=%s", typeStr, url.QueryEscape("Gasto registrado exitosamente")), http.StatusSeeOther)
}

func (h *WebHandler) HandleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.TriggerToast(w, "ID de gasto inválido", "error")
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.expenseStore.DeleteExpense(id); err != nil {
		h.logger.Error("deleting expense", "error", err)
		utils.TriggerToast(w, "Error al eliminar gasto", "error")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.TriggerToast(w, "Gasto eliminado", "success")
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

func (h *WebHandler) HandleQuickCreateExpenseCategory(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	category := &store.ExpenseCategory{
		Name: name,
	}

	if err := h.expenseStore.CreateExpenseCategory(category); err != nil {
		h.logger.Error("creating expense category", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	html := "<option value=\"" + strconv.FormatInt(category.ID, 10) + "\">" + category.Name + "</option>"
	w.Write([]byte(html))
}