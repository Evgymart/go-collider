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

func (s EventStore) GetAll() ([]models.Event, error) {
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
		order by event_id desc; 
	`

	err := s.db.Select(&events, query)
	if err != nil {
		return nil, err
	}

	return events, nil
}
