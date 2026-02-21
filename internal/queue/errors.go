// Package queue provides error types for the event queue system.
package queue

import "errors"

// Queue-related error types for proper error checking with errors.Is().
var (
	// ErrQueueStopped is returned when attempting to enqueue to a stopped queue.
	ErrQueueStopped = errors.New("queue is stopped")

	// ErrQueueFull is returned when the queue has reached its maximum size.
	ErrQueueFull = errors.New("queue full")

	// ErrQueueChannelFull is returned when the in-memory channel is full.
	ErrQueueChannelFull = errors.New("queue channel full")

	// ErrPersistenceClosed is returned when attempting to use closed persistence.
	ErrPersistenceClosed = errors.New("persistence is closed")

	// ErrQueueAlreadyStarted is returned when attempting to start an already started queue.
	ErrQueueAlreadyStarted = errors.New("queue already started")

	// ErrQueueAlreadyStopped is returned when attempting to stop an already stopped queue.
	ErrQueueAlreadyStopped = errors.New("queue already stopped")
)
