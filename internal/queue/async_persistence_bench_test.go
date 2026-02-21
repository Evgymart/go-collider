// Package queue provides benchmark for async persistence verification
package queue

import (
	"collider/internal/models"
	"collider/internal/snowflake"
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// BenchmarkEnqueueAsync measures the performance of enqueue operations with async persistence.
// This should be significantly faster than sync persistence since BoltDB writes don't block.
func BenchmarkEnqueueAsync(b *testing.B) {
	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_async.db")

	p, err := NewPersistence(persistencePath)
	if err != nil {
		b.Fatalf("failed to create persistence: %v", err)
	}
	defer p.Close()

	// Warmup - fill the write buffer
	sf, _ := snowflake.New(1)
	for i := 0; i < 100; i++ {
		eventID, _ := sf.Generate()
		event := PendingEvent{
			EventID:   eventID,
			UserID:    1,
			EventType: "warmup",
			Metadata:  []byte("{}"),
		}
		errCh := p.EnqueueAsync(event)
		// Don't wait for result, just drain the channel
		go func() { <-errCh }()
	}

	// Give warmup writes time to complete
	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark the async enqueue operation
	b.RunParallel(func(pb *testing.PB) {
		sf, _ := snowflake.New(1)
		for pb.Next() {
			eventID, _ := sf.Generate()
			metadata := json.RawMessage(`{"page": "/benchmark", "action": "click"}`)
			event := PendingEvent{
				EventID:   eventID,
				UserID:    1,
				EventType: "benchmark.async",
				Metadata:  metadata,
			}

			// Async enqueue - should return immediately
			errCh := p.EnqueueAsync(event)

			// Drain error channel to prevent goroutine leaks
			// In production, this would be handled by a monitoring goroutine
			go func() { <-errCh }()
		}
	})
}

// BenchmarkEnqueueSync compares synchronous enqueue performance (baseline).
// This is the OLD behavior that was causing slow RPS.
func BenchmarkEnqueueSync(b *testing.B) {
	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_sync.db")

	p, err := NewPersistence(persistencePath)
	if err != nil {
		b.Fatalf("failed to create persistence: %v", err)
	}
	defer p.Close()

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark the sync enqueue operation
	b.RunParallel(func(pb *testing.PB) {
		sf, _ := snowflake.New(1)
		for pb.Next() {
			eventID, _ := sf.Generate()
			metadata := json.RawMessage(`{"page": "/benchmark", "action": "click"}`)
			event := PendingEvent{
				EventID:   eventID,
				UserID:    1,
				EventType: "benchmark.sync",
				Metadata:  metadata,
			}

			// Sync enqueue - blocks on BoltDB write
			if err := p.Enqueue(event); err != nil {
				b.Fatalf("enqueue failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentEnqueueAsync simulates 50 concurrent connections.
// This is the real-world scenario that should show significant improvement.
func BenchmarkConcurrentEnqueueAsync(b *testing.B) {
	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_concurrent.db")

	p, err := NewPersistence(persistencePath)
	if err != nil {
		b.Fatalf("failed to create persistence: %v", err)
	}
	defer p.Close()

	b.ResetTimer()
	b.ReportAllocs()

	// Simulate 50 concurrent goroutines
	const numGoroutines = 50
	var wg sync.WaitGroup
	itemsPerGoroutine := b.N / numGoroutines

	start := make(chan struct{})

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sf, _ := snowflake.New(1)
			<-start // Wait for signal

			for i := 0; i < itemsPerGoroutine; i++ {
				eventID, _ := sf.Generate()
				metadata := json.RawMessage(`{"page": "/benchmark"}`)
				event := PendingEvent{
					EventID:   eventID,
					UserID:    int64(time.Now().UnixNano() % 1000),
					EventType: "concurrent.async",
					Metadata:  metadata,
				}

				errCh := p.EnqueueAsync(event)
				go func() { <-errCh }()
			}
		}()
	}

	// Start all goroutines at once
	close(start)
	wg.Wait()
}

// BenchmarkMockEventStore is a simple mock for benchmarking.
type BenchmarkMockStore struct {
	mu          sync.Mutex
	eventCount  int
	lastBatchAt time.Time
}

func (s *BenchmarkMockStore) BatchInsertEvents(events []models.Event) error {
	return s.BatchInsertEventsContext(nil, events)
}

func (s *BenchmarkMockStore) BatchInsertEventsContext(_ context.Context, events []models.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventCount += len(events)
	s.lastBatchAt = time.Now()
	// Simulate some processing time
	time.Sleep(time.Duration(len(events)) * 10 * time.Microsecond)
	return nil
}

func (s *BenchmarkMockStore) GetOrCreateTypeID(eventType string) (int64, error) {
	return 1, nil
}

// BenchmarkEventQueueEnqueue measures full queue enqueue performance.
func BenchmarkEventQueueEnqueue(b *testing.B) {
	tempDir := b.TempDir()
	persistencePath := filepath.Join(tempDir, "bench_queue.db")

	config := DefaultConfig()
	config.MaxSize = 100000
	config.BatchSize = 500
	config.FlushInterval = 100 * time.Millisecond
	config.PersistencePath = persistencePath

	mockStore := &BenchmarkMockStore{}

	queue, err := NewEventQueue(mockStore, config)
	if err != nil {
		b.Fatalf("failed to create queue: %v", err)
	}

	if err := queue.Start(); err != nil {
		b.Fatalf("failed to start queue: %v", err)
	}
	defer queue.Shutdown(5 * time.Second)

	// Warmup
	sf, _ := snowflake.New(1)
	for i := 0; i < 100; i++ {
		eventID, _ := sf.Generate()
		metadata := json.RawMessage(`{"page": "/warmup"}`)
		queue.Enqueue(1, "warmup", metadata, eventID)
	}
	time.Sleep(150 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		sf, _ := snowflake.New(1)
		userID := time.Now().UnixNano() % 1000
		for pb.Next() {
			eventID, _ := sf.Generate()
			metadata := json.RawMessage(`{"page": "/benchmark", "action": "click"}`)
			queue.Enqueue(userID, "benchmark.async", metadata, eventID)
		}
	})
}
