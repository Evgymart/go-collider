// Package queue provides benchmarks for async event queue performance
package queue

import (
	"collider/internal/models"
	"collider/internal/snowflake"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// BenchmarkSyncInsert benchmarks the synchronous insert pattern (baseline)
func BenchmarkSyncInsert(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, ctx, db)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		sf, _ := snowflake.New(1)
		for pb.Next() {
			eventID, _ := sf.Generate()
			metadata := json.RawMessage(`{"page": "/benchmark", "action": "click"}`)
			event := models.Event{
				ID:       eventID,
				UserID:   1,
				Type:     "benchmark.sync",
				Metadata: metadata,
			}

			if err := benchmarkInsertSingle(db, event); err != nil {
				b.Fatalf("insert failed: %v", err)
			}
		}
	})
}

// BenchmarkAsyncQueue benchmarks the async queue pattern
func BenchmarkAsyncQueue(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, ctx, db)

	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_queue.db")

	eventStore := &benchmarkEventStore{db: db}

	config := DefaultConfig()
	config.MaxSize = 50000
	config.BatchSize = 500
	config.FlushInterval = 100 * time.Millisecond
	config.PersistencePath = persistencePath

	queue, err := NewEventQueue(eventStore, config)
	if err != nil {
		b.Fatalf("failed to create queue: %v", err)
	}

	if err := queue.Start(); err != nil {
		b.Fatalf("failed to start queue: %v", err)
	}
	defer queue.Shutdown(10 * time.Second)

	// Warmup
	sf, _ := snowflake.New(1)
	for i := 0; i < 100; i++ {
		eventID, _ := sf.Generate()
		metadata := json.RawMessage(`{"page": "/warmup"}`)
		queue.Enqueue(1, "warmup", metadata, eventID)
	}
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		sf, _ := snowflake.New(1)
		for pb.Next() {
			eventID, _ := sf.Generate()
			metadata := json.RawMessage(`{"page": "/benchmark", "action": "click"}`)
			queue.Enqueue(1, "benchmark.async", metadata, eventID)
		}
	})

	// Wait for all events to be flushed
	stats := queue.GetStats()
	for stats.TotalFlushed < int64(b.N) && stats.CurrentDepth > 0 {
		time.Sleep(50 * time.Millisecond)
		stats = queue.GetStats()
	}

	b.Logf("Queue stats: enqueued=%d, flushed=%d, depth=%d, maxDepth=%d",
		stats.TotalEnqueued, stats.TotalFlushed, stats.CurrentDepth, stats.MaxDepth)
}

// BenchmarkAsyncQueue_50connections simulates 50 concurrent connections
func BenchmarkAsyncQueue_50connections(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, ctx, db)

	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_queue_50.db")

	eventStore := &benchmarkEventStore{db: db}

	config := DefaultConfig()
	config.MaxSize = 100000
	config.BatchSize = 500
	config.FlushInterval = 100 * time.Millisecond
	config.PersistencePath = persistencePath

	queue, err := NewEventQueue(eventStore, config)
	if err != nil {
		b.Fatalf("failed to create queue: %v", err)
	}

	if err := queue.Start(); err != nil {
		b.Fatalf("failed to start queue: %v", err)
	}
	defer queue.Shutdown(10 * time.Second)

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate 50 concurrent connections
	b.RunParallel(func(pb *testing.PB) {
		sf, _ := snowflake.New(1)
		userID := time.Now().UnixNano() % 1000 // Random-ish user ID
		for pb.Next() {
			eventID, _ := sf.Generate()
			metadata := json.RawMessage(`{"page": "/benchmark", "action": "click"}`)
			queue.Enqueue(userID, "benchmark.async.50", metadata, eventID)
		}
	})

	// Wait for flush
	stats := queue.GetStats()
	for stats.TotalFlushed < int64(b.N) && stats.CurrentDepth > 0 {
		time.Sleep(50 * time.Millisecond)
		stats = queue.GetStats()
	}

	b.Logf("Queue stats: enqueued=%d, flushed=%d",
		stats.TotalEnqueued, stats.TotalFlushed)
}

// BenchmarkLatency_P50_P95_P99 measures latency percentiles
func BenchmarkLatency_P50_P95_P99(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	ctx := context.Background()
	db := setupBenchmarkDB(b)
	defer cleanupBenchmarkDB(b, ctx, db)

	eventStore := &benchmarkEventStore{db: db}

	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_latency.db")

	config := DefaultConfig()
	config.MaxSize = 50000
	config.BatchSize = 500
	config.FlushInterval = 100 * time.Millisecond
	config.PersistencePath = persistencePath

	queue, err := NewEventQueue(eventStore, config)
	if err != nil {
		b.Fatalf("failed to create queue: %v", err)
	}

	if err := queue.Start(); err != nil {
		b.Fatalf("failed to start queue: %v", err)
	}
	defer queue.Shutdown(10 * time.Second)

	// Measure enqueue latency (not end-to-end, just the accept time)
	latencies := make([]time.Duration, b.N)

	sf, _ := snowflake.New(1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		eventID, _ := sf.Generate()
		start := time.Now()
		metadata := json.RawMessage(`{"page": "/latency"}`)
		queue.Enqueue(1, "benchmark.latency", metadata, eventID)
		latencies[i] = time.Since(start)
	}

	b.StopTimer()

	// Calculate percentiles
	// Simple selection algorithm for percentiles
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Simple bubble sort (fine for benchmark results)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p50 := sorted[len(sorted)*50/100]
	p95 := sorted[len(sorted)*95/100]
	p99 := sorted[len(sorted)*99/100]

	b.Logf("Latency percentiles: p50=%v, p95=%v, p99=%v", p50, p95, p99)
}

