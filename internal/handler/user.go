package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type UserHandler struct {
	userSvc       service.UserService
	permissionSvc service.PermissionService
}

func NewUserHandler(userSvc service.UserService, permissionSvc service.PermissionService) *UserHandler {
	return &UserHandler{userSvc: userSvc, permissionSvc: permissionSvc}
}

func (h *UserHandler) callerRoleID(r *http.Request) *uuid.UUID {
	if claims := GetClaims(r); claims != nil {
		return &claims.RoleID
	}
	return nil
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeUser, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	p := ParsePagination(r)
	page, err := h.userSvc.List(r.Context(), &p)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, page)
}

type createUserRequest struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	RoleID    uuid.UUID `json:"role_id"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeUser, h.callerRoleID(r))
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !ok {
		Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var req createUserRequest
	if !DecodeBody(w, r, &req) {
		return
	}

	user := &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		RoleID:    req.RoleID,
	}
	if claims := GetClaims(r); claims != nil {
		user.CreatedBy = &claims.UserID
	}

	result, err := h.userSvc.Create(r.Context(), user, req.Password)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanReadResource(r.Context(), models.ResourceTypeUser, h.callerRoleID(r))
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

	user, err := h.userSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}
	user.PasswordHash = ""

	JSON(w, http.StatusOK, user)
}

type updateUserRequest struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	RoleID    uuid.UUID `json:"role_id"`
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeUser, h.callerRoleID(r))
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

	user, err := h.userSvc.GetByID(r.Context(), id)
	if err != nil {
		ServiceError(w, err)
		return
	}

	var req updateUserRequest
	if !DecodeBody(w, r, &req) {
		return
	}

	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Email = req.Email
	user.RoleID = req.RoleID

	if claims := GetClaims(r); claims != nil {
		user.UpdatedBy = &claims.UserID
	}

	result, err := h.userSvc.Update(r.Context(), user)
	if err != nil {
		ServiceError(w, err)
		return
	}
	result.PasswordHash = ""

	JSON(w, http.StatusOK, result)
}

type updatePasswordRequest struct {
	Password string `json:"password"`
}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Allow updating own password, or if the caller has write access to users.
	if id != claims.UserID {
		ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeUser, &claims.RoleID)
		if err != nil {
			ServiceError(w, err)
			return
		}
		if !ok {
			Error(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	var req updatePasswordRequest
	if !DecodeBody(w, r, &req) {
		return
	}

	if err := h.userSvc.UpdatePassword(r.Context(), id, req.Password, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	ok, err := h.permissionSvc.CanWriteResource(r.Context(), models.ResourceTypeUser, h.callerRoleID(r))
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

	if err := h.userSvc.Delete(r.Context(), id, claims.UserID); err != nil {
		ServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
