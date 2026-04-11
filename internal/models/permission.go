package models

import "github.com/google/uuid"

type Permission struct {
	ID          uuid.UUID `json:"id"`
	RoleID      uuid.UUID `json:"role_id"`
	CompositeID uuid.UUID `json:"composite_id"`
	CanRead     bool      `json:"can_read"`
	CanWrite    bool      `json:"can_write"`
	Audit
}
