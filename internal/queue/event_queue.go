// Package queue provides an asynchronous event queue with batch processing.
// Events are enqueued immediately and persisted to disk, then flushed to the
// database in batches based on time or size thresholds.
package queue

import (
	"collider/internal/models"
	"collider/internal/retry"
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	MaxSize int

	FlushInterval time.Duration

	BatchSize int

	PersistencePath string

	RetryConfig retry.Config

	DeadLetterQueueSize int
}

func DefaultConfig() Config {
	return Config{
		MaxSize:             50000,
		FlushInterval:       50 * time.Millisecond,
		BatchSize:           500,
		PersistencePath:     "/data/pending_events.db",
		RetryConfig:         retry.DefaultConfig(),
		DeadLetterQueueSize: 1000,
	}
}

type EventStore interface {
	BatchInsertEvents(events []models.Event) error

	BatchInsertEventsContext(ctx context.Context, events []models.Event) error

	GetOrCreateTypeID(eventType string) (int64, error)
}

type EnqueueResult struct {
	EventID int64

	QueuedAt time.Time

	QueueDepth int

	Error error
}

type Stats struct {
	CurrentDepth int

	MaxDepth int

	TotalEnqueued int64

	TotalFlushed int64

	TotalFailed int64

	DeadLetterCount int

	PersistencePending int

	LastFlushAt time.Time
}

type EventQueue struct {
	config Config

	store       EventStore
	persistence *Persistence

	eventCh chan PendingEvent

	shutdownCtx      context.Context
	shutdownCancel   context.CancelFunc
	shutdownSignalCh chan struct{}
	wg               sync.WaitGroup

	enqueueMu sync.RWMutex

	stats struct {
		currentDepth  atomic.Int64
		maxDepth      atomic.Int64
		totalEnqueued atomic.Int64
		totalFlushed  atomic.Int64
		totalFailed   atomic.Int64
		lastFlushAt   atomic.Int64
	}

	dlq *retry.DeadLetterQueue

	started atomic.Bool
	stopped atomic.Bool
}

