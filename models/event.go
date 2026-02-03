package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        uuid.UUID       `json:"id" db:"event_id"`
	UserID    uuid.UUID       `json:"user_id" db:"user_id"`
	TypeID    uuid.UUID       `json:"type_id" db:"type_id"`
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
	Metadata  json.RawMessage `json:"metadata" db:"metadata"`
	Type      string          `json:"type" db:"type"`
}

type PaginatedEvents struct {
	Data  []Event `json:"data"`
	Page  uint    `json:"page"`
	Limit uint    `json:"limit"`
	Total uint    `json:"total"`
}

type CreateEventInput struct {
	UserID   uuid.UUID       `json:"user_id"`
	Type     string          `json:"event_type"`
	Metadata json.RawMessage `json:"metadata"`
}

type CreateEventData struct {
	UserID   uuid.UUID
	TypeID   uuid.UUID
	Metadata json.RawMessage
}
