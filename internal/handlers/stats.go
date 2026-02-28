package handlers

import (
	"collider/internal/models"

	"errors"
	"fmt"
	"net/http"
	"time"
)

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")
	eventType := query.Get("type")
	var eventTypeId *int64
	var err error

	fromTime, err := parseTimestamp(fromStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	toTime, err := parseTimestamp(toStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return
	}

	if eventType != "" {
		eventTypeId, err = h.statsRepository.GetEventTypeId(eventType)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("event type '%s' not found: %w", eventType, err))
			return
		}
	}

	data := models.GetStatsData{
		TypeID: eventTypeId,
		From:   fromTime,
		To:     toTime,
	}

	if cachedBytes, hit := h.statsRepository.GetStatsRaw(r.Context(), data); hit {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(cachedBytes)
		return
	}

	stats, err := h.statsRepository.GetStats(r.Context(), data)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJson(w, http.StatusOK, stats)
}

func parseTimestamp(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, s)
	if err != nil {
		parsedTime, err = time.Parse("2006-01-02T15:04:05Z", s)
		if err != nil {
			return nil, errors.New("invalid date format")
		}
	}
	return &parsedTime, nil
}
