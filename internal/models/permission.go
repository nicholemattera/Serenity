package models

import "github.com/google/uuid"

type ResourceType string

const (
	ResourceTypeComposite  ResourceType = "composite"
	ResourceTypeField      ResourceType = "field"
	ResourceTypeUser       ResourceType = "user"
	ResourceTypeRole       ResourceType = "role"
	ResourceTypeEntity     ResourceType = "entity"
	ResourceTypeFieldValue ResourceType = "field_value"
	ResourceTypePermission ResourceType = "permission"
)

// Permission grants a role read/write access to either a specific composite or
// a built-in resource type. Exactly one of CompositeID or ResourceType is set.
type Permission struct {
	ID           uuid.UUID     `json:"id"`
	RoleID       uuid.UUID     `json:"role_id"`
	CompositeID  *uuid.UUID    `json:"composite_id,omitempty"`
	ResourceType *ResourceType `json:"resource_type,omitempty"`
	CanRead      bool          `json:"can_read"`
	CanWrite     bool          `json:"can_write"`
	Audit
}
