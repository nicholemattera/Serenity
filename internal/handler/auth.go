package handler

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

type AuthHandler struct {
	authSvc service.AuthService
	userSvc service.UserService
	roleSvc service.RoleService
}

func NewAuthHandler(authSvc service.AuthService, userSvc service.UserService, roleSvc service.RoleService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userSvc: userSvc, roleSvc: roleSvc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !DecodeBody(w, r, &req) {
		return
	}

	token, err := h.authSvc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"token": token})
}

type registerRequest struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	RoleID    uuid.UUID `json:"role_id"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !DecodeBody(w, r, &req) {
		return
	}

	role, err := h.roleSvc.GetByID(r.Context(), req.RoleID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	if !role.AllowRegistration {
		Error(w, http.StatusForbidden, "registration is not allowed for this role")
		return
	}

	user := &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		RoleID:    req.RoleID,
	}

	result, err := h.userSvc.Create(r.Context(), user, req.Password)
	if err != nil {
		ServiceError(w, err)
		return
	}

	JSON(w, http.StatusCreated, result)
}
