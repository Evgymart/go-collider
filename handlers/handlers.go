package handlers

import (
	"collider/database"
	"encoding/json"
	"net/http"
)

type Handlers struct {
	store *database.EventStore
}

func NewHandlers(store *database.EventStore) *Handlers {
	return &Handlers{
		store: store,
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
