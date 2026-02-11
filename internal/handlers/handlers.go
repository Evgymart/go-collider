package handlers

import (
	"collider/internal/stores"

	"encoding/json"
	"net/http"
)

type Handlers struct {
	eventStore *stores.EventStore
	statsStore *stores.StatsStore
}

func NewHandlers(e *stores.EventStore, s *stores.StatsStore) *Handlers {
	return &Handlers{
		eventStore: e,
		statsStore: s,
	}
}

func respondWithJson(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, statusCode int, err error) {
	respondWithJson(w, statusCode, map[string]string{"error": err.Error()})
}
