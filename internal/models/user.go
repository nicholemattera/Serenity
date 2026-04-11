package models

import "github.com/google/uuid"

type User struct {
	ID           uuid.UUID `json:"id"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	RoleID       uuid.UUID `json:"role_id"`
	Audit
}
