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

func (s EventStore) GetPaginated(page uint, limit uint, UserID *uuid.UUID) ([]models.Event, error) {
	offset := (page - 1) * limit
	var events []models.Event

	query := `
        select
            event_id,
            user_id,
            type_id,
            timestamp,
            metadata,
            name as type
        from events
        inner join event_types using (type_id)
        where $1::uuid is null or user_id = $1
        order by event_id desc
        limit $2
        offset $3
    `

	var userIDArg interface{} = nil
	if UserID != nil {
		userIDArg = *UserID
	}

	err := s.db.Select(&events, query, userIDArg, limit, offset)
	return events, err
}

func (s EventStore) GetTotal(UserID *uuid.UUID) (int, error) {
	query := `
		select count(*) from events
		where $1::uuid is null or user_id = $1
	`

	var userIDArg interface{} = nil
	if UserID != nil {
		userIDArg = *UserID
	}

	var count int
	err := s.db.Get(&count, query, userIDArg)
	return count, err
}

func (s EventStore) GetEvetTypeId(name string) (*uuid.UUID, error) {
	query := `
		select type_id from event_types where name = $1
	`
	var typeID *uuid.UUID
	err := s.db.Get(&typeID, query, name)
	return typeID, err
}

func (s EventStore) CreateEventWithType(userID uuid.UUID, eventType string, metadata []byte) (*models.Event, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var typeID uuid.UUID
	getTypeQuery := `
		insert into event_types (name)
		values ($1)
		on conflict (name)
			do update set name = excluded.name
		returning type_id
	`
	err = tx.Get(&typeID, getTypeQuery, eventType)
	if err != nil {
		return nil, err
	}

	var event models.Event
	createEventQuery := `
		insert into events (user_id, type_id, timestamp, metadata)
		values ($1, $2, $3, $4)
		returning event_id, user_id, type_id, timestamp, metadata
	`
	now := time.Now()
	err = tx.QueryRowx(createEventQuery, userID, typeID, now, metadata).StructScan(&event)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &event, nil
}
