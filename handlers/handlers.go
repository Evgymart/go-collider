package handlers

import (
	"collider/database"
	"collider/models"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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

func (h *Handlers) GetEventsPaginated(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit < 1 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	events, err := h.store.GetPaginated(uint(page), uint(limit))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	total, err := h.store.GetTotal()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, errors.New("error fetching total"))
		return
	}

	respondWithJson(w, http.StatusOK, &models.PaginatedEvents{
		Data:  events,
		Limit: uint(limit),
		Page:  uint(page),
		Total: uint(total),
	})
}

func (h *Handlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var inputEvent models.CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&inputEvent); err != nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	eventTypeId, err := h.store.GetOrCreateEventType(inputEvent.Type)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, errors.New("error fetching event type"))
		return
	}

	createdEvent, err := h.store.CreateEvent(models.CreateEventData{
		UserID:   inputEvent.UserID,
		TypeID:   eventTypeId,
		Metadata: inputEvent.Metadata,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	createdEvent.Type = inputEvent.Type
	respondWithJson(w, http.StatusCreated, createdEvent)
}
