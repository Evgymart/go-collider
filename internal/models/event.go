package models

import (
	"encoding/json"
	"time"
)

type Event struct {
	ID        int64           `json:"id" db:"event_id"`
	UserID    int64           `json:"user_id" db:"user_id"`
	TypeID    int64           `json:"type_id" db:"type_id"`
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
	UserID   int64           `json:"user_id"`
	Type     string          `json:"event_type"`
	Metadata json.RawMessage `json:"metadata"`
}

type EventData struct {
	Data Event `json:"data"`
}

// CachedPaginatedEvents represents a paginated events response that may be
// served from cache as raw JSON bytes. This enables pass-through caching
// to avoid double JSON serialization (unmarshal from cache + marshal for response).
type CachedPaginatedEvents struct {
	// RawJSON contains pre-serialized JSON from cache. If set, Data should not be used.
	RawJSON []byte

	// Data contains the deserialized PaginatedEvents. If RawJSON is set, this is nil.
	Data *PaginatedEvents
}

// IsCached returns true if the response contains cached raw JSON bytes.
func (c *CachedPaginatedEvents) IsCached() bool {
	return len(c.RawJSON) > 0
}
