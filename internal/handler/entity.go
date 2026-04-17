package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type EntityHandler struct {
	entitySvc     service.EntityService
	compositeSvc  service.CompositeService
	permissionSvc service.PermissionService
}

func NewEntityHandler(entitySvc service.EntityService, compositeSvc service.CompositeService, permissionSvc service.PermissionService) *EntityHandler {
	return &EntityHandler{entitySvc: entitySvc, compositeSvc: compositeSvc, permissionSvc: permissionSvc}
}

func (h *EntityHandler) callerRoleID(r *http.Request) *uuid.UUID {
	if claims := GetClaims(r); claims != nil {
		return &claims.RoleID
	}
	return nil
}

// compositeForEntity fetches the entity's composite and checks read access.
func (h *EntityHandler) requireCompositeRead(w http.ResponseWriter, r *http.Request, compositeID uuid.UUID) (*models.Composite, bool) {
	detail, err := h.compositeSvc.GetByID(r.Context(), compositeID, false)
	if err != nil {
		ServiceError(w, err)
		return nil, false
	}
	composite := &detail.Composite
	ok, err := h.permissionSvc.CanRead(r.Context(), composite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return nil, false
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return nil, false
	}
	return composite, true
}

func (h *EntityHandler) requireCompositeWrite(w http.ResponseWriter, r *http.Request, compositeID uuid.UUID) (*models.Composite, bool) {
	detail, err := h.compositeSvc.GetByID(r.Context(), compositeID, false)
	if err != nil {
		ServiceError(w, err)
		return nil, false
	}
	composite := &detail.Composite
	ok, err := h.permissionSvc.CanWrite(r.Context(), composite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return nil, false
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return nil, false
	}
	return composite, true
}

func (h *EntityHandler) ListByComposite(w http.ResponseWriter, r *http.Request) {
	compositeID, err := uuid.Parse(chi.URLParam(r, "compositeID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid composite id")
		return
	}

	if _, ok := h.requireCompositeRead(w, r, compositeID); !ok {
		return
	}

	page, err := h.entitySvc.ListByComposite(r.Context(), compositeID, ParsePagination(r), ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

func (h *EntityHandler) ListChildren(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Fetch the parent entity to determine the composite for permission check.
	parent, err := h.entitySvc.GetByID(r.Context(), id, false)
	if err != nil {
		ServiceError(w, err)
		return
	}

	if _, ok := h.requireCompositeRead(w, r, parent.CompositeID); !ok {
		return
	}

	page, err := h.entitySvc.ListChildren(r.Context(), id, ParsePagination(r), ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

type createEntityRequest struct {
	CompositeID uuid.UUID  `json:"composite_id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	ParentID    *uuid.UUID `json:"parent_id"`
	AfterID     *uuid.UUID `json:"after_id"`
}

func (h *EntityHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if _, ok := h.requireCompositeWrite(w, r, req.CompositeID); !ok {
		return
	}

	entity := &models.Entity{
		CompositeID: req.CompositeID,
		Name:        req.Name,
		Slug:        req.Slug,
	}
	if claims := GetClaims(r); claims != nil {
		entity.CreatedBy = &claims.UserID
	}

	result, err := h.entitySvc.Create(r.Context(), entity, req.ParentID, req.AfterID)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *EntityHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	entity, err := h.entitySvc.GetByID(r.Context(), id, false)
	if err != nil {
		ServiceError(w, err)
		return
	}

	if _, ok := h.requireCompositeRead(w, r, entity.CompositeID); !ok {
		return
	}

	if ParseEnrich(r) {
		enriched, err := h.entitySvc.GetByID(r.Context(), id, true)
		if err != nil {
			ServiceError(w, err)
			return
		}
		JSON(w, http.StatusOK, enriched)
		return
	}

	JSON(w, http.StatusOK, entity)
}

func (h *EntityHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	compositeID, err := uuid.Parse(chi.URLParam(r, "compositeID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid composite id")
		return
	}

	if _, ok := h.requireCompositeRead(w, r, compositeID); !ok {
		return
	}

	entity, err := h.entitySvc.GetBySlug(r.Context(), compositeID, chi.URLParam(r, "slug"), ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, entity)
}

type updateEntityRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *EntityHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.entitySvc.GetByID(r.Context(), id, false)
	if err != nil {
		ServiceError(w, err)
		return
	}

	if _, ok := h.requireCompositeWrite(w, r, existing.CompositeID); !ok {
		return
	}

	var req updateEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	entity := &models.Entity{
		ID:          id,
		CompositeID: existing.CompositeID,
		Name:        req.Name,
		Slug:        req.Slug,
	}
	if claims := GetClaims(r); claims != nil {
		entity.UpdatedBy = &claims.UserID
	}

	result, err := h.entitySvc.Update(r.Context(), entity, ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *EntityHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.entitySvc.GetByID(r.Context(), id, false)
	if err != nil {
		ServiceError(w, err)
		return
	}

	if _, ok := h.requireCompositeWrite(w, r, existing.CompositeID); !ok {
		return
	}

	if err := h.entitySvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type moveRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
	AfterID  *uuid.UUID `json:"after_id"`
}

func (h *EntityHandler) Move(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.entitySvc.GetByID(r.Context(), id, false)
	if err != nil {
		ServiceError(w, err)
		return
	}

	if _, ok := h.requireCompositeWrite(w, r, existing.CompositeID); !ok {
		return
	}

	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.entitySvc.Move(r.Context(), id, req.ParentID, req.AfterID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type moveRootRequest struct {
	AfterID *uuid.UUID `json:"after_id"`
}

func (h *EntityHandler) MoveRoot(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.entitySvc.GetByID(r.Context(), id, false)
	if err != nil {
		ServiceError(w, err)
		return
	}

	if _, ok := h.requireCompositeWrite(w, r, existing.CompositeID); !ok {
		return
	}

	var req moveRootRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.entitySvc.MoveRoot(r.Context(), id, req.AfterID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
