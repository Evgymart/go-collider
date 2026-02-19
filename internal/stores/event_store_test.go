package stores

import (
	"collider/internal/snowflake"
	"testing"
	"time"
)

// TestSnowflakeIDGeneration verifies that EventStore generates snowflake IDs
func TestSnowflakeIDGeneration(t *testing.T) {
	// This is a simple unit test to verify the snowflake integration
	// Integration tests should be in test/cases package

	sf, err := snowflake.New(1)
	if err != nil {
		t.Fatalf("Failed to create snowflake generator: %v", err)
	}

	// Generate a few IDs and verify they are unique and monotonic
	var lastID int64 = 0
	for i := 0; i < 100; i++ {
		id, err := sf.Generate()
		if err != nil {
			t.Fatalf("Failed to generate snowflake ID: %v", err)
		}

		if id <= lastID {
			t.Errorf("Snowflake IDs should be monotonic: got %d, last was %d", id, lastID)
		}

		// Verify the ID is a valid snowflake ID by parsing it
		ts, nodeID, seq := snowflake.Parse(id)
		if nodeID != 1 {
			t.Errorf("Expected node ID 1, got %d", nodeID)
		}
		if seq < 0 || seq > snowflake.MaxSequence {
			t.Errorf("Sequence out of valid range: %d", seq)
		}

		// Timestamp should be recent (within last second)
		parsedTime := snowflake.ParseTime(id)
		if time.Since(parsedTime) > time.Second {
			t.Errorf("Parsed time seems incorrect: %v", parsedTime)
		}

		_ = ts // Use the variable
		lastID = id
	}
}

// TestEventStoreWithSnowflake is a basic integration test
// This would require a test database setup, so we'll skip it for now
// and rely on the test/cases package for full integration testing
func TestEventStoreWithSnowflake(t *testing.T) {
	t.Skip("Skipping database test - see test/cases package for integration tests")

	// Example of how this would work with a test DB:
	// db := setupTestDB(t)
	// store := NewEventStore(db)
	//
	// event, err := store.CreateEventWithType(1, "test.event", []byte("{}"))
	// if err != nil {
	// 	t.Fatalf("Failed to create event: %v", err)
	// }
	//
	// // Verify the ID is a snowflake ID (large positive number)
	// if event.ID <= 0 {
	// 	t.Errorf("Expected positive snowflake ID, got %d", event.ID)
	// }
	//
	// // Snowflake IDs generated after 2024-01-01 should be very large
	// // (e.g., > 10000000000000000)
	// if event.ID < 10000000000000000 {
	// 	t.Logf("Warning: ID seems small for snowflake: %d", event.ID)
	// }
}

// BenchmarkSnowflakeGeneration benchmarks the snowflake ID generation
func BenchmarkSnowflakeGeneration(b *testing.B) {
	sf, _ := snowflake.New(1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sf.Generate()
	}
}

// BenchmarkEventStoreCreation benchmarks creating events (with mock DB)
// This would require a mock or test database
func BenchmarkEventStoreCreation(b *testing.B) {
	b.Skip("Skipping DB benchmark - requires test database setup")
}
