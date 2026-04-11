package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nicholemattera/serenity/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	List(ctx context.Context, p Pagination) (*Page[models.User], error)
	Update(ctx context.Context, user *models.User) (*models.User, error)
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
}

type userRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.New()
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.Exec(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role_id, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, user.ID, user.FirstName, user.LastName, user.Email, user.PasswordHash, user.RoleID,
		user.CreatedAt, user.UpdatedAt, user.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, first_name, last_name, email, password_hash, role_id,
		       created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.PasswordHash, &user.RoleID,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&user.CreatedBy, &user.UpdatedBy, &user.DeletedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, first_name, last_name, email, password_hash, role_id,
		       created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`, email).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.PasswordHash, &user.RoleID,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&user.CreatedBy, &user.UpdatedBy, &user.DeletedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

func (r *userRepository) List(ctx context.Context, p Pagination) (*Page[models.User], error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, first_name, last_name, email, password_hash, role_id,
		       created_at, updated_at, deleted_at, created_by, updated_by, deleted_by
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1 OFFSET $2
	`, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.PasswordHash, &user.RoleID,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
			&user.CreatedBy, &user.UpdatedBy, &user.DeletedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return &Page[models.User]{Data: users, Total: total, Limit: p.Limit, Offset: p.Offset}, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	user.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET first_name = $1, last_name = $2, email = $3, role_id = $4, updated_at = $5, updated_by = $6
		WHERE id = $7 AND deleted_at IS NULL
	`, user.FirstName, user.LastName, user.Email, user.RoleID, user.UpdatedAt, user.UpdatedBy, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE users SET deleted_at = $1, deleted_by = $2 WHERE id = $3 AND deleted_at IS NULL
	`, now, deletedBy, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
