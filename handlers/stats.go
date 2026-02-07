package handlers

import (
	"collider/models"
	"net/http"
	"time"
)

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	fromStr := query.Get("from")
	toStr := query.Get("to")
	var fromTime, toTime time.Time
	var err error

	if fromStr != "" {
		fromTime, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			fromTime, err = time.Parse("2006-01-02T15:04:05Z", fromStr)
			if err != nil {
				http.Error(w, "Invalid from time format", http.StatusBadRequest)
				return
			}
		}
	}

	if toStr != "" {
		toTime, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			toTime, err = time.Parse("2006-01-02T15:04:05Z", toStr)
			if err != nil {
				http.Error(w, "Invalid to time format", http.StatusBadRequest)
				return
			}
		}
	}

	stats, err := h.statsStore.GetStats(models.GetStatsData{
		TypeID: nil,
		From:   &fromTime,
		To:     &toTime,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err)
		return
	}

	respondWithJson(w, http.StatusOK, stats)
}
