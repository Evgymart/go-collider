package handlers

import (
	"collider/internal/cache"
	"collider/internal/stores"

	"net/http"
)

type Handlers struct {
	eventStore *stores.EventStore
	statsStore *stores.StatsStore
	cache      *cache.Cache
}

func NewHandlers(e *stores.EventStore, s *stores.StatsStore, c *cache.Cache) *Handlers {
	return &Handlers{
		eventStore: e,
		statsStore: s,
		cache:      c,
	}
}

func respondWithJson(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
}

func respondWithError(w http.ResponseWriter, statusCode int, err error) {
	respondWithJson(w, statusCode, map[string]string{"error": err.Error()})
}
