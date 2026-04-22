package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type RoleHandler struct {
	roleSvc       service.RoleService
	permissionSvc service.PermissionService
}

func NewRoleHandler(roleSvc service.RoleService, permissionSvc service.PermissionService) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc, permissionSvc: permissionSvc}
}

func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}

	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeRole, roleID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	p := ParsePagination(r)
	page, err := h.roleSvc.List(r.Context(), &p)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}

	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeRole, roleID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var role models.Role
	if !DecodeBody(w, r, &role) {
		return
	}
	if claims != nil {
		role.CreatedBy = &claims.UserID
	}

	result, err := h.roleSvc.Create(r.Context(), &role)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *RoleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}

	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeRole, roleID)
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

	role, err := h.roleSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, role)
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}

	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeRole, roleID)
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

	var role models.Role
	if !DecodeBody(w, r, &role) {
		return
	}
	role.ID = id
	if claims != nil {
		role.UpdatedBy = &claims.UserID
	}

	result, err := h.roleSvc.Update(r.Context(), &role)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	var roleID *uuid.UUID
	if claims != nil {
		roleID = &claims.RoleID
	}

	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeRole, roleID)
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

	if err := h.roleSvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
