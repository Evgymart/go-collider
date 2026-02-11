package handlers

import (
	"collider/internal/models"
	"collider/pkg/pagination"

	"encoding/json"
	"errors"
	"net/http"
	"regexp"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (h *Handlers) GetEventsPaginated(w http.ResponseWriter, r *http.Request) {
	params := pagination.ParseFromRequest(r)

	events, err := h.eventStore.GetPaginated(params.Page, params.Limit, nil)
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
		Limit: params.Limit,
		Page:  params.Page,
		Total: uint(total),
	})
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
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, errors.New("invalid user id"))
		return
	}

	events, err := h.eventStore.GetPaginated(params.Page, params.Limit, &userUUID)
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
		Limit: params.Limit,
		Page:  params.Page,
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
