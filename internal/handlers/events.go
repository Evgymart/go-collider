package handlers

import (
	"collider/internal/models"
	"collider/internal/queue"
	"collider/pkg/pagination"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/lib/pq"
)

func (h *Handlers) GetEventsPaginated(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)
	ctx := r.Context()

	response, err := h.eventRepository.GetPaginatedCached(ctx, params.Page, params.Limit, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	// Use pass-through caching if we have raw JSON bytes
	if response.IsCached() {
		respondWithJsonBytes(w, http.StatusOK, response.RawJSON)
		return
	}

	respondWithJson(w, http.StatusOK, response.Data)
}

func (h *Handlers) GetUserEventsPaginated(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)

	path := r.URL.Path
	re := regexp.MustCompile(`^/users/([^/]+)/events$`)
	matches := re.FindStringSubmatch(path)
	if matches == nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid URL format"))
		return
	}
	userIDStr := matches[1]
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	if userID <= 0 {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	ctx := r.Context()
	response, err := h.eventRepository.GetPaginatedCached(ctx, params.Page, params.Limit, &userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	// Use pass-through caching if we have raw JSON bytes
	if response.IsCached() {
		respondWithJsonBytes(w, http.StatusOK, response.RawJSON)
		return
	}

	respondWithJson(w, http.StatusOK, response.Data)
}

func (h *Handlers) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var inputEvent models.CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&inputEvent); err != nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	if inputEvent.Type == "" {
		respondWithError(w, http.StatusBadRequest, errors.New("event_type is required"))
		return
	}

	if inputEvent.UserID <= 0 {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	if h.eventQueue != nil {
		h.createEventAsync(w, r, inputEvent)
		return
	}

	h.createEventSync(w, r, inputEvent)
}

func (h *Handlers) createEventAsync(w http.ResponseWriter, r *http.Request, inputEvent models.CreateEventInput) {
	eventID, err := h.eventRepository.GenerateEventID()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to generate event ID: %w", err))
		return
	}

	metadata := inputEvent.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	result := h.eventQueue.Enqueue(inputEvent.UserID, inputEvent.Type, metadata, eventID)

	if result.Error != nil {
		if errors.Is(result.Error, queue.ErrQueueFull) || errors.Is(result.Error, queue.ErrQueueChannelFull) {
			h.insertEventSyncAndRespond(w, r.Context(), inputEvent, eventID, metadata)
			return
		}
		respondWithError(w, http.StatusInternalServerError, result.Error)
		return
	}

	ctx := r.Context()
	h.eventRepository.InvalidateCaches(ctx)

	response := models.EventData{
		Data: models.Event{
			ID:        result.EventID,
			UserID:    inputEvent.UserID,
			TypeID:    0, // Will be filled by the worker
			Type:      inputEvent.Type,
			Metadata:  metadata,
			Timestamp: result.QueuedAt,
		},
	}
	respondWithJson(w, http.StatusAccepted, response)
}

func (h *Handlers) insertEventSyncAndRespond(w http.ResponseWriter, ctx context.Context, inputEvent models.CreateEventInput, eventID int64, metadata []byte) {
	typeID, err := h.eventRepository.GetOrCreateTypeID(inputEvent.Type)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to get type ID: %w", err))
		return
	}

	event := models.Event{
		ID:        eventID,
		UserID:    inputEvent.UserID,
		TypeID:    typeID,
		Type:      inputEvent.Type,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}

	if err := h.eventRepository.InsertEventWithID(ctx, event); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			respondWithError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	response := models.EventData{
		Data: event,
	}
	respondWithJson(w, http.StatusAccepted, response)
}

func (h *Handlers) createEventSync(w http.ResponseWriter, r *http.Request, inputEvent models.CreateEventInput) {
	ctx := r.Context()
	createdEvent, err := h.eventRepository.CreateEventWithType(
		ctx,
		inputEvent.UserID,
		inputEvent.Type,
		inputEvent.Metadata,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			respondWithError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJson(w, http.StatusCreated, createdEvent)
}
