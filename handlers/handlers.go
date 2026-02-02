package handlers

import (
	"collider/database"
	"encoding/json"
	"errors"
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

func (h *Handlers) GetAllEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.store.GetAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, errors.New("error fetching events"))
		return
	}

	respondWithJson(w, http.StatusOK, events)
}
