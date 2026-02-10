package cases_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"collider/database"
	"collider/handlers"
	"collider/models"
	"collider/test/utils"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func setupHandlers(t *testing.T) (*sqlx.DB, *handlers.Handlers) {
	t.Helper()
	db := utils.SetupTestDB(t)
	eventStore := database.NewEventStore(db)
	statsStore := database.NewStatsStore(db)
	h := handlers.NewHandlers(eventStore, statsStore)
	return db, h
}

func TestCreateEvent(t *testing.T) {
	db, h := setupHandlers(t)

	t.Run("creates event successfully with valid input", func(t *testing.T) {
		userID := utils.CreateTestUser(db)

		requestBody := models.CreateEventInput{
			UserID:   userID,
			Type:     "user.login",
			Metadata: json.RawMessage(`{"ip": "192.168.1.1", "device": "mobile", "page": "/login"}`),
		}
		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusCreated, rr.Body.String())
		}

		var response models.Event
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response.UserID != userID {
			t.Errorf("expected user_id %v, got %v", userID, response.UserID)
		}
		if response.Type != "user.login" {
			t.Errorf("expected type 'user.login', got '%s'", response.Type)
		}
		if response.ID == uuid.Nil {
			t.Error("expected non-empty event ID")
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(response.Metadata, &metadata); err != nil {
			t.Fatalf("failed to parse metadata: %v", err)
		}
		if metadata["ip"] != "192.168.1.1" {
			t.Errorf("expected ip '192.168.1.1', got '%v'", metadata["ip"])
		}
		if metadata["device"] != "mobile" {
			t.Errorf("expected device 'mobile', got '%v'", metadata["device"])
		}
		if metadata["page"] != "/login" {
			t.Errorf("expected page '/login', got '%v'", metadata["page"])
		}
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusBadRequest, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}
		if response["error"] == "" {
			t.Error("expected error message in response")
		}
	})

	t.Run("returns 400 for non-existent user", func(t *testing.T) {
		newEventType := "transaction.test.event"
		requestBody := models.CreateEventInput{
			UserID:   uuid.New(),
			Type:     newEventType,
			Metadata: json.RawMessage(`{"page": "/login"}`),
		}
		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusBadRequest, rr.Body.String())
		}

		var typeID *uuid.UUID
		err = db.Get(&typeID, "select type_id from event_types where name = $1", newEventType)
		if err == nil && typeID != nil {
			t.Errorf("event type should not be created when event creation fails")
		}
	})

	t.Run("creates event with empty metadata", func(t *testing.T) {
		userID := utils.CreateTestUser(db)

		requestBody := models.CreateEventInput{
			UserID:   userID,
			Type:     "user.logout",
			Metadata: json.RawMessage(`{}`),
		}
		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusCreated, rr.Body.String())
		}

		var response models.Event
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response.Type != "user.logout" {
			t.Errorf("expected type 'user.logout', got '%s'", response.Type)
		}
	})

	t.Run("creates event and reuses existing event type", func(t *testing.T) {
		userID := utils.CreateTestUser(db)

		requestBody1 := models.CreateEventInput{
			UserID:   userID,
			Type:     "page.view",
			Metadata: json.RawMessage(`{"page": "/home"}`),
		}
		body1, _ := json.Marshal(requestBody1)
		req1 := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body1))
		req1.Header.Set("Content-Type", "application/json")
		rr1 := httptest.NewRecorder()
		h.CreateEvent(rr1, req1)

		if status := rr1.Code; status != http.StatusCreated {
			t.Errorf("first event: got %v want %v", status, http.StatusCreated)
		}

		requestBody2 := models.CreateEventInput{
			UserID:   userID,
			Type:     "page.view",
			Metadata: json.RawMessage(`{"page": "/about"}`),
		}
		body2, _ := json.Marshal(requestBody2)
		req2 := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body2))
		req2.Header.Set("Content-Type", "application/json")
		rr2 := httptest.NewRecorder()
		h.CreateEvent(rr2, req2)

		if status := rr2.Code; status != http.StatusCreated {
			t.Errorf("second event: got %v want %v", status, http.StatusCreated)
		}

		var response1, response2 models.Event
		json.Unmarshal(rr1.Body.Bytes(), &response1)
		json.Unmarshal(rr2.Body.Bytes(), &response2)

		if response1.TypeID != response2.TypeID {
			t.Errorf("expected same type_id for same event type, got %v and %v",
				response1.TypeID, response2.TypeID)
		}
	})

	t.Run("returns 400 for empty event_type", func(t *testing.T) {
		userID := utils.CreateTestUser(db)

		requestBody := models.CreateEventInput{
			UserID:   userID,
			Type:     "",
			Metadata: json.RawMessage(`{"page": "/login"}`),
		}
		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusBadRequest, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}
		expectedErrMsg := "event_type is required"
		if response["error"] != expectedErrMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, response["error"])
		}
	})

	t.Run("returns 400 for invalid UUID user_id", func(t *testing.T) {
		requestBody := models.CreateEventInput{
			UserID:   uuid.Nil,
			Type:     "test.event",
			Metadata: json.RawMessage(`{"page": "/test"}`),
		}
		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusBadRequest, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}
		expectedErrMsg := "invalid user id"
		if response["error"] != expectedErrMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, response["error"])
		}
	})

	t.Run("creates event with null metadata defaults to empty object", func(t *testing.T) {
		userID := utils.CreateTestUser(db)

		requestBody := models.CreateEventInput{
			UserID: userID,
			Type:   "test.metadata.null",
		}
		body, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		h.CreateEvent(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusCreated, rr.Body.String())
		}

		var response models.Event
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(response.Metadata, &metadata); err != nil {
			t.Fatalf("failed to parse metadata: %v", err)
		}
		if len(metadata) != 0 {
			t.Errorf("expected empty metadata object, got %v", metadata)
		}
	})
}

func TestGetEventsPaginated(t *testing.T) {
	db, h := setupHandlers(t)

	t.Run("returns empty list when no events exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		rr := httptest.NewRecorder()

		h.GetEventsPaginated(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var response models.PaginatedEvents
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if len(response.Data) != 0 {
			t.Errorf("expected empty data, got %d events", len(response.Data))
		}
		if response.Total != 0 {
			t.Errorf("expected total 0, got %d", response.Total)
		}
	})

	t.Run("returns paginated events", func(t *testing.T) {
		userID := utils.CreateTestUser(db)

		for i := 0; i < 3; i++ {
			requestBody := models.CreateEventInput{
				UserID:   userID,
				Type:     "test.event",
				Metadata: json.RawMessage(`{"index": ` + string(rune('0'+i)) + `}`),
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

		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		rr := httptest.NewRecorder()

		h.GetEventsPaginated(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}

		var response models.PaginatedEvents
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response.Total != 3 {
			t.Errorf("expected total 3, got %d", response.Total)
		}
		if len(response.Data) != 3 {
			t.Errorf("expected 3 events, got %d", len(response.Data))
		}
		if response.Page != 1 {
			t.Errorf("expected page 1, got %d", response.Page)
		}
		if response.Limit != 20 {
			t.Errorf("expected limit 20, got %d", response.Limit)
		}
	})
}

func TestGetUserEventsPaginated(t *testing.T) {
	_, h := setupHandlers(t)

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/not-a-uuid/events", nil)
		rr := httptest.NewRecorder()

		h.GetUserEventsPaginated(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s",
				status, http.StatusBadRequest, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse error response: %v", err)
		}
		expectedErrMsg := "invalid user id"
		if response["error"] != expectedErrMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, response["error"])
		}
	})
}
