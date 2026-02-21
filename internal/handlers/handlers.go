package handlers

import (
	"collider/internal/cache"
	"collider/internal/queue"
	"collider/internal/stores"

	"encoding/json"
	"net/http"
)

type Handlers struct {
	eventStore *stores.EventStore
	statsStore *stores.StatsStore
	cache      *cache.Cache
	eventQueue *queue.EventQueue
}

func NewHandlers(e *stores.EventStore, s *stores.StatsStore, c *cache.Cache) *Handlers {
	return &Handlers{
		eventStore: e,
		statsStore: s,
		cache:      c,
		eventQueue: nil, // Will be set after queue is created
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

func respondWithError(w http.ResponseWriter, statusCode int, err error) {
	respondWithJson(w, statusCode, map[string]string{"error": err.Error()})
}
