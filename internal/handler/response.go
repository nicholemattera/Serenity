package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/nicholemattera/serenity/internal/repository"
	"github.com/nicholemattera/serenity/internal/service"
)

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
	case errors.Is(err, repository.ErrNoRowsAffected):
	case errors.Is(err, service.ErrNotFound):
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
