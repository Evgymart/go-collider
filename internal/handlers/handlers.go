package handlers

import (
	"collider/internal/queue"
	"collider/internal/repositories"

	"encoding/json"
	"net/http"
)

type Handlers struct {
	eventRepository repositories.EventRepository
	statsRepository repositories.StatsRepository
	eventQueue      *queue.EventQueue
}

func NewHandlers(e repositories.EventRepository, s repositories.StatsRepository) *Handlers {
	return &Handlers{
		eventRepository: e,
		statsRepository: s,
		eventQueue:      nil, // Will be set after queue is created
	}
}

func (h *Handlers) SetEventQueue(eq *queue.EventQueue) {
	h.eventQueue = eq
}

func respondWithJson(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// Log the error but we can't send another response since headers are already sent
		// In practice, this typically means the client disconnected
	}
}

// respondWithJsonBytes writes pre-serialized JSON bytes directly to the response.
// This is used for pass-through caching to avoid double JSON serialization.
func respondWithJsonBytes(w http.ResponseWriter, statusCode int, payload []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(payload); err != nil {
		// Log the error but we can't send another response since headers are already sent
		// In practice, this typically means the client disconnected
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, err error) {
	respondWithJson(w, statusCode, map[string]string{"error": err.Error()})
}
