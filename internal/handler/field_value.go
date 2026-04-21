package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type FieldValueHandler struct {
	fieldValueSvc service.FieldValueService
	entitySvc     service.EntityService
	compositeSvc  service.CompositeService
	permissionSvc service.PermissionService
}

func NewFieldValueHandler(fieldValueSvc service.FieldValueService, entitySvc service.EntityService, compositeSvc service.CompositeService, permissionSvc service.PermissionService) *FieldValueHandler {
	return &FieldValueHandler{
		fieldValueSvc: fieldValueSvc,
		entitySvc:     entitySvc,
		compositeSvc:  compositeSvc,
		permissionSvc: permissionSvc,
	}
}

func (h *FieldValueHandler) callerRoleID(r *http.Request) *uuid.UUID {
	if claims := GetClaims(r); claims != nil {
		return &claims.RoleID
	}
	return nil
}

func (h *FieldValueHandler) compositeForEntity(r *http.Request, entityID uuid.UUID) (*models.Composite, error) {
	entity, err := h.entitySvc.GetByID(r.Context(), entityID, false)
	if err != nil {
		return nil, err
	}
	detail, err := h.compositeSvc.GetByID(r.Context(), entity.CompositeID, false)
	if err != nil {
		return nil, err
	}
	return &detail.Composite, nil
}

func (h *FieldValueHandler) ListByEntity(w http.ResponseWriter, r *http.Request) {
	entityID, err := uuid.Parse(chi.URLParam(r, "entityID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid entity id")
		return
	}

	composite, err := h.compositeForEntity(r, entityID)
	if err != nil {
		ServiceError(w, err)
		return
	}

	ok, err := h.permissionSvc.CanRead(r.Context(), composite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	p := ParsePagination(r)
	page, err := h.fieldValueSvc.ListByEntity(r.Context(), entityID, &p)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

type setFieldValueRequest struct {
	EntityID uuid.UUID `json:"entity_id"`
	FieldID  uuid.UUID `json:"field_id"`
	Value    string    `json:"value"`
}

func (h *FieldValueHandler) Set(w http.ResponseWriter, r *http.Request) {
	var req setFieldValueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	composite, err := h.compositeForEntity(r, req.EntityID)
	if err != nil {
		ServiceError(w, err)
		return
	}

	ok, err := h.permissionSvc.CanWrite(r.Context(), composite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	fv := &models.FieldValue{
		EntityID: req.EntityID,
		FieldID:  req.FieldID,
		Value:    req.Value,
	}
	if claims := GetClaims(r); claims != nil {
		fv.CreatedBy = &claims.UserID
		fv.UpdatedBy = &claims.UserID
	}

	result, err := h.fieldValueSvc.Set(r.Context(), fv)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *FieldValueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	fv, err := h.fieldValueSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}

	composite, err := h.compositeForEntity(r, fv.EntityID)
	if err != nil {
		ServiceError(w, err)
		return
	}

	ok, err := h.permissionSvc.CanRead(r.Context(), composite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	JSON(w, http.StatusOK, fv)
}

func (h *FieldValueHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	fv, err := h.fieldValueSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}

	composite, err := h.compositeForEntity(r, fv.EntityID)
	if err != nil {
		ServiceError(w, err)
		return
	}

	ok, err := h.permissionSvc.CanWrite(r.Context(), composite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	claims := GetClaims(r)
	if err := h.fieldValueSvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
