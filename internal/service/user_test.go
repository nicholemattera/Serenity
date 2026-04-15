package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/service"
)

func TestUserService_Create_HashesPassword(t *testing.T) {
	ctx := context.Background()
	plainPassword := "super-secret"

	var savedUser *models.User
	svc := service.NewUserService(&mockUserRepo{
		create: func(_ context.Context, user *models.User) (*models.User, error) {
			savedUser = user
			return user, nil
		},
	}, bcrypt.MinCost)

	_, err := svc.Create(ctx, &models.User{
		Email:  "user@example.com",
		RoleID: uuid.New(),
	}, plainPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("password hash is not the plain text password", func(t *testing.T) {
		if savedUser.PasswordHash == plainPassword {
			t.Error("password was stored in plain text")
		}
	})

	t.Run("password hash is a valid bcrypt hash of the plain password", func(t *testing.T) {
		if err := bcrypt.CompareHashAndPassword([]byte(savedUser.PasswordHash), []byte(plainPassword)); err != nil {
			t.Errorf("stored hash does not match plain password: %v", err)
		}
	})
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{}, bcrypt.MinCost)

	_, err := svc.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, service.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
