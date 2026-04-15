package models

import "github.com/google/uuid"

type Entity struct {
	ID           uuid.UUID `json:"id"`
	CompositeID  uuid.UUID `json:"composite_id"`
	TreeID       uuid.UUID `json:"tree_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Left         int       `json:"left"`
	Right        int       `json:"right"`
	RootPosition *int      `json:"root_position,omitempty"`
	Audit
}
