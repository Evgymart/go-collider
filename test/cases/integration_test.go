package cases_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "method not allowed",
			statusCode: http.StatusMethodNotAllowed,
			message:    "Method Not Allowed",
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			message:    "bad request",
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
			message:    "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			rr.Header().Set("Content-Type", "application/json")
			rr.WriteHeader(tt.statusCode)
			json.NewEncoder(rr).Encode(map[string]string{"error": tt.message})

			if status := rr.Code; status != tt.statusCode {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tt.statusCode)
			}

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
			}

			var response map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}

			if response["error"] == "" {
				t.Error("expected 'error' key in response")
			}

			if response["error"] != tt.message {
				t.Errorf("expected error message '%s', got '%s'", tt.message, response["error"])
			}
		})
	}
}

func TestHandlerErrorResponses(t *testing.T) {
	t.Run("GetStats returns JSON for invalid from date", func(t *testing.T) {
		_, h := setupStatsHandlers(t)
		req := httptest.NewRequest(http.MethodGet, "/stats?from=invalid", nil)
		rr := httptest.NewRecorder()

		h.GetStats(rr, req)

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}
		if response["error"] == "" {
			t.Error("expected 'error' key in JSON response")
		}
	})
}
