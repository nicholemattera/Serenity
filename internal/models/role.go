package models

import "github.com/google/uuid"

type Role struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	HierarchyLevel int       `json:"hierarchy_level"`
	SessionTimeout int       `json:"session_timeout"` // in seconds
	Audit
}
