package handlers

import (
	"collider/internal/cache"
	"collider/internal/models"

	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")
	eventType := query.Get("type")
	var fromTime, toTime *time.Time
	var eventTypeId *int64
	var err error

	if fromStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02T15:04:05Z", fromStr)
			if err != nil {
				respondWithError(w, http.StatusBadRequest, errors.New("invalid date format"))
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
				respondWithError(w, http.StatusBadRequest, errors.New("invalid date format"))
				return
			}
		}
		toTime = &parsedTime
	}

	if eventType != "" {
		eventTypeId, err = h.eventStore.GetEventTypeId(eventType)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, fmt.Errorf("event type '%s' not found: %w", eventType, err))
			return
		}
	}

	if h.cache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
		defer cancel()

		cacheKey := cache.StatsKey(fromTime, toTime, eventTypeId)
		var cachedStats models.Stats

		if h.cache.Get(ctx, cacheKey, &cachedStats) {
			log.Printf("cache hit: %s", cacheKey)
			respondWithJson(w, http.StatusOK, &cachedStats)
			return
		}
	}

	stats, err := h.statsStore.GetStats(models.GetStatsData{
		TypeID: eventTypeId,
		From:   fromTime,
		To:     toTime,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Errorf("failed to get stats: %w", err))
		return
	}

	if h.cache != nil {
		h.cache.Set(context.Background(), cache.StatsKey(fromTime, toTime, eventTypeId), stats, cache.TTLStats)
	}

	respondWithJson(w, http.StatusOK, stats)
}
