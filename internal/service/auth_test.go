package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

func hashedPassword(t *testing.T, plain string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	return string(hash)
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	roleID := uuid.New()

	validUser := &models.User{
		ID:           userID,
		Email:        "user@example.com",
		PasswordHash: hashedPassword(t, "correct-password"),
		RoleID:       roleID,
	}
	validRole := &models.Role{
		ID:             roleID,
		SessionTimeout: 3600,
	}

	newAuth := func(user *models.User, role *models.Role) service.AuthService {
		return service.NewAuthService(
			&mockUserRepo{
				getByEmail: func(_ context.Context, email string) (*models.User, error) {
					if user != nil && email == user.Email {
						return user, nil
					}
					return nil, pgx.ErrNoRows
				},
			},
			&mockRoleRepo{
				getByID: func(_ context.Context, id uuid.UUID) (*models.Role, error) {
					if role != nil && id == role.ID {
						return role, nil
					}
					return nil, pgx.ErrNoRows
				},
			},
			"test-secret",
			"serenity",
			"serenity",
		)
	}

	t.Run("returns token on valid credentials", func(t *testing.T) {
		svc := newAuth(validUser, validRole)
		token, err := svc.Login(ctx, "user@example.com", "correct-password")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token == "" {
			t.Error("expected non-empty token")
		}
	})

	t.Run("returns ErrUnauthorized for unknown email", func(t *testing.T) {
		svc := newAuth(validUser, validRole)
		_, err := svc.Login(ctx, "unknown@example.com", "correct-password")
		if !errors.Is(err, service.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("returns ErrUnauthorized for wrong password", func(t *testing.T) {
		svc := newAuth(validUser, validRole)
		_, err := svc.Login(ctx, "user@example.com", "wrong-password")
		if !errors.Is(err, service.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()
	userID := uuid.New()

	user := &models.User{
		ID:           userID,
		Email:        "user@example.com",
		PasswordHash: hashedPassword(t, "password"),
		RoleID:       roleID,
	}
	role := &models.Role{ID: roleID, SessionTimeout: 3600}

	svc := service.NewAuthService(
		&mockUserRepo{
			getByEmail: func(_ context.Context, _ string) (*models.User, error) { return user, nil },
		},
		&mockRoleRepo{
			getByID: func(_ context.Context, _ uuid.UUID) (*models.Role, error) { return role, nil },
		},
		"test-secret",
		"serenity",
		"serenity",
	)

	t.Run("valid token returns correct claims", func(t *testing.T) {
		token, err := svc.Login(ctx, user.Email, "password")
		if err != nil {
			t.Fatalf("login failed: %v", err)
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("expected UserID %v, got %v", userID, claims.UserID)
		}
		if claims.RoleID != roleID {
			t.Errorf("expected RoleID %v, got %v", roleID, claims.RoleID)
		}
	})

	t.Run("tampered token returns ErrUnauthorized", func(t *testing.T) {
		token, _ := svc.Login(ctx, user.Email, "password")
		_, err := svc.ValidateToken(token + "tampered")
		if !errors.Is(err, service.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("token signed with different secret returns ErrUnauthorized", func(t *testing.T) {
		otherSvc := service.NewAuthService(
			&mockUserRepo{
				getByEmail: func(_ context.Context, _ string) (*models.User, error) { return user, nil },
			},
			&mockRoleRepo{
				getByID: func(_ context.Context, _ uuid.UUID) (*models.Role, error) { return role, nil },
			},
			"different-secret",
			"serenity",
			"serenity",
		)
		token, _ := otherSvc.Login(ctx, user.Email, "password")
		_, err := svc.ValidateToken(token)
		if !errors.Is(err, service.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("garbage string returns ErrUnauthorized", func(t *testing.T) {
		_, err := svc.ValidateToken("not.a.token")
		if !errors.Is(err, service.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})
}
