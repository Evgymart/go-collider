package handlers

import (
	"collider/models"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")
	eventType := query.Get("type")
	var fromTime, toTime *time.Time
	var eventTypeId *uuid.UUID
	var err error

	if fromStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02T15:04:05Z", fromStr)
			if err != nil {
				http.Error(w, "Invalid from time format", http.StatusBadRequest)
				return
			}
		}
		fromTime = &parsedTime
	}

	if toStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02T15:04:05Z", toStr)
			if err != nil {
				http.Error(w, "Invalid to time format", http.StatusBadRequest)
				return
			}
		}
		toTime = &parsedTime
	}

	if eventType != "" {
		eventTypeId, err = h.eventStore.GetEvetTypeId(eventType)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, errors.New("no event by that id"))
			return
		}
	}

	stats, err := h.statsStore.GetStats(models.GetStatsData{
		TypeID: eventTypeId,
		From:   fromTime,
		To:     toTime,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJson(w, http.StatusOK, stats)
}
