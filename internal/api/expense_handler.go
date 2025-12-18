package api

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/RamunnoAJ/aesovoy-server/internal/utils"
	"github.com/google/uuid"
)

type ExpenseHandler struct {
	expenseStore store.ExpenseStore
	logger       *slog.Logger
	uploadDir    string
}

func NewExpenseHandler(expenseStore store.ExpenseStore, logger *slog.Logger) *ExpenseHandler {
	// Ensure upload directory exists
	// Ideally this should be configurable, but hardcoding relative path for now as per constraints
	uploadDir := "uploads/expenses"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		logger.Error("failed to create upload directory", "error", err)
	}
	return &ExpenseHandler{
		expenseStore: expenseStore,
		logger:       logger,
		uploadDir:    uploadDir,
	}
}

// HandleCreateExpense godoc
// @Summary      Creates an expense
// @Description  Creates a new expense with optional receipt image
// @Tags         expenses
// @Accept       multipart/form-data
// @Produce      json
// @Param        amount       formData  string  true  "Amount"
// @Param        category     formData  string  true  "Category"
// @Param        type         formData  string  true  "Type (local/production)"
// @Param        date         formData  string  true  "Date (YYYY-MM-DD)"
// @Param        provider_id  formData  int     false "Provider ID"
// @Param        image        formData  file    false "Receipt Image"
// @Success      201          {object}  store.Expense
// @Failure      400          {object}  utils.HTTPError
// @Failure      500          {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/expenses [post]
func (h *ExpenseHandler) HandleCreateExpense(w http.ResponseWriter, r *http.Request) {
	// Limit upload size (e.g. 10MB)
	r.ParseMultipartForm(10 << 20)

	amountStr := r.FormValue("amount")
	category := r.FormValue("category")
	typeStr := r.FormValue("type")
	dateStr := r.FormValue("date")
	providerIDStr := r.FormValue("provider_id")

	// Validation
	if amountStr == "" || category == "" || typeStr == "" || dateStr == "" {
		utils.Error(w, http.StatusBadRequest, "missing required fields")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid date format (YYYY-MM-DD)")
		return
	}

	var providerID *int64
	if providerIDStr != "" {
		pid, err := strconv.ParseInt(providerIDStr, 10, 64)
		if err == nil {
			providerID = &pid
		}
	}

	// Handle File Upload
	var imagePath string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		
		ext := filepath.Ext(header.Filename)
		filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
		destPath := filepath.Join(h.uploadDir, filename)
		
		dst, err := os.Create(destPath)
		if err != nil {
			h.logger.Error("failed to create file", "error", err)
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			h.logger.Error("failed to save file", "error", err)
			utils.Error(w, http.StatusInternalServerError, "internal server error")
			return
		}
		imagePath = destPath
	} else if err != http.ErrMissingFile {
		h.logger.Error("file upload error", "error", err)
		utils.Error(w, http.StatusBadRequest, "file upload error")
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
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusCreated, utils.Envelope{"expense": expense}, "", nil)
}

// HandleGetExpenses godoc
// @Summary      Gets expenses
// @Description  Responds with a list of expenses, filtered by type or date
// @Tags         expenses
// @Produce      json
// @Param        type       query     string  false "Type (local/production)"
// @Param        start_date query     string  false "Start Date (YYYY-MM-DD)"
// @Param        end_date   query     string  false "End Date (YYYY-MM-DD)"
// @Param        limit      query     int     false "Limit"
// @Param        offset     query     int     false "Offset"
// @Success      200        {object}  []store.Expense
// @Failure      500        {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/expenses [get]
func (h *ExpenseHandler) HandleGetExpenses(w http.ResponseWriter, r *http.Request) {
	typeStr := r.URL.Query().Get("type")
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

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
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"expenses": expenses}, "", nil)
}

// HandleGetExpenseByID godoc
// @Summary      Gets an expense
// @Description  Responds with a single expense
// @Tags         expenses
// @Produce      json
// @Param        id   path      int      true  "Expense ID"
// @Success      200  {object}  store.Expense
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/expenses/{id} [get]
func (h *ExpenseHandler) HandleGetExpenseByID(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	expense, err := h.expenseStore.GetExpenseByID(id)
	if err != nil {
		h.logger.Error("getting expense", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if expense == nil {
		utils.Error(w, http.StatusNotFound, "expense not found")
		return
	}

	utils.OK(w, http.StatusOK, utils.Envelope{"expense": expense}, "", nil)
}

// HandleDeleteExpense godoc
// @Summary      Deletes an expense
// @Description  Soft deletes an expense
// @Tags         expenses
// @Param        id   path      int      true  "Expense ID"
// @Success      204
// @Failure      400  {object}  utils.HTTPError
// @Failure      404  {object}  utils.HTTPError
// @Failure      500  {object}  utils.HTTPError
// @Security     BearerAuth
// @Router       /api/v1/expenses/{id} [delete]
func (h *ExpenseHandler) HandleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := utils.ReadIDParam(r)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	err = h.expenseStore.DeleteExpense(id)
	if err == sql.ErrNoRows {
		utils.Error(w, http.StatusNotFound, "expense not found")
		return
	}
	if err != nil {
		h.logger.Error("deleting expense", "error", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