// BenchmarkThroughput_AtDifferentBatchSizes tests different batch sizes
func BenchmarkThroughput_AtDifferentBatchSizes(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	batchSizes := []int{50, 100, 250, 500, 1000}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("batch_%d", batchSize), func(b *testing.B) {
			ctx := context.Background()
			db := setupBenchmarkDB(b)
			defer cleanupBenchmarkDB(b, ctx, db)

			eventStore := &benchmarkEventStore{db: db}

			tempDir := b.TempDir()
			persistencePath := filepath.Join(tempDir, fmt.Sprintf("bench_batch_%d.db", batchSize))

			config := DefaultConfig()
			config.MaxSize = 50000
			config.BatchSize = batchSize
			config.FlushInterval = 100 * time.Millisecond
			config.PersistencePath = persistencePath

			queue, err := NewEventQueue(eventStore, config)
			if err != nil {
				b.Fatalf("failed to create queue: %v", err)
			}

			if err := queue.Start(); err != nil {
				b.Fatalf("failed to start queue: %v", err)
			}
			defer queue.Shutdown(10 * time.Second)

			b.ResetTimer()
			b.ReportAllocs()

			sf, _ := snowflake.New(1)
			for i := 0; i < b.N; i++ {
				eventID, _ := sf.Generate()
				metadata := json.RawMessage(`{"page": "/batch"}`)
				queue.Enqueue(1, "benchmark.batch", metadata, eventID)
			}

			// Report ops/sec
			b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "ops")
		})
	}
}

// Helper types and functions for benchmarks

type benchmarkEventStore struct {
	db *sqlx.DB
}

func (s *benchmarkEventStore) BatchInsertEvents(events []models.Event) error {
	return s.BatchInsertEventsContext(context.Background(), events)
}

func (s *benchmarkEventStore) BatchInsertEventsContext(ctx context.Context, events []models.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Simple batch insert using UNNEST
	query := `
		INSERT INTO events (event_id, user_id, type_id, metadata)
		SELECT * FROM unnest($1::bigint[], $2::bigint[], $3::bigint[], $4::jsonb[])
	`

	ids := make([]int64, len(events))
	userIDs := make([]int64, len(events))
	typeIDs := make([]int64, len(events))
	metadataArray := make([]interface{}, len(events))

	// Use type_id = 1 for all benchmarks (assumes it exists)
	for i, event := range events {
		metadata := event.Metadata
		if len(metadata) == 0 {
			metadata = []byte("{}")
		}

		ids[i] = event.ID
		userIDs[i] = event.UserID
		typeIDs[i] = 1 // Pre-create this type in setup
		metadataArray[i] = metadata
	}

	_, err := s.db.ExecContext(ctx, query, ids, userIDs, typeIDs, metadataArray)
	return err
}

func (s *benchmarkEventStore) GetOrCreateTypeID(eventType string) (int64, error) {
	// For benchmarks, we use a single type ID
	return 1, nil
}

func benchmarkInsertSingle(db *sqlx.DB, event models.Event) error {
	query := `
		INSERT INTO events (event_id, user_id, type_id, metadata)
		VALUES ($1, $2, $3, $4)
	`

	metadata := event.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	_, err := db.Exec(query, event.ID, event.UserID, 1, metadata)
	return err
}

func setupBenchmarkDB(b testing.TB) *sqlx.DB {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("benchdb"),
		postgres.WithUsername("benchuser"),
		postgres.WithPassword("benchpass"),
	)
	if err != nil {
		b.Fatalf("failed to start postgres container: %v", err)
	}

	b.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			b.Errorf("failed to terminate postgres container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		b.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		b.Fatalf("failed to connect to database: %v", err)
	}

	// Setup schema
	schema := `
		CREATE TABLE users (
			user_id BIGSERIAL PRIMARY KEY,
			name VARCHAR(50) NOT NULL
		);

		CREATE TABLE event_types (
			type_id BIGSERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE
		);

		CREATE TABLE events (
			event_id BIGINT NOT NULL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			type_id BIGINT NOT NULL,
			"timestamp" TIMESTAMP(0) NOT NULL DEFAULT NOW(),
			metadata JSONB NOT NULL
		);

		INSERT INTO users (user_id, name) VALUES
			(1, 'bench_user'),
			(2, 'bench_user2');

		INSERT INTO event_types (type_id, name) VALUES
			(1, 'benchmark.test');
	`

	if _, err := db.Exec(schema); err != nil {
		b.Fatalf("failed to setup schema: %v", err)
	}

	return db
}

func cleanupBenchmarkDB(b testing.TB, ctx context.Context, db *sqlx.DB) {
	if err := db.Close(); err != nil {
		b.Logf("warning: failed to close database: %v", err)
	}
}
