package database

import (
	"collider/models"

	"github.com/jmoiron/sqlx"
)

type EventStore struct {
	db *sqlx.DB
}

func NewEventStore(db *sqlx.DB) *EventStore {
	return &EventStore{db: db}
}

func (s EventStore) GetPaginated(page uint, limit uint) ([]models.Event, error) {
	offset := (page - 1) * limit
	var events []models.Event

	query := `
		select
			event_id as id,
			user_id,
			type_id,
			timestamp,
			metadata,
			name as type
		from events
		inner join event_types using (type_id)
		order by event_id desc
		limit $1
		offset $2
	`

	err := s.db.Select(&events, query, limit, offset)
	return events, err
}

func (s EventStore) GetTotal() (int, error) {
	query := `
		select count(*) from events
	`

	var count int
	err := s.db.Get(&count, query)
	return count, err
}
