package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type PermissionHandler struct {
	permissionSvc service.PermissionService
}

func NewPermissionHandler(permissionSvc service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permissionSvc: permissionSvc}
}

func (h *PermissionHandler) requireReadAccess(w http.ResponseWriter, r *http.Request) (*uuid.UUID, bool) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeRole, roleID)
	if err != nil {
		ServiceError(w, err)
		return nil, false
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return nil, false
	}
	return roleID, true
}

func (h *PermissionHandler) requireWriteAccess(w http.ResponseWriter, r *http.Request) (*uuid.UUID, bool) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeRole, roleID)
	if err != nil {
		ServiceError(w, err)
		return nil, false
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return nil, false
	}
	return roleID, true
}

func (h *PermissionHandler) ListByRole(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireReadAccess(w, r); !ok {
		return
	}

	roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid role id")
		return
	}

	page, err := h.permissionSvc.ListByRole(r.Context(), roleID, ParsePagination(r))
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

func (h *PermissionHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.requireWriteAccess(w, r)
	if !ok {
		return
	}

	var permission models.Permission
	if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if claims != nil {
		permission.CreatedBy = &GetClaims(r).UserID
	}

	result, err := h.permissionSvc.Create(r.Context(), &permission)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *PermissionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireReadAccess(w, r); !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	permission, err := h.permissionSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, permission)
}

func (h *PermissionHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.requireWriteAccess(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var permission models.Permission
	if err := json.NewDecoder(r.Body).Decode(&permission); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	permission.ID = id
	if claims != nil {
		userID := GetClaims(r).UserID
		permission.UpdatedBy = &userID
	}

	result, err := h.permissionSvc.Update(r.Context(), &permission)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *PermissionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireWriteAccess(w, r); !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	claims := GetClaims(r)
	if err := h.permissionSvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
