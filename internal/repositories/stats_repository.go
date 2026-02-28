package repositories

import (
	"collider/internal/cache"
	"collider/internal/models"
	"collider/internal/stores"

	"context"
	"fmt"
	"log"
)

// StatsRepository defines the interface for stats operations with caching.
type StatsRepository interface {
	// GetStats retrieves statistics with caching support.
	GetStats(ctx context.Context, data models.GetStatsData) (*models.Stats, error)

	// GetEventTypeId gets an event type ID by name (used for validation).
	GetEventTypeId(eventType string) (*int64, error)
}

// CachedStatsRepository implements StatsRepository with caching support.
type CachedStatsRepository struct {
	store      *stores.StatsStore
	eventStore *stores.EventStore
	cache      *cache.Cache
}

// NewStatsRepository creates a new StatsRepository. If cache is nil, caching is disabled.
func NewStatsRepository(store *stores.StatsStore, eventStore *stores.EventStore, c *cache.Cache) StatsRepository {
	return &CachedStatsRepository{
		store:      store,
		eventStore: eventStore,
		cache:      c,
	}
}

func (r *CachedStatsRepository) GetStats(ctx context.Context, data models.GetStatsData) (*models.Stats, error) {
	if r.cache != nil {
		cacheKey := cache.StatsKey(data.From, data.To, data.TypeID)
		var cachedStats models.Stats

		if r.cache.Get(ctx, cacheKey, &cachedStats) {
			log.Printf("cache hit: %s", cacheKey)
			return &cachedStats, nil
		}
	}

	stats, err := r.store.GetStats(data)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	if r.cache != nil {
		r.cache.Set(context.Background(), cache.StatsKey(data.From, data.To, data.TypeID), stats, cache.TTLStats)
	}

	return stats, nil
}

func (r *CachedStatsRepository) GetEventTypeId(eventType string) (*int64, error) {
	return r.eventStore.GetEventTypeId(eventType)
}
