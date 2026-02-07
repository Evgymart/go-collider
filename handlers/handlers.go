package handlers

import (
	"collider/database"
	"encoding/json"
	"net/http"
)

type Handlers struct {
	eventStore *database.EventStore
	statsStore *database.StatsStore
}

func NewHandlers(e *database.EventStore, s *database.StatsStore) *Handlers {
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
