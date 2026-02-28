// Package cache provides a Redis-backed caching layer using Dragonfly.
// It handles serialization, namespacing, and graceful degradation on cache failures.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// TTLEvents is the cache duration for paginated event lists.
	TTLEvents = 5 * time.Minute

	// TTLStats is the cache duration for analytics stats.
	TTLStats = 15 * time.Minute

	// TTLEventTypeId is the cache duration for event type ID lookups.
	TTLEventTypeId = 30 * time.Minute
)

// keyPrefixes define namespace prefixes for different cache entry types.
// Namespacing prevents key collisions and enables pattern-based invalidation.
const (
	keyPrefixEvents      = "events"
	keyPrefixStats       = "stats"
	keyPrefixEventTypeId = "event_type_id"
)

// CacheStats tracks cache health metrics.
type CacheStats struct {
	mu     sync.RWMutex
	hits   int64
	misses int64
	errors int64
}

// Cache provides caching operations with graceful degradation.
// All cache operations fall back gracefully on errors, allowing the application
// to continue functioning by hitting the database directly.
type Cache struct {
	client *redis.Client
	stats  CacheStats
}

// New creates a new Cache instance with the given Redis client.
// The client should be configured with appropriate timeouts and connection pooling.
func New(client *redis.Client) *Cache {
	return &Cache{
		client: client,
	}
}

// NewFromURL creates a new Cache instance by connecting to the Redis server at url.
// The url should be in the format "host:port".
// poolSize specifies the maximum number of socket connections.
// Connection failures are logged but do not return an error, allowing the app to start.
func NewFromURL(url string, poolSize int) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr:         url,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		PoolSize:     poolSize,
		MinIdleConns: 2,
	})

	// Test connection but don't fail if it's unavailable - cache is optional
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("cache: failed to connect to redis at %s: %v (continuing without cache)", url, err)
		return &Cache{client: nil}
	}

	log.Printf("cache: connected to redis at %s", url)
	return &Cache{client: client}
}

// Close closes the Redis client connection.
// It is safe to call on a nil client.
func (c *Cache) Close() error {
	// Log cache stats before closing
	c.stats.mu.RLock()
	hits := c.stats.hits
	misses := c.stats.misses
	errors := c.stats.errors
	c.stats.mu.RUnlock()

	total := hits + misses
	if total > 0 {
		hitRate := float64(hits) / float64(total) * 100
		log.Printf("cache: stats - hits: %d, misses: %d, errors: %d, hit rate: %.2f%%",
			hits, misses, errors, hitRate)
	}

	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

// Get retrieves and deserializes a value into dest.
// Returns true if the value was found and successfully deserialized.
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) bool {
	if c.client == nil {
		c.stats.mu.Lock()
		c.stats.misses++
		c.stats.mu.Unlock()
		return false
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			c.stats.mu.Lock()
			c.stats.misses++
			c.stats.mu.Unlock()
			return false
		}
		c.stats.mu.Lock()
		c.stats.errors++
		c.stats.mu.Unlock()
		log.Printf("cache: get error for key %s: %v", key, err)
		return false
	}

	if err := json.Unmarshal(data, dest); err != nil {
		c.stats.mu.Lock()
		c.stats.errors++
		c.stats.mu.Unlock()
		log.Printf("cache: unmarshal error for key %s: %v (deleting corrupt key)", key, err)
		// Delete the corrupt key to prevent repeated failures
		// Use background context since the request context may be timed out
		if delErr := c.client.Del(context.Background(), key).Err(); delErr != nil {
			log.Printf("cache: delete corrupt key error for %s: %v", key, delErr)
		}
		return false
	}

	c.stats.mu.Lock()
	c.stats.hits++
	c.stats.mu.Unlock()
	return true
}

