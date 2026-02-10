# API Specification: Go-Collider

## Overview

Go-collider is a lightweight event tracking and analytics API built with Go and PostgreSQL. It provides endpoints for creating events, retrieving paginated event lists, and getting analytics statistics.

**Base URL:** `http://localhost:8080`

**Authentication:** None (all endpoints are publicly accessible)

---

## Endpoints

### 1. GET /events

Retrieve a paginated list of all events.

**Method:** GET

**Path:** `/events`

**Authentication:** None

**Query Parameters:**

| Name | Type | Required | Default | Constraints | Description |
|------|------|----------|---------|-------------|-------------|
| page | integer | No | 1 | Must be >= 1 | Page number for pagination |
| limit | integer | No | 20 | 1-100 | Items per page (max 100) |

**Response:**

**Status Code:** 200 OK

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "type_id": "550e8400-e29b-41d4-a716-446655440002",
      "timestamp": "2024-01-15T10:30:00Z",
      "metadata": {"ip": "192.168.1.1", "user_agent": "Mozilla/5.0..."},
      "type": "user.login"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 100
}
```

**Error Responses:**

| Status Code | Description | Example Response |
|-------------|-------------|------------------|
| 500 | Internal Server Error | `{"error": "failed to query events"}` |

**Example cURL:**

```bash
curl -X GET "http://localhost:8080/events?page=1&limit=20"
```

---

### 2. POST /events

Create a new event. The event type is automatically created if it doesn't exist.

**Method:** POST

**Path:** `/events`

**Authentication:** None

**Request Body:**

| Name | Type | Required | Description | Constraints |
|------|------|----------|-------------|-------------|
| user_id | string (uuid) | Yes | ID of the user associated with the event | Must exist in users table |
| event_type | string | Yes | Name/type of the event | Max 255 characters |
| metadata | object | No | Additional event data as JSON | Any valid JSON object |

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "event_type": "user.login",
  "metadata": {
    "ip": "192.168.1.1",
    "user_agent": "Mozilla/5.0",
    "referrer": "https://example.com"
  }
}
```

**Response:**

**Status Code:** 201 Created

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "550e8400-e29b-41d4-a716-446655440001",
  "type_id": "550e8400-e29b-41d4-a716-446655440002",
  "timestamp": "2024-01-15T10:30:00Z",
  "metadata": {"ip": "192.168.1.1", "user_agent": "Mozilla/5.0"},
  "type": "user.login"
}
```

**Error Responses:**

| Status Code | Description | Example Response |
|-------------|-------------|------------------|
| 400 | Bad Request - Invalid JSON | `{"error": "invalid request body"}` |
| 400 | Bad Request - Invalid UUID format | `{"error": "invalid user id"}` |
| 400 | Bad Request - User does not exist | `{"error": "invalid user_id"}` |
| 500 | Internal Server Error | `{"error": "failed to create event"}` |

**Example cURL:**

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "550e8400-e29b-41d4-a716-446655440001",
    "event_type": "user.login",
    "metadata": {"ip": "192.168.1.1"}
  }'
```

---

### 3. GET /users/{id}/events

Retrieve a paginated list of events for a specific user.

**Method:** GET

**Path:** `/users/{id}/events`

**Authentication:** None

**Path Parameters:**

| Name | Type | Required | Description | Constraints |
|------|------|----------|-------------|-------------|
| id | string (uuid) | Yes | User ID to filter events by | Valid UUID format |

**Query Parameters:**

| Name | Type | Required | Default | Constraints | Description |
|------|------|----------|---------|-------------|-------------|
| page | integer | No | 1 | Must be >= 1 | Page number for pagination |
| limit | integer | No | 20 | 1-100 | Items per page (max 100) |

**Response:**

**Status Code:** 200 OK

```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "type_id": "550e8400-e29b-41d4-a716-446655440002",
      "timestamp": "2024-01-15T10:30:00Z",
      "metadata": {"ip": "192.168.1.1"},
      "type": "user.login"
    }
  ],
  "page": 1,
  "limit": 20,
  "total": 50
}
```

**Error Responses:**

| Status Code | Description | Example Response |
|-------------|-------------|------------------|
| 400 | Bad Request - Invalid URL format | `{"error": "invalid url"}` |
| 400 | Bad Request - Invalid UUID format | `{"error": "invalid user id"}` |
| 500 | Internal Server Error | `{"error": "failed to query user events"}` |

**Example cURL:**

```bash
curl -X GET "http://localhost:8080/users/550e8400-e29b-41d4-a716-446655440001/events?page=1&limit=20"
```

---

### 4. GET /stats

Get analytics statistics for events, optionally filtered by date range and event type.

**Method:** GET

**Path:** `/stats`

**Authentication:** None

**Query Parameters:**

| Name | Type | Required | Description | Constraints |
|------|------|----------|-------------|-------------|
| from | string (datetime) | No | Start date filter | RFC3339 format (e.g., `2024-01-01T00:00:00Z`) |
| to | string (datetime) | No | End date filter | RFC3339 format (e.g., `2024-01-31T23:59:59Z`) |
| type | string | No | Filter by event type name | Must exist in event_types table |

**Response:**

**Status Code:** 200 OK

