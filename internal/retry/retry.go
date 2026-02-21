// Package retry provides exponential backoff retry mechanisms for operations
// that may fail transiently. It includes dead letter queue support for
// permanently failed events.
package retry

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

type Config struct {
	MaxAttempts int

	BaseDelay time.Duration

	MaxDelay time.Duration

	Multiplier float64
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts: 5,
		BaseDelay:   50 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Multiplier:  2.0,
	}
}

type DeadLetterEvent struct {
	Event     interface{}
	Reason    string
	Attempts  int
	LastError error
	Timestamp time.Time
}

type DeadLetterQueue struct {
	mu      sync.RWMutex
	events  []DeadLetterEvent
	maxSize int
}

func NewDeadLetterQueue(maxSize int) *DeadLetterQueue {
	return &DeadLetterQueue{
		events:  make([]DeadLetterEvent, 0, maxSize),
		maxSize: maxSize,
	}
}

func (dlq *DeadLetterQueue) Add(event DeadLetterEvent) {
	dlq.mu.Lock()
	defer dlq.mu.Unlock()

	dlq.events = append(dlq.events, event)
	if len(dlq.events) > dlq.maxSize {
		copy(dlq.events, dlq.events[1:])
		dlq.events = dlq.events[:dlq.maxSize]
	}
}

func (dlq *DeadLetterQueue) Len() int {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()
	return len(dlq.events)
}

func (dlq *DeadLetterQueue) GetAll() []DeadLetterEvent {
	dlq.mu.RLock()
	defer dlq.mu.RUnlock()

	result := make([]DeadLetterEvent, len(dlq.events))
	copy(result, dlq.events)
	return result
}

type RetryFunc func(ctx context.Context) error

type Result struct {
	Success  bool
	Attempts int
	Err      error
}

func WithRetry(ctx context.Context, config Config, fn RetryFunc) Result {
	var lastErr error
	delay := config.BaseDelay

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			return Result{
				Success:  true,
				Attempts: attempt + 1,
				Err:      nil,
			}
		}

		lastErr = err

		if ctx.Err() != nil {
			return Result{
				Success:  false,
				Attempts: attempt + 1,
				Err:      fmt.Errorf("retry cancelled: %w", ctx.Err()),
			}
		}

		if attempt < config.MaxAttempts-1 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return Result{
					Success:  false,
					Attempts: attempt + 1,
					Err:      fmt.Errorf("retry cancelled during backoff: %w", ctx.Err()),
				}
			}

			delay = time.Duration(math.Min(
				float64(delay)*config.Multiplier,
				float64(config.MaxDelay),
			))
		}
	}

	return Result{
		Success:  false,
		Attempts: config.MaxAttempts,
		Err:      fmt.Errorf("max retries exceeded: %w", lastErr),
	}
}

type RetryableFunc func(err error) bool

func DefaultRetryable(err error) bool {
	if err == nil {
		return false
	}
	return ctxErr(err) == nil
}

func ctxErr(err error) error {
	switch {
	case err == context.Canceled:
		return context.Canceled
	case err == context.DeadlineExceeded:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

// BatchRetry retries a batch operation. If the batch fails, it falls back
// to retrying individual items.
// fn is the batch function that should be retried.
// itemFn is called for each item if the batch retry fails.
func BatchRetry[T any](ctx context.Context, config Config, items []T, fn func(context.Context, []T) error, itemFn func(context.Context, T) error, dlq *DeadLetterQueue) {
	if len(items) == 0 {
		return
	}

	result := WithRetry(ctx, config, func(c context.Context) error {
		return fn(c, items)
	})

	if result.Success {
		return
	}

	log.Printf("batch retry failed after %d attempts: %v, falling back to individual items", result.Attempts, result.Err)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, item := range items {
		wg.Add(1)
		go func(i T) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			itemResult := WithRetry(ctx, config, func(c context.Context) error {
				return itemFn(c, i)
			})

			if !itemResult.Success {
				dlq.Add(DeadLetterEvent{
					Event:     i,
					Reason:    "max retries exceeded",
					Attempts:  itemResult.Attempts,
					LastError: itemResult.Err,
					Timestamp: time.Now(),
				})
				log.Printf("item retry failed after %d attempts: %v", itemResult.Attempts, itemResult.Err)
			}
		}(item)
	}

	wg.Wait()
}
