package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/nicholemattera/serenity/internal/models"
	"github.com/nicholemattera/serenity/internal/repository"
)

type UserService interface {
	Create(ctx context.Context, user *models.User, plainPassword string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	List(ctx context.Context, p *repository.Pagination) (*repository.Page[models.User], error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, plainPassword string, updatedBy uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type userService struct {
	repo       repository.UserRepository
	bcryptCost int
}

func NewUserService(repo repository.UserRepository, bcryptCost int) UserService {
	return &userService{repo: repo, bcryptCost: bcryptCost}
}

func (s *userService) Create(ctx context.Context, user *models.User, plainPassword string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = string(hash)

	result, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	return result, nil
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *userService) List(ctx context.Context, p *repository.Pagination) (*repository.Page[models.User], error) {
	return s.repo.List(ctx, p)
}

func (s *userService) Update(ctx context.Context, user *models.User) (*models.User, error) {
	if _, err := s.GetByID(ctx, user.ID); err != nil {
		return nil, err
	}
	result, err := s.repo.Update(ctx, user)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *userService) UpdatePassword(ctx context.Context, id uuid.UUID, plainPassword string, updatedBy uuid.UUID) error {
	user, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = string(hash)
	user.UpdatedBy = &updatedBy
	_, err = s.repo.Update(ctx, user)
	return err
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if _, err := s.GetByID(ctx, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id, deletedBy)
}
