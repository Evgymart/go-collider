package handlers

import (
	"collider/internal/cache"
	"collider/internal/models"
	"collider/internal/queue"
	"collider/pkg/pagination"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/lib/pq"
)

func (h *Handlers) GetEventsPaginated(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()

		cacheKey := cache.EventsKey(params.Page, params.Limit)
		var cachedResponse models.PaginatedEvents

		if h.cache.Get(ctx, cacheKey, &cachedResponse) {
			log.Printf("cache hit: %s", cacheKey)
			respondWithJson(w, http.StatusOK, &cachedResponse)
			return
		}
	}

	events, err := h.eventStore.GetPaginated(params.Page, params.Limit, nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to get paginated events: %w", err))
		return
	}

	total, err := h.eventStore.GetTotal(nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to get total events: %w", err))
		return
	}

	response := &models.PaginatedEvents{
		Data:  events,
		Limit: params.Limit,
		Page:  params.Page,
		Total: uint(total),
	}

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()
		h.cache.Set(ctx, cache.EventsKey(params.Page, params.Limit), response, cache.TTLEvents)
	}

	respondWithJson(w, http.StatusOK, response)
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

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()

		cacheKey := cache.UserEventsKey(userID, params.Page, params.Limit)
		var cachedResponse models.PaginatedEvents

		if h.cache.Get(ctx, cacheKey, &cachedResponse) {
			log.Printf("cache hit: %s", cacheKey)
			respondWithJson(w, http.StatusOK, &cachedResponse)
			return
		}
	}

	events, err := h.eventStore.GetPaginated(params.Page, params.Limit, &userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to get user events: %w", err))
		return
	}

	total, err := h.eventStore.GetTotal(&userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to get total user events: %w", err))
		return
	}

	response := &models.PaginatedEvents{
		Data:  events,
		Limit: params.Limit,
		Page:  params.Page,
		Total: uint(total),
	}

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()
		h.cache.Set(ctx, cache.UserEventsKey(userID, params.Page, params.Limit), response, cache.TTLUserEvents)
	}

	respondWithJson(w, http.StatusOK, response)
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
	eventID, err := h.eventStore.GenerateEventID()
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
			log.Printf("queue: full, falling back to sync insert for event %d", eventID)
			h.insertEventSyncAndRespond(w, inputEvent, eventID, metadata)
			return
		}
		respondWithError(w, http.StatusInternalServerError, result.Error)
		return
	}

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
		defer cancel()
		h.cache.InvalidateUserEvents(ctx, inputEvent.UserID)
		h.cache.InvalidateStats(ctx)
	}

	typeID, err := h.eventStore.GetOrCreateTypeID(inputEvent.Type)
	if err != nil {
		typeID = 0
	}

	response := models.EventData{
		Data: models.Event{
			ID:        result.EventID,
			UserID:    inputEvent.UserID,
			TypeID:    typeID,
			Type:      inputEvent.Type,
			Metadata:  metadata,
			Timestamp: result.QueuedAt,
		},
	}
	respondWithJson(w, http.StatusAccepted, response)
}

func (h *Handlers) insertEventSyncAndRespond(w http.ResponseWriter, inputEvent models.CreateEventInput, eventID int64, metadata []byte) {
	typeID, err := h.eventStore.GetOrCreateTypeID(inputEvent.Type)
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

	if err := h.eventStore.InsertEventWithID(event); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			respondWithError(w, http.StatusBadRequest, errors.New("invalid user_id"))
			return
		}
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		h.cache.InvalidateUserEvents(ctx, inputEvent.UserID)
		h.cache.InvalidateStats(ctx)
	}

	response := models.EventData{
		Data: event,
	}
	respondWithJson(w, http.StatusAccepted, response)
}

func (h *Handlers) createEventSync(w http.ResponseWriter, r *http.Request, inputEvent models.CreateEventInput) {
	createdEvent, err := h.eventStore.CreateEventWithType(
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

	if h.cache != nil {
		ctx := context.Background()
		h.cache.InvalidateUserEvents(ctx, inputEvent.UserID)
		h.cache.InvalidateStats(ctx)
	}

	createdEvent.Type = inputEvent.Type
	respondWithJson(w, http.StatusCreated, createdEvent)
}
