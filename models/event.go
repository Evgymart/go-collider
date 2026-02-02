package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType struct {
	ID     uuid.UUID `json:"id" db:"id"`
	Name   string    `json:"name" db:"name"`
	Events []*Event  `json:"events,omitempty" db:"-"`
}

type Event struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    uuid.UUID       `json:"user_id" db:"user_id"`
	TypeID    uuid.UUID       `json:"type_id" db:"type_id"`
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
	Metadata  json.RawMessage `json:"metadata" db:"metadata"`
}
