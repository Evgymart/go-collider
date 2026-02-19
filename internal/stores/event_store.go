package stores

import (
	"collider/internal/models"
	"collider/internal/snowflake"
	"log"
	"sync"

	"github.com/jmoiron/sqlx"
)

type EventStore struct {
	db              *sqlx.DB
	snowflake       *snowflake.Snowflake
	typeCache       sync.Map
	cacheWarmedUp   bool
	cacheWarmupOnce sync.Once
}

func NewEventStore(db *sqlx.DB, nodeID int64) *EventStore {
	sf, err := snowflake.New(nodeID)
	if err != nil {
		panic(err)
	}

	return &EventStore{
		db:        db,
		snowflake: sf,
	}
}

func (s EventStore) GetPaginated(page uint, limit uint, UserID *int64) ([]models.Event, error) {
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
        where $1::bigint is null or user_id = $1
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

func (s EventStore) GetTotal(UserID *int64) (int, error) {
	query := `
		select count(*) from events
		where $1::bigint is null or user_id = $1
	`

	var userIDArg interface{} = nil
	if UserID != nil {
		userIDArg = *UserID
	}

	var count int
	err := s.db.Get(&count, query, userIDArg)
	return count, err
}

func (s EventStore) GetEventTypeId(name string) (*int64, error) {
	query := `
		select type_id from event_types where name = $1
	`
	var typeID *int64
	err := s.db.Get(&typeID, query, name)
	return typeID, err
}

func (s *EventStore) warmTypeCache() error {
	query := `
		select name, type_id from event_types
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var typeID int64
		if err := rows.Scan(&name, &typeID); err != nil {
			return err
		}
		s.typeCache.Store(name, typeID)
	}
	return rows.Err()
}

func (s *EventStore) getOrCreateTypeID(eventType string) (int64, error) {
	if cached, ok := s.typeCache.Load(eventType); ok {
		return cached.(int64), nil
	}

	query := `
		with ins as (
			insert into event_types (name)
			values ($1)
			on conflict (name) do nothing
			returning type_id
		)
		select type_id from ins
		union all
		select type_id from event_types where name = $1
		limit 1
	`
	var typeID int64
	err := s.db.Get(&typeID, query, eventType)
	if err != nil {
		return 0, err
	}

	s.typeCache.Store(eventType, typeID)
	return typeID, nil
}

func (s *EventStore) CreateEventWithType(userID int64, eventType string, metadata []byte) (*models.Event, error) {
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	var warmupErr error
	s.cacheWarmupOnce.Do(func() {
		warmupErr = s.warmTypeCache()
		s.cacheWarmedUp = true
	})
	if warmupErr != nil {
		return nil, warmupErr
	}

	typeID, err := s.getOrCreateTypeID(eventType)
	if err != nil {
		return nil, err
	}

	eventID, err := s.snowflake.Generate()
	if err != nil {
		log.Printf("error: failed to generate snowflake ID: %v", err)
		return nil, err
	}

	var event models.Event
	query := `
		insert into events (event_id, user_id, type_id, metadata)
		values ($1, $2, $3, $4)
		returning event_id, user_id, type_id, timestamp, metadata
	`
	err = s.db.QueryRowx(query, eventID, userID, typeID, metadata).StructScan(&event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}