```json
{
  "total_events": 1000,
  "unique_users": 150,
  "top_pages": {
    "/home": 500,
    "/dashboard": 300,
    "/settings": 200
  }
}
```

**Error Responses:**

| Status Code | Description | Example Response |
|-------------|-------------|------------------|
| 400 | Bad Request - Invalid date format | `{"error": "invalid date format"}` |
| 400 | Bad Request - Event type not found | `{"error": "event type not found"}` |
| 500 | Internal Server Error | `{"error": "failed to get stats"}` |

**Example cURL:**

```bash
# All stats
curl -X GET "http://localhost:8080/stats"

# Filtered by date range
curl -X GET "http://localhost:8080/stats?from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z"

# Filtered by event type
curl -X GET "http://localhost:8080/stats?type=user.login"
```

---

## Data Models

### Event

| Field | Type | Description |
|-------|------|-------------|
| id | string (uuid) | Unique event identifier |
| user_id | string (uuid) | ID of the user who triggered the event |
| type_id | string (uuid) | ID of the event type |
| timestamp | string (datetime) | When the event occurred (ISO 8601) |
| metadata | object | Additional event data as JSON |
| type | string | Name of the event type |

### PaginatedEvents

| Field | Type | Description |
|-------|------|-------------|
| data | array of Event | Array of event objects |
| page | integer | Current page number |
| limit | integer | Items per page |
| total | integer | Total number of events |

### CreateEventInput

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| user_id | string (uuid) | Yes | ID of the user |
| event_type | string | Yes | Name of the event type |
| metadata | object | No | Additional event data |

### Stats

| Field | Type | Description |
|-------|------|-------------|
| total_events | integer | Total number of events |
| unique_users | integer | Count of unique users |
| top_pages | object | Top pages with event counts |

---

## Type Reference

### Go Type to JSON Type Mappings

| Go Type | JSON Type | Example |
|---------|-----------|---------|
| string | string | `"hello"` |
| int, int32, int64 | number | `42` |
| uint | number | `20` |
| bool | boolean | `true` |
| uuid.UUID | string (uuid format) | `"550e8400-e29b-41d4-a716-446655440000"` |
| time.Time | string (ISO 8601) | `"2024-01-15T10:30:00Z"` |
| json.RawMessage | object | `{"key": "value"}` |

---

## Validation Rules

### UUID Validation
All UUID fields must be valid UUID v4 format:
- Example: `550e8400-e29b-41d4-a716-446655440000`

### Pagination
- `page`: Must be >= 1 (default: 1)
- `limit`: Must be between 1-100 (default: 20)

### Date Formats
Supported formats for `from` and `to` parameters:
- RFC3339: `2024-01-01T00:00:00Z`
- Go reference format: `2006-01-02T15:04:05Z`

### Event Creation
- `user_id` must exist in the users table
- `event_type` is automatically created in event_types table if it doesn't exist
- `metadata` can be any valid JSON object (stored as JSONB)

---

## Database Schema Reference

### users
| Column | Type | Constraints |
|--------|------|-------------|
| user_id | uuid | PRIMARY KEY, DEFAULT gen_random_uuid() |
| name | varchar(50) | NOT NULL |
| created_at | timestamp | NOT NULL, DEFAULT NOW() |

### event_types
| Column | Type | Constraints |
|--------|------|-------------|
| type_id | uuid | PRIMARY KEY, DEFAULT gen_random_uuid() |
| name | varchar(255) | NOT NULL, UNIQUE |

### events
| Column | Type | Constraints |
|--------|------|-------------|
| event_id | uuid | PRIMARY KEY, DEFAULT gen_random_uuid() |
| user_id | uuid | NOT NULL, FOREIGN KEY → users(user_id) |
| type_id | uuid | NOT NULL, FOREIGN KEY → event_types(type_id) |
| timestamp | timestamp | NOT NULL, DEFAULT NOW() |
| metadata | jsonb | NOT NULL |

---

## Testing Checklist

### GET /events
- [ ] Valid request returns 200 with paginated events
- [ ] Page parameter filters correctly
- [ ] Limit parameter respects max of 100
- [ ] Default pagination values work (page=1, limit=20)
- [ ] Empty result set returns empty data array

### POST /events
- [ ] Valid request returns 201 with created event
- [ ] Missing user_id returns 400
- [ ] Missing event_type returns 400
- [ ] Invalid UUID format for user_id returns 400
- [ ] Non-existent user_id returns 400
- [ ] Metadata with valid JSON is accepted
- [ ] New event_type is created automatically
- [ ] Existing event_type is reused
- [ ] Timestamp is auto-generated

### GET /users/{id}/events
- [ ] Valid request returns 200 with user's events
- [ ] Invalid UUID format returns 400
- [ ] Non-existent user returns empty data array
- [ ] Pagination works correctly
- [ ] Only returns events for specified user

### GET /stats
- [ ] Valid request returns 200 with stats
- [ ] Invalid date format returns 400
- [ ] from parameter filters start date
- [ ] to parameter filters end date
- [ ] type parameter filters by event type
- [ ] Non-existent event type returns 400
- [ ] top_pages returns JSON object with counts
- [ ] unique_users returns count of distinct users

### General
- [ ] All errors return JSON with "error" key
- [ ] All timestamps are ISO 8601 format
- [ ] All UUIDs are valid v4 format
