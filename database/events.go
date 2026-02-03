package database

import (
	"collider/models"
	"time"

	"github.com/google/uuid"
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

func (s EventStore) GetOrCreateEventType(name string) (uuid.UUID, error) {
	query := `
        insert into event_types (name)
        values ($1)
        on conflict (name)
            do update set name = excluded.name
        returning type_id;
	`

	var typeID uuid.UUID
	err := s.db.Get(&typeID, query, name)
	return typeID, err
}

func (s EventStore) CreateEvent(data models.CreateEventData) (*models.Event, error) {
	var event models.Event
	query := `
		insert into events (user_id, type_id, timestamp, metadata) values ($1, $2, $3, $4)
		returning event_id, user_id, type_id, timestamp, metadata;
	`

	now := time.Now()
	err := s.db.QueryRowx(query, data.UserID, data.TypeID, now, data.Metadata).StructScan(&event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}
