package cases_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"collider/database"
	"collider/handlers"
	"collider/models"
	"collider/test/utils"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func setupStatsHandlers(t *testing.T) (*sqlx.DB, *handlers.Handlers) {
	t.Helper()
	db := utils.SetupTestDB(t)
	eventStore := database.NewEventStore(db)
	statsStore := database.NewStatsStore(db)
	h := handlers.NewHandlers(eventStore, statsStore)
	return db, h
}

func createEvent(t *testing.T, h *handlers.Handlers, userID uuid.UUID, eventType string, page string) {
	t.Helper()

	requestBody := models.CreateEventInput{
		UserID:   userID,
		Type:     eventType,
		Metadata: json.RawMessage(`{"page": "` + page + `"}`),
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.CreateEvent(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test event: %s", rr.Body.String())
	}
}

func TestGetStats_Empty(t *testing.T) {
	_, h := setupStatsHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response models.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.TotalEvents != 0 {
		t.Errorf("expected total_events 0, got %d", response.TotalEvents)
	}
	if response.UniqueUsers != 0 {
		t.Errorf("expected unique_users 0, got %d", response.UniqueUsers)
	}

	var topPages map[string]int
	if err := json.Unmarshal(response.TopPages, &topPages); err != nil {
		t.Fatalf("failed to parse top_pages: %v", err)
	}
	if len(topPages) != 0 {
		t.Errorf("expected empty top_pages, got %v", topPages)
	}
}

func TestGetStats_CorrectStats(t *testing.T) {
	db, h := setupStatsHandlers(t)

	userID1 := utils.CreateTestUser(db)
	userID2 := utils.CreateTestUser(db)

	createEvent(t, h, userID1, "page.view", "/home")
	createEvent(t, h, userID1, "page.view", "/about")
	createEvent(t, h, userID2, "page.view", "/home")
	createEvent(t, h, userID2, "user.login", "/login")

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var response models.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.TotalEvents != 4 {
		t.Errorf("expected total_events 4, got %d", response.TotalEvents)
	}
	if response.UniqueUsers != 2 {
		t.Errorf("expected unique_users 2, got %d", response.UniqueUsers)
	}

	var topPages map[string]int
	if err := json.Unmarshal(response.TopPages, &topPages); err != nil {
		t.Fatalf("failed to parse top_pages: %v", err)
	}

	if topPages["/home"] != 2 {
		t.Errorf("expected /home to have 2 events, got %d", topPages["/home"])
	}
	if topPages["/about"] != 1 {
		t.Errorf("expected /about to have 1 event, got %d", topPages["/about"])
	}
	if topPages["/login"] != 1 {
		t.Errorf("expected /login to have 1 event, got %d", topPages["/login"])
	}
}

func TestGetStats_FiltersByEventType(t *testing.T) {
	db, h := setupStatsHandlers(t)

	userID := utils.CreateTestUser(db)

	createEvent(t, h, userID, "page.view", "/home")
	createEvent(t, h, userID, "page.view", "/about")
	createEvent(t, h, userID, "user.login", "/login")

	req := httptest.NewRequest(http.MethodGet, "/stats?type=page.view", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response models.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.TotalEvents != 2 {
		t.Errorf("expected total_events 2 (only page.view events), got %d", response.TotalEvents)
	}

	var topPages map[string]int
	if err := json.Unmarshal(response.TopPages, &topPages); err != nil {
		t.Fatalf("failed to parse top_pages: %v", err)
	}

	if len(topPages) != 2 {
		t.Errorf("expected 2 unique pages, got %d", len(topPages))
	}
}

func TestGetStats_InvalidFromTime(t *testing.T) {
	_, h := setupStatsHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/stats?from=invalid-time", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if response["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestGetStats_InvalidToTime(t *testing.T) {
	_, h := setupStatsHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/stats?to=not-a-time", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if response["error"] == "" {
		t.Error("expected error message in response")
	}
}

func TestGetStats_NonexistentEventType(t *testing.T) {
	_, h := setupStatsHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/stats?type=nonexistent.event", nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	expectedErrMsg := "event type not found"
	if response["error"] != expectedErrMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, response["error"])
	}
}

func TestGetStatsWithTimeRange_ValidRange(t *testing.T) {
	db, h := setupStatsHandlers(t)

	userID := utils.CreateTestUser(db)

	createEvent(t, h, userID, "page.view", "/home")

	from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/stats?from="+url.QueryEscape(from)+"&to="+url.QueryEscape(to), nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, rr.Body.String())
	}

	var response models.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.TotalEvents != 1 {
		t.Errorf("expected total_events 1, got %d", response.TotalEvents)
	}
}

func TestGetStatsWithTimeRange_OnlyFrom(t *testing.T) {
	db, h := setupStatsHandlers(t)

	userID := utils.CreateTestUser(db)

	createEvent(t, h, userID, "page.view", "/home")

	from := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/stats?from="+url.QueryEscape(from), nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response models.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.TotalEvents != 1 {
		t.Errorf("expected total_events 1, got %d", response.TotalEvents)
	}
}

func TestGetStatsWithTimeRange_FiltersOutsideRange(t *testing.T) {
	db, h := setupStatsHandlers(t)

	userID := utils.CreateTestUser(db)

	createEvent(t, h, userID, "page.view", "/home")

	from := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	to := time.Now().Add(48 * time.Hour).Format(time.RFC3339)

	req := httptest.NewRequest(http.MethodGet, "/stats?from="+url.QueryEscape(from)+"&to="+url.QueryEscape(to), nil)
	rr := httptest.NewRecorder()

	h.GetStats(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response models.Stats
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.TotalEvents != 0 {
		t.Errorf("expected total_events 0 (no events in future time range), got %d", response.TotalEvents)
	}
}
