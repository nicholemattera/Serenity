package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/nicholemattera/serenity/internal/repository"
	"github.com/nicholemattera/serenity/internal/service"
)

// DecodeBody decodes the JSON request body into v.
// Returns true on success; on failure it writes the appropriate error response
// (413 for oversized bodies, 400 for malformed JSON) and returns false.
func DecodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			Error(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		Error(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

func ServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNoRowsAffected),
		errors.Is(err, service.ErrNotFound):
		Error(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		Error(w, http.StatusConflict, "conflict")
	case errors.Is(err, service.ErrUnauthorized):
		Error(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, service.ErrForbidden):
		Error(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrInvalidInput):
		Error(w, http.StatusUnprocessableEntity, err.Error())
	default:
		Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func ParsePagination(r *http.Request) repository.Pagination {
	limit := 20
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return repository.Pagination{Limit: limit, Offset: offset}
}

func ParseEnrich(r *http.Request) bool {
	return r.URL.Query().Get("enrich") == "true"
}