func NewEventQueue(store EventStore, config Config) (*EventQueue, error) {
	if config.MaxSize <= 0 {
		return nil, fmt.Errorf("max size must be positive")
	}
	if config.BatchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive")
	}
	if config.BatchSize > config.MaxSize {
		return nil, fmt.Errorf("batch size cannot exceed max size")
	}
	if config.FlushInterval <= 0 {
		return nil, fmt.Errorf("flush interval must be positive")
	}

	dlq := retry.NewDeadLetterQueue(config.DeadLetterQueueSize)

	var persistence *Persistence
	var err error
	if config.PersistencePath != "" {
		persistence, err = NewPersistence(config.PersistencePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistence: %w", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())

	return &EventQueue{
		config:           config,
		store:            store,
		persistence:      persistence,
		eventCh:          make(chan PendingEvent, config.MaxSize),
		shutdownCtx:      shutdownCtx,
		shutdownCancel:   shutdownCancel,
		shutdownSignalCh: make(chan struct{}),
		dlq:              dlq,
	}, nil
}

func (eq *EventQueue) Start() error {
	if !eq.started.CompareAndSwap(false, true) {
		return ErrQueueAlreadyStarted
	}

	if eq.persistence != nil {
		events, err := eq.persistence.Recover()
		if err != nil {
			return fmt.Errorf("failed to recover pending events: %w", err)
		}

		recoveredCount := 0
		for _, event := range events {
			select {
			case eq.eventCh <- event:
				eq.stats.totalEnqueued.Add(1)
				recoveredCount++
			case <-time.After(100 * time.Millisecond):
				log.Printf("queue: timeout re-enqueuing event %d during recovery", event.EventID)
			}
		}

		if recoveredCount > 0 {
			log.Printf("queue: recovered %d events from persistence", recoveredCount)
		}
		if len(events) != recoveredCount {
			log.Printf("queue: warning: %d events could not be re-enqueued during recovery", len(events)-recoveredCount)
		}
	}

	eq.wg.Add(1)
	go eq.flushLoop()

	log.Printf("queue: started (maxSize=%d, batchSize=%d, flushInterval=%v)",
		eq.config.MaxSize, eq.config.BatchSize, eq.config.FlushInterval)

	return nil
}

func (eq *EventQueue) Enqueue(userID int64, eventType string, metadata []byte, eventID int64) EnqueueResult {
	eq.enqueueMu.RLock()
	stopped := eq.stopped.Load()
	eq.enqueueMu.RUnlock()

	if stopped {
		return EnqueueResult{
			Error: ErrQueueStopped,
		}
	}

	depth := int(eq.stats.currentDepth.Load())

	if depth >= eq.config.MaxSize {
		return EnqueueResult{
			Error: fmt.Errorf("%w (depth=%d, max=%d)", ErrQueueFull, depth, eq.config.MaxSize),
		}
	}

	now := time.Now()
	pendingEvent := PendingEvent{
		EventID:    eventID,
		UserID:     userID,
		EventType:  eventType,
		Metadata:   metadata,
		EnqueuedAt: now.UnixNano(),
	}

	if eq.persistence != nil {
		errCh := eq.persistence.EnqueueAsync(pendingEvent)
		go func() {
			if err := <-errCh; err != nil {
				log.Printf("queue: async persistence failed for event %d: %v", eventID, err)
			}
		}()
	}

	eq.enqueueMu.Lock()
	select {
	case eq.eventCh <- pendingEvent:
		eq.stats.totalEnqueued.Add(1)
		newDepth := eq.stats.currentDepth.Add(1)

		for {
			maxD := eq.stats.maxDepth.Load()
			if newDepth <= maxD || eq.stats.maxDepth.CompareAndSwap(maxD, newDepth) {
				break
			}
		}

		eq.enqueueMu.Unlock()
		return EnqueueResult{
			EventID:    eventID,
			QueuedAt:   now,
			QueueDepth: int(newDepth),
		}
	default:
		eq.enqueueMu.Unlock()
		return EnqueueResult{
			Error: ErrQueueChannelFull,
		}
	}
}

func (eq *EventQueue) flushLoop() {
	defer eq.wg.Done()

	ticker := time.NewTicker(eq.config.FlushInterval)
	defer ticker.Stop()

	batch := make([]PendingEvent, 0, eq.config.BatchSize)

	flushBatch := func(events []PendingEvent) {
		if len(events) == 0 {
			return
		}

		start := time.Now()
		err := eq.flush(events)
		duration := time.Since(start)

		if err != nil {
			log.Printf("queue: flush failed for %d events after %v: %v", len(events), duration, err)
		} else {
			log.Printf("queue: flushed %d events in %v", len(events), duration)
		}
	}

	for {
		select {
		case event := <-eq.eventCh:
			batch = append(batch, event)
			eq.stats.currentDepth.Add(-1)

			if len(batch) >= eq.config.BatchSize {
				flushBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				flushBatch(batch)
				batch = batch[:0]
			}

		case <-eq.shutdownSignalCh:
			draining := true
			for draining {
				select {
				case event := <-eq.eventCh:
					batch = append(batch, event)
					eq.stats.currentDepth.Add(-1)
				default:
					draining = false
				}
			}

			if len(batch) > 0 {
				log.Printf("queue: shutdown: flushing %d remaining events", len(batch))
				flushBatch(batch)
			}
			return
		}
	}
}

func (eq *EventQueue) flush(events []PendingEvent) error {
	if len(events) == 0 {
		return nil
	}

	modelEvents := make([]models.Event, len(events))
	eventIDs := make([]int64, len(events))

	for i, pending := range events {
		modelEvents[i] = models.Event{
			ID:       pending.EventID,
			UserID:   pending.UserID,
			Type:     pending.EventType,
			Metadata: pending.Metadata,
		}
		eventIDs[i] = pending.EventID
	}

	dlqSizeBefore := eq.dlq.Len()

	retry.BatchRetry(
		eq.shutdownCtx,
		eq.config.RetryConfig,
		modelEvents,
		eq.batchInsertWrapper,
		eq.insertSingleWrapper,
		eq.dlq,
	)

	dlqSizeAfter := eq.dlq.Len()
	failedInThisBatch := dlqSizeAfter - dlqSizeBefore

	var idsToRemove []int64
	if failedInThisBatch == 0 {
		idsToRemove = eventIDs
	} else if failedInThisBatch < len(eventIDs) {
		dlqEvents := eq.dlq.GetAll()
		failedIDs := make(map[int64]struct{}, failedInThisBatch)
		startIdx := len(dlqEvents) - int(failedInThisBatch)
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(dlqEvents); i++ {
			if event, ok := dlqEvents[i].Event.(models.Event); ok {
				failedIDs[event.ID] = struct{}{}
			}
		}

		for _, id := range eventIDs {
			if _, failed := failedIDs[id]; !failed {
				idsToRemove = append(idsToRemove, id)
			}
		}
	}

	if eq.persistence != nil && len(idsToRemove) > 0 {
		if err := eq.persistence.Remove(idsToRemove); err != nil {
			log.Printf("queue: failed to remove %d persisted events: %v", len(idsToRemove), err)
		} else if failedInThisBatch > 0 {
			log.Printf("queue: removed %d succeeded events from persistence, %d events failed", len(idsToRemove), failedInThisBatch)
		}
	}

	flushedCount := int64(len(eventIDs) - failedInThisBatch)
	eq.stats.totalFlushed.Add(flushedCount)
	eq.stats.lastFlushAt.Store(time.Now().UnixNano())

	eq.stats.totalFailed.Store(int64(dlqSizeAfter))

	return nil
}

func (eq *EventQueue) batchInsertWrapper(ctx context.Context, events []models.Event) error {
	return eq.store.BatchInsertEventsContext(ctx, events)
}

func (eq *EventQueue) insertSingleWrapper(ctx context.Context, event models.Event) error {
	return eq.store.BatchInsertEventsContext(ctx, []models.Event{event})
}

func (eq *EventQueue) Shutdown(timeout time.Duration) error {
	if !eq.stopped.CompareAndSwap(false, true) {
		return ErrQueueAlreadyStopped
	}

	log.Printf("queue: shutting down (timeout=%v)...", timeout)

	eq.enqueueMu.Lock()

	close(eq.shutdownSignalCh)

	eq.enqueueMu.Unlock()

	done := make(chan struct{})
	go func() {
		eq.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("queue: shutdown complete")
	case <-time.After(timeout):
		log.Printf("queue: shutdown timeout after %v", timeout)
	}

	eq.shutdownCancel()

	if eq.persistence != nil {
		if err := eq.persistence.Close(); err != nil {
			log.Printf("queue: failed to close persistence: %v", err)
		}
	}

	return nil
}

func (eq *EventQueue) GetStats() Stats {
	stats := Stats{
		CurrentDepth:    int(eq.stats.currentDepth.Load()),
		MaxDepth:        int(eq.stats.maxDepth.Load()),
		TotalEnqueued:   eq.stats.totalEnqueued.Load(),
		TotalFlushed:    eq.stats.totalFlushed.Load(),
		TotalFailed:     eq.stats.totalFailed.Load(),
		DeadLetterCount: eq.dlq.Len(),
	}

	if eq.persistence != nil {
		if count, err := eq.persistence.Count(); err == nil {
			stats.PersistencePending = count
		}
	}

	lastFlush := eq.stats.lastFlushAt.Load()
	if lastFlush > 0 {
		stats.LastFlushAt = time.Unix(0, lastFlush)
	}

	return stats
}

func (eq *EventQueue) IsStarted() bool {
	return eq.started.Load()
}

func (eq *EventQueue) IsStopped() bool {
	return eq.stopped.Load()
}
