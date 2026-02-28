package stores

import (
	"collider/internal/models"
	"collider/internal/snowflake"
	"context"
	"fmt"
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

func (s *EventStore) GenerateEventID() (int64, error) {
	return s.snowflake.Generate()
}

func (s *EventStore) GetPaginated(page uint, limit uint, UserID *int64) ([]models.Event, error) {
	offset := (page - 1) * limit
	var events []models.Event

	var query string
	var args []interface{}

	baseSelect := `
        select
            event_id,
            user_id,
            type_id,
            timestamp,
            metadata,
            name as type
        from events
        inner join event_types using (type_id)
    `

	if UserID == nil {
		query = baseSelect + `
            order by event_id desc
            limit $1
            offset $2
        `
		args = []interface{}{limit, offset}
	} else {
		query = baseSelect + `
            where user_id = $1
            order by event_id desc
            limit $2
            offset $3
        `
		args = []interface{}{*UserID, limit, offset}
	}

	err := s.db.Select(&events, query, args...)
	return events, err
}

// GetTotal returns the total count of events.
func (s *EventStore) GetTotal(UserID *int64) (int, error) {
	var query string
	var args []interface{}

	if UserID == nil {
		query = `select count(*) from events`
		args = []interface{}{}
	} else {
		query = `select count(*) from events where user_id = $1`
		args = []interface{}{*UserID}
	}

	var count int
	err := s.db.Get(&count, query, args...)
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

func (s *EventStore) GetOrCreateTypeID(eventType string) (int64, error) {
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

	typeID, err := s.GetOrCreateTypeID(eventType)
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

func (s *EventStore) BatchInsertEvents(events []models.Event) error {
	return s.BatchInsertEventsContext(context.Background(), events)
}

func (s *EventStore) BatchInsertEventsContext(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}

	var warmupErr error
	s.cacheWarmupOnce.Do(func() {
		warmupErr = s.warmTypeCache()
		s.cacheWarmedUp = true
	})
	if warmupErr != nil {
		return warmupErr
	}

	typeMap := make(map[string]int64)
	uniqueTypes := make([]string, 0)
	for _, event := range events {
		if event.Type == "" {
			return fmt.Errorf("event type cannot be empty")
		}
		if _, exists := typeMap[event.Type]; !exists {
			uniqueTypes = append(uniqueTypes, event.Type)
		}
		typeMap[event.Type] = 0 // placeholder
	}

	for _, eventType := range uniqueTypes {
		typeID, err := s.GetOrCreateTypeID(eventType)
		if err != nil {
			return fmt.Errorf("failed to get type ID for %s: %w", eventType, err)
		}
		typeMap[eventType] = typeID
	}

	query := `
		insert into events (event_id, user_id, type_id, metadata)
		select * from unnest($1::bigint[], $2::bigint[], $3::bigint[], $4::jsonb[])
	`

	ids := make([]int64, len(events))
	userIDs := make([]int64, len(events))
	typeIDs := make([]int64, len(events))
	metadataArray := make([]interface{}, len(events))

	for i, event := range events {
		metadata := event.Metadata
		if len(metadata) == 0 {
			metadata = []byte("{}")
		}

		ids[i] = event.ID
		userIDs[i] = event.UserID
		typeID, ok := typeMap[event.Type]
		if !ok {
			return fmt.Errorf("type ID not found for event type: %s", event.Type)
		}
		typeIDs[i] = typeID
		metadataArray[i] = metadata
	}

	_, err := s.db.ExecContext(ctx, query, ids, userIDs, typeIDs, metadataArray)
	if err != nil {
		return fmt.Errorf("batch insert failed: %w", err)
	}

	log.Printf("batch insert: successfully inserted %d events", len(events))
	return nil
}

func (s *EventStore) InsertEventWithID(event models.Event) error {
	if len(event.Metadata) == 0 {
		event.Metadata = []byte("{}")
	}

	query := `
		insert into events (event_id, user_id, type_id, metadata)
		values ($1, $2, $3, $4)
	`

	_, err := s.db.Exec(query, event.ID, event.UserID, event.TypeID, event.Metadata)
	if err != nil {
		return fmt.Errorf("insert event failed: %w", err)
	}

	return nil
}
