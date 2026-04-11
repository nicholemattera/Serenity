package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type FieldType string

const (
	FieldTypeAssociation FieldType = "association"
	FieldTypeCheckbox    FieldType = "checkbox"
	FieldTypeColor       FieldType = "color"
	FieldTypeDate        FieldType = "date"
	FieldTypeDateTime    FieldType = "datetime"
	FieldTypeDropdown    FieldType = "dropdown"
	FieldTypeEmail       FieldType = "email"
	FieldTypeFile        FieldType = "file"
	FieldTypeLongText    FieldType = "long_text"
	FieldTypeNumber      FieldType = "number"
	FieldTypePhone       FieldType = "phone"
	FieldTypeShortText   FieldType = "short_text"
	FieldTypeTime        FieldType = "time"
	FieldTypeURL         FieldType = "url"
)

type Field struct {
	ID           uuid.UUID       `json:"id"`
	CompositeID  uuid.UUID       `json:"composite_id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Type         FieldType       `json:"type"`
	Required     bool            `json:"required"`
	Position     int             `json:"position"`
	DefaultValue *string         `json:"default_value,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	Audit
}
