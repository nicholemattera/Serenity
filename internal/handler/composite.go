package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type CompositeHandler struct {
	compositeSvc  service.CompositeService
	permissionSvc service.PermissionService
}

func NewCompositeHandler(compositeSvc service.CompositeService, permissionSvc service.PermissionService) *CompositeHandler {
	return &CompositeHandler{compositeSvc: compositeSvc, permissionSvc: permissionSvc}
}

func (h *CompositeHandler) callerRoleID(r *http.Request) *uuid.UUID {
	if claims := GetClaims(r); claims != nil {
		return &claims.RoleID
	}
	return nil
}

func (h *CompositeHandler) List(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeComposite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	p := ParsePagination(r)
	page, err := h.compositeSvc.List(r.Context(), &p, ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

type createCompositeRequest struct {
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	DefaultRead  bool   `json:"default_read"`
	DefaultWrite bool   `json:"default_write"`
}

func (h *CompositeHandler) Create(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeComposite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createCompositeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	composite := &models.Composite{
		Name:         req.Name,
		Slug:         req.Slug,
		DefaultRead:  req.DefaultRead,
		DefaultWrite: req.DefaultWrite,
	}
	if claims := GetClaims(r); claims != nil {
		composite.CreatedBy = &claims.UserID
	}

	result, err := h.compositeSvc.Create(r.Context(), composite)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *CompositeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeComposite, h.callerRoleID(r))
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

	composite, err := h.compositeSvc.GetByID(r.Context(), id, ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, composite)
}

func (h *CompositeHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeComposite, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	composite, err := h.compositeSvc.GetBySlug(r.Context(), chi.URLParam(r, "slug"), ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, composite)
}

func (h *CompositeHandler) Update(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeComposite, h.callerRoleID(r))
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

	var req createCompositeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	composite := &models.Composite{
		ID:           id,
		Name:         req.Name,
		Slug:         req.Slug,
		DefaultRead:  req.DefaultRead,
		DefaultWrite: req.DefaultWrite,
	}
	if claims := GetClaims(r); claims != nil {
		composite.UpdatedBy = &claims.UserID
	}

	result, err := h.compositeSvc.Update(r.Context(), composite, ParseEnrich(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *CompositeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeComposite, h.callerRoleID(r))
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

	if err := h.compositeSvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
