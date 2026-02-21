// Package queue provides persistent queue storage for event batching.
// It uses BoltDB as an embedded key-value store for write-ahead logging,
// ensuring events are not lost on crashes or restarts.
package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

type PendingEvent struct {
	EventID int64 `json:"event_id"`

	UserID int64 `json:"user_id"`

	EventType string `json:"event_type"`

	Metadata []byte `json:"metadata"`

	EnqueuedAt int64 `json:"enqueued_at"`
}

// writeRequest represents a pending write operation.
type writeRequest struct {
	event PendingEvent
	errCh chan error // Send error back to caller (nil on success)
}

// Persistence provides durable storage for pending events using BoltDB.
// Events are written to disk asynchronously by a background goroutine.
// Trade-off: Small window of data loss on crash is acceptable for event tracking.
type Persistence struct {
	mu     sync.RWMutex
	db     *bbolt.DB
	path   string
	closed bool

	writeCh chan writeRequest
	wg      sync.WaitGroup
}

const (
	bucketName = "pending_events"
)

const (
	writeBufferSize = 1000
)

func NewPersistence(path string) (*Persistence, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create persistence directory: %w", err)
	}

	db, err := bbolt.Open(path, 0600, &bbolt.Options{
		Timeout:      time.Second,
		NoGrowSync:   false,
		FreelistType: bbolt.FreelistArrayType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open persistence database: %w", err)
	}

	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	p := &Persistence{
		db:      db,
		path:    path,
		closed:  false,
		writeCh: make(chan writeRequest, writeBufferSize),
	}

	p.wg.Add(1)
	go p.writeLoop()

	log.Printf("persistence: initialized at %s (async writes)", path)

	return p, nil
}

func (p *Persistence) writeLoop() {
	defer p.wg.Done()

	for req := range p.writeCh {
		p.mu.RLock()
		closed := p.closed
		p.mu.RUnlock()

		if closed {
			req.errCh <- fmt.Errorf("persistence is closed")
			close(req.errCh)
			continue
		}

		err := p.db.Update(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(bucketName))
			if bucket == nil {
				return fmt.Errorf("bucket not found")
			}
			key := fmt.Sprintf("%d", req.event.EventID)
			value, err := json.Marshal(req.event)
			if err != nil {
				return fmt.Errorf("failed to marshal event: %w", err)
			}
			return bucket.Put([]byte(key), value)
		})

		req.errCh <- err
		close(req.errCh)
	}
}

func (p *Persistence) EnqueueAsync(event PendingEvent) chan error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	errCh := make(chan error, 1)

	if p.closed {
		errCh <- fmt.Errorf("persistence is closed")
		close(errCh)
		return errCh
	}

	select {
	case p.writeCh <- writeRequest{event: event, errCh: errCh}:
	default:
		errCh <- fmt.Errorf("persistence write buffer full")
		close(errCh)
	}

	return errCh
}

// Enqueue writes an event to persistent storage synchronously.
// Deprecated: Use EnqueueAsync for better performance.
func (p *Persistence) Enqueue(event PendingEvent) error {
	errCh := p.EnqueueAsync(event)
	return <-errCh
}

func (p *Persistence) Recover() ([]PendingEvent, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return nil, fmt.Errorf("persistence is closed")
	}

	var events []PendingEvent

	err := p.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var event PendingEvent
			if err := json.Unmarshal(v, &event); err != nil {
				log.Printf("persistence: failed to unmarshal event %s: %v", string(k), err)
				return nil
			}
			events = append(events, event)
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("failed to recover events: %w", err)
	}

	log.Printf("persistence: recovered %d events", len(events))
	return events, nil
}

func (p *Persistence) Remove(eventIDs []int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("persistence is closed")
	}

	if len(eventIDs) == 0 {
		return nil
	}

	err := p.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}

		for _, id := range eventIDs {
			key := fmt.Sprintf("%d", id)
			if err := bucket.Delete([]byte(key)); err != nil {
				log.Printf("persistence: failed to remove event %s: %v", key, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to remove events: %w", err)
	}

	return nil
}

// Count returns the number of pending events in persistent storage.
func (p *Persistence) Count() (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, fmt.Errorf("persistence is closed")
	}

	var count int

	err := p.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("bucket not found")
		}
		count = bucket.Stats().KeyN
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

func (p *Persistence) Close() error {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()
		return nil
	}

	p.closed = true
	p.mu.Unlock()

	close(p.writeCh)

	p.wg.Wait()

	if err := p.db.Close(); err != nil {
		return fmt.Errorf("failed to close persistence: %w", err)
	}

	log.Printf("persistence: closed %s", p.path)
	return nil
}

type PersistenceStats struct {
	Path         string
	PendingCount int
	DatabaseSize int64
}

func (p *Persistence) GetStats() (PersistenceStats, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return PersistenceStats{}, fmt.Errorf("persistence is closed")
	}

	count, err := p.Count()
	if err != nil {
		return PersistenceStats{}, err
	}

	fileInfo, err := os.Stat(p.path)
	if err != nil {
		return PersistenceStats{}, fmt.Errorf("failed to stat database file: %w", err)
	}

	return PersistenceStats{
		Path:         p.path,
		PendingCount: count,
		DatabaseSize: fileInfo.Size(),
	}, nil
}
