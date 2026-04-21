package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type FieldHandler struct {
	fieldSvc      service.FieldService
	permissionSvc service.PermissionService
}

func NewFieldHandler(fieldSvc service.FieldService, permissionSvc service.PermissionService) *FieldHandler {
	return &FieldHandler{fieldSvc: fieldSvc, permissionSvc: permissionSvc}
}

func (h *FieldHandler) callerRoleID(r *http.Request) *uuid.UUID {
	if claims := GetClaims(r); claims != nil {
		return &claims.RoleID
	}
	return nil
}

func (h *FieldHandler) ListByComposite(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeField, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	compositeID, err := uuid.Parse(chi.URLParam(r, "compositeID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid composite id")
		return
	}

	p := ParsePagination(r)
	page, err := h.fieldSvc.ListByComposite(r.Context(), compositeID, &p)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

type createFieldRequest struct {
	CompositeID  uuid.UUID        `json:"composite_id"`
	Name         string           `json:"name"`
	Slug         string           `json:"slug"`
	Type         models.FieldType `json:"type"`
	Required     bool             `json:"required"`
	Position     int              `json:"position"`
	DefaultValue *string          `json:"default_value"`
	Metadata     json.RawMessage  `json:"metadata"`
}

func (h *FieldHandler) Create(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeField, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	field := &models.Field{
		CompositeID:  req.CompositeID,
		Name:         req.Name,
		Slug:         req.Slug,
		Type:         req.Type,
		Required:     req.Required,
		Position:     req.Position,
		DefaultValue: req.DefaultValue,
		Metadata:     req.Metadata,
	}
	if claims := GetClaims(r); claims != nil {
		field.CreatedBy = &claims.UserID
	}

	result, err := h.fieldSvc.Create(r.Context(), field)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *FieldHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeField, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	field, err := h.fieldSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, field)
}

func (h *FieldHandler) Update(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeField, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	field, err := h.fieldSvc.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusNotFound, "not found")
		return
	}

	var req createFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	field.Name = req.Name
	field.Slug = req.Slug
	field.Type = req.Type
	field.Required = req.Required
	field.Position = req.Position
	field.DefaultValue = req.DefaultValue
	field.Metadata = req.Metadata

	if claims := GetClaims(r); claims != nil {
		field.UpdatedBy = &claims.UserID
	}

	result, err := h.fieldSvc.Update(r.Context(), field)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *FieldHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeField, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.fieldSvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
