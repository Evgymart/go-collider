// Package repositories provides a repository layer that handles caching logic
// on top of the database stores. This separates concerns between data access
// and caching strategies.
package repositories

import (
	"collider/internal/cache"
	"collider/internal/models"
	"collider/internal/stores"

	"context"
	"fmt"
	"time"
)

const cacheInvalidationTimeout = 50 * time.Millisecond

// EventRepository defines the interface for event data operations with caching.
type EventRepository interface {
	// GetPaginated retrieves paginated events with caching support.
	GetPaginated(ctx context.Context, page, limit uint, userID *int64) (*models.PaginatedEvents, error)

	// CreateEvent creates a new event and invalidates relevant caches.
	CreateEvent(ctx context.Context, input models.CreateEventInput) (*models.EventData, error)

	// CreateEventWithType creates a new event with a specific type and invalidates relevant caches.
	CreateEventWithType(ctx context.Context, userID int64, eventType string, metadata []byte) (*models.Event, error)

	// GenerateEventID generates a new unique event ID.
	GenerateEventID() (int64, error)

	// GetOrCreateTypeID gets or creates an event type ID.
	GetOrCreateTypeID(eventType string) (int64, error)

	// InsertEventWithID inserts an event with a specific ID and invalidates relevant caches.
	InsertEventWithID(ctx context.Context, event models.Event) error

	// BatchInsertEventsContext batch inserts events (used by queue).
	BatchInsertEventsContext(ctx context.Context, events []models.Event) error

	// InvalidateCaches invalidates all relevant caches after an event is created.
	InvalidateCaches(ctx context.Context)
}

// CachedEventRepository implements EventRepository with caching support.
type CachedEventRepository struct {
	store *stores.EventStore
	cache *cache.Cache
}

// NewEventRepository creates a new EventRepository. If cache is nil, caching is disabled.
func NewEventRepository(store *stores.EventStore, c *cache.Cache) EventRepository {
	return &CachedEventRepository{
		store: store,
		cache: c,
	}
}

func (r *CachedEventRepository) GetPaginated(ctx context.Context, page, limit uint, userID *int64) (*models.PaginatedEvents, error) {
	cacheKey := cache.EventsKey(page, limit, userID)

	// Try cache first
	if r.cache != nil {
		var cachedResponse models.PaginatedEvents
		if r.cache.Get(ctx, cacheKey, &cachedResponse) {
			return &cachedResponse, nil
		}
	}

	// Cache miss - get from store
	events, err := r.store.GetPaginated(page, limit, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get paginated events: %w", err)
	}

	total, err := r.store.GetTotal(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}

	response := &models.PaginatedEvents{
		Data:  events,
		Limit: limit,
		Page:  page,
		Total: uint(total),
	}

	// Cache the entire response
	if r.cache != nil {
		r.cache.Set(context.Background(), cacheKey, response, cache.TTLEvents)
	}

	return response, nil
}

func (r *CachedEventRepository) CreateEvent(ctx context.Context, input models.CreateEventInput) (*models.EventData, error) {
	event, err := r.CreateEventWithType(ctx, input.UserID, input.Type, input.Metadata)
	if err != nil {
		return nil, err
	}

	return &models.EventData{Data: *event}, nil
}

func (r *CachedEventRepository) CreateEventWithType(ctx context.Context, userID int64, eventType string, metadata []byte) (*models.Event, error) {
	event, err := r.store.CreateEventWithType(userID, eventType, metadata)
	if err != nil {
		return nil, err
	}

	r.invalidateCaches(ctx)

	event.Type = eventType
	return event, nil
}

func (r *CachedEventRepository) GenerateEventID() (int64, error) {
	return r.store.GenerateEventID()
}

func (r *CachedEventRepository) GetOrCreateTypeID(eventType string) (int64, error) {
	cacheKey := cache.EventTypeIdKey(eventType)

	// Try cache first
	if r.cache != nil {
		var cachedTypeID int64
		if r.cache.Get(context.Background(), cacheKey, &cachedTypeID) {
			return cachedTypeID, nil
		}
	}

	// Cache miss - get from store
	typeID, err := r.store.GetOrCreateTypeID(eventType)
	if err != nil {
		return 0, fmt.Errorf("failed to get or create type ID: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(context.Background(), cacheKey, typeID, cache.TTLEventTypeId)
	}

	return typeID, nil
}

func (r *CachedEventRepository) InsertEventWithID(ctx context.Context, event models.Event) error {
	if err := r.store.InsertEventWithID(event); err != nil {
		return err
	}

	r.invalidateCaches(ctx)
	return nil
}

func (r *CachedEventRepository) BatchInsertEventsContext(ctx context.Context, events []models.Event) error {
	return r.store.BatchInsertEventsContext(ctx, events)
}

func (r *CachedEventRepository) invalidateCaches(ctx context.Context) {
	if r.cache == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, cacheInvalidationTimeout)
	defer cancel()

	// Invalidate all events cache (global and user-specific)
	r.cache.InvalidateEvents(ctx)
	// Invalidate stats since new events affect analytics
	r.cache.InvalidateStats(ctx)
}

func (r *CachedEventRepository) InvalidateCaches(ctx context.Context) {
	r.invalidateCaches(ctx)
}