// Set serializes and stores a value with the specified TTL.
func (c *Cache) Set(_ context.Context, key string, value interface{}, ttl time.Duration) {
	if c.client == nil {
		return
	}

	data, err := json.Marshal(value)
	if err != nil {
		log.Printf("cache: marshal error for key %s: %v", key, err)
		return
	}

	// Use background context to ensure the Set operation completes
	// even if the request context (used for Get) has timed out
	ctx := context.Background()
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.stats.mu.Lock()
		c.stats.errors++
		c.stats.mu.Unlock()
		log.Printf("cache: set error for key %s: %v", key, err)
	}
}

// Delete removes a key from the cache.
func (c *Cache) Delete(_ context.Context, key string) {
	if c.client == nil {
		return
	}

	// Use background context for cache invalidation operations
	ctx := context.Background()
	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.stats.mu.Lock()
		c.stats.errors++
		c.stats.mu.Unlock()
		log.Printf("cache: delete error for key %s: %v", key, err)
	}
}

// DeleteByPattern removes all keys matching the given pattern.
func (c *Cache) DeleteByPattern(_ context.Context, pattern string) {
	if c.client == nil {
		return
	}

	// Use background context for cache invalidation operations
	ctx := context.Background()

	// Use SCAN to avoid blocking the server
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	keysToDelete := []string{}

	for iter.Next(ctx) {
		keysToDelete = append(keysToDelete, iter.Val())
		if len(keysToDelete) >= 100 {
			if err := c.client.Del(ctx, keysToDelete...).Err(); err != nil {
				c.stats.mu.Lock()
				c.stats.errors++
				c.stats.mu.Unlock()
				log.Printf("cache: batch delete error: %v", err)
			}
			keysToDelete = []string{}
		}
	}

	// Delete remaining keys
	if len(keysToDelete) > 0 {
		if err := c.client.Del(ctx, keysToDelete...).Err(); err != nil {
			c.stats.mu.Lock()
			c.stats.errors++
			c.stats.mu.Unlock()
			log.Printf("cache: batch delete error: %v", err)
		}
	}

	if err := iter.Err(); err != nil {
		c.stats.mu.Lock()
		c.stats.errors++
		c.stats.mu.Unlock()
		log.Printf("cache: scan error for pattern %s: %v", pattern, err)
	}
}

// EventsKey builds a cache key for paginated events lists.
// Handles both global events (userID=nil) and user-specific events.
// Format: "events:page:{page}:limit:{limit}:user:{user_id|all}"
func EventsKey(page, limit uint, userID *int64) string {
	if userID == nil {
		return fmt.Sprintf("%s:page:%d:limit:%d:user:all", keyPrefixEvents, page, limit)
	}
	return fmt.Sprintf("%s:page:%d:limit:%d:user:%d", keyPrefixEvents, page, limit, *userID)
}

// StatsKey builds a cache key for stats queries.
// Format: "stats:from:{timestamp}:to:{timestamp}:type:{type_id}"
// Nil timestamps are represented as "nil" in the key.
func StatsKey(from, to *time.Time, typeID *int64) string {
	fromStr := "nil"
	if from != nil {
		fromStr = from.Format(time.RFC3339)
	}

	toStr := "nil"
	if to != nil {
		toStr = to.Format(time.RFC3339)
	}

	typeStr := "nil"
	if typeID != nil {
		typeStr = fmt.Sprintf("%d", *typeID)
	}

	return fmt.Sprintf("%s:from:%s:to:%s:type:%s", keyPrefixStats, fromStr, toStr, typeStr)
}

// EventTypeIdKey builds a cache key for event type ID lookups.
// Format: "event_type_id:{name}"
func EventTypeIdKey(name string) string {
	return fmt.Sprintf("%s:%s", keyPrefixEventTypeId, name)
}

// InvalidateEvents removes all cached event lists (both global and user-specific).
// Called when any event is created, as it affects pagination.
func (c *Cache) InvalidateEvents(ctx context.Context) {
	c.DeleteByPattern(ctx, keyPrefixEvents+":*")
}

// InvalidateStats removes all cached stats.
// Called when new events are added that would affect analytics.
func (c *Cache) InvalidateStats(ctx context.Context) {
	c.DeleteByPattern(ctx, keyPrefixStats+":*")
}
