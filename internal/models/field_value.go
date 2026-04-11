package models

import "github.com/google/uuid"

type FieldValue struct {
	ID       uuid.UUID `json:"id"`
	EntityID uuid.UUID `json:"entity_id"`
	FieldID  uuid.UUID `json:"field_id"`
	Value    string    `json:"value"`
	Audit
}
