package models

import "github.com/google/uuid"

type Composite struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	DefaultRead  bool      `json:"default_read"`
	DefaultWrite bool      `json:"default_write"`
	Audit
}
