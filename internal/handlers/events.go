package handlers

import (
	"collider/internal/cache"
	"collider/internal/models"
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

	// Fetch from database on cache miss
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

	// Invalidate cache entries after creating an event
	// Fire-and-forget is acceptable here as cache invalidation is quick
	// and eventual consistency is sufficient for this use case.
	if h.cache != nil {
		go func() {
			ctx := context.Background()
			h.cache.InvalidateUserEvents(ctx, inputEvent.UserID)
			h.cache.InvalidateStats(ctx)
		}()
	}

	// Note: We don't invalidate global events cache on every event creation
	// because the pagination would only shift for newly created events,
	// and stale data for 5 minutes is acceptable for the use case.

	createdEvent.Type = inputEvent.Type
	respondWithJson(w, http.StatusCreated, createdEvent)
}
