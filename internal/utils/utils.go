package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
type ValidationErrors []FieldError

type Envelope map[string]any

type Meta struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
	Total  int `json:"total,omitempty"`
}

type APIResponse struct {
	Error      bool         `json:"error"`
	Message    string       `json:"message,omitempty"`
	StatusCode int          `json:"statusCode"`
	Data       any          `json:"data,omitempty"`
	Errors     []FieldError `json:"errors,omitempty"`
	Meta       *Meta        `json:"meta,omitempty"`
}

func ReadIDParam(r *http.Request) (int64, error) {
	idParam := chi.URLParam(r, "id")
	if idParam == "" {
		return 0, errors.New("invalid id parameter")
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		return 0, errors.New("invalid id parameter type")
	}

	return id, nil
}

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range v {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(e.Field + ": " + e.Message)
	}
	return b.String()
}

func write(w http.ResponseWriter, status int, body APIResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	js, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(APIResponse{
			Error:      true,
			Message:    "failed to encode response",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	w.WriteHeader(status)
	js = append(js, '\n')
	_, _ = w.Write(js)
}

func OK(w http.ResponseWriter, status int, data any, msg string, meta *Meta) {
	write(w, status, APIResponse{
		Error:      false,
		Message:    msg,
		StatusCode: status,
		Data:       data,
		Meta:       meta,
	})
}

func Fail(w http.ResponseWriter, status int, msg string, errs []FieldError) {
	write(w, status, APIResponse{
		Error:      true,
		Message:    msg,
		StatusCode: status,
		Errors:     errs,
	})
}

func Error(w http.ResponseWriter, status int, msg string) {
	Fail(w, status, msg, nil)
}

func Tern(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
