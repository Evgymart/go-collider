package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type GetStatsData struct {
	From   *time.Time `db:"from"`
	To     *time.Time `db:"to"`
	TypeID *uuid.UUID `db:"type_id"`
}

type Stats struct {
	TotalEvents int             `json:"total_events" db:"total_events"`
	UniqueUsers int             `json:"unique_users" db:"unique_users"`
	TopPages    json.RawMessage `json:"top_pages" db:"top_pages"`
}
