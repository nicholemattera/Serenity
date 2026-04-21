package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/nicholemattera/serenity/internal/models"
)

func TestAuth_Login(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	role, err := srv.roleSvc.Create(ctx, &models.Role{
		Name:           "admin",
		HierarchyLevel: 1,
		SessionTimeout: 3600,
	})
	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}

	_, err = srv.userSvc.Create(ctx, &models.User{
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane@example.com",
		RoleID:    role.ID,
	}, "password123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	t.Run("valid credentials returns token", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/auth/login", map[string]string{
			"email":    "jane@example.com",
			"password": "password123",
		}, "")
		assertStatus(t, rr, http.StatusOK)

		var resp map[string]string
		decode(t, rr, &resp)
		if resp["token"] == "" {
			t.Error("expected non-empty token")
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/auth/login", map[string]string{
			"email":    "jane@example.com",
			"password": "wrong",
		}, "")
		assertStatus(t, rr, http.StatusUnauthorized)
	})

	t.Run("unknown email returns 401", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/auth/login", map[string]string{
			"email":    "nobody@example.com",
			"password": "password123",
		}, "")
		assertStatus(t, rr, http.StatusUnauthorized)
	})
}

func TestAuth_Register(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	openRole, err := srv.roleSvc.Create(ctx, &models.Role{
		Name:              "member",
		HierarchyLevel:    10,
		SessionTimeout:    3600,
		AllowRegistration: true,
	})
	if err != nil {
		t.Fatalf("failed to create open role: %v", err)
	}

	closedRole, err := srv.roleSvc.Create(ctx, &models.Role{
		Name:              "admin",
		HierarchyLevel:    1,
		SessionTimeout:    3600,
		AllowRegistration: false,
	})
	if err != nil {
		t.Fatalf("failed to create closed role: %v", err)
	}

	t.Run("registers successfully with open role", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/auth/register", map[string]any{
			"first_name": "Alice",
			"last_name":  "Smith",
			"email":      "alice@example.com",
			"password":   "secure123",
			"role_id":    openRole.ID,
		}, "")
		assertStatus(t, rr, http.StatusCreated)

		var user map[string]any
		decode(t, rr, &user)
		if user["email"] != "alice@example.com" {
			t.Errorf("expected email alice@example.com, got %v", user["email"])
		}
	})

	t.Run("registration blocked for closed role", func(t *testing.T) {
		rr := srv.do(http.MethodPost, "/v1/auth/register", map[string]any{
			"first_name": "Bob",
			"last_name":  "Smith",
			"email":      "bob@example.com",
			"password":   "secure123",
			"role_id":    closedRole.ID,
		}, "")
		assertStatus(t, rr, http.StatusForbidden)
	})
}
