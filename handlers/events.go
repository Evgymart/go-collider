package handlers

import (
	"collider/models"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

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

	events, err := h.eventStore.GetPaginated(uint(page), uint(limit), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	total, err := h.eventStore.GetTotal(nil)
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

func (h *Handlers) GetUserEventsPaginated(w http.ResponseWriter, r *http.Request) {
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

	path := r.URL.Path
	re := regexp.MustCompile(`^/users/([^/]+)/events$`)
	matches := re.FindStringSubmatch(path)
	if matches == nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid URL format"))
		return
	}
	userIDStr := matches[1]
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	events, err := h.eventStore.GetPaginated(uint(page), uint(limit), &userUUID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	total, err := h.eventStore.GetTotal(&userUUID)
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

	if inputEvent.Type == "" {
		respondWithError(w, http.StatusBadRequest, errors.New("event_type is required"))
		return
	}

	if inputEvent.UserID == uuid.Nil {
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

	createdEvent.Type = inputEvent.Type
	respondWithJson(w, http.StatusCreated, createdEvent)
}
