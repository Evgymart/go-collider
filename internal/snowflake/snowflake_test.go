package snowflake

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		nodeID  int64
		wantErr bool
	}{
		{"valid node ID", 0, false},
		{"valid node ID max", 1023, false},
		{"invalid node ID negative", -1, true},
		{"invalid node ID too large", 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerate(t *testing.T) {
	sf, err := New(1)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	id, err := sf.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("Generate() returned non-positive ID: %d", id)
	}
}

func TestGenerateUniqueness(t *testing.T) {
	sf, err := New(1)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ids := make(map[int64]bool)
	for i := 0; i < 10000; i++ {
		id, err := sf.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}
		if ids[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		ids[id] = true
	}
}

func TestGenerateConcurrent(t *testing.T) {
	sf, err := New(1)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	var wg sync.WaitGroup
	ids := make(chan int64, 1000)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				id, err := sf.Generate()
				if err != nil {
					t.Errorf("Generate() failed: %v", err)
					return
				}
				ids <- id
			}
		}()
	}

	wg.Wait()
	close(ids)

	uniqueIDs := make(map[int64]bool)
	for id := range ids {
		if uniqueIDs[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		uniqueIDs[id] = true
	}
}

func TestParse(t *testing.T) {
	sf, err := New(42)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	id, err := sf.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	timestamp, nodeID, sequence := Parse(id)

	if nodeID != 42 {
		t.Errorf("Parse() nodeID = %d, want 42", nodeID)
	}

	if sequence < 0 || sequence > MaxSequence {
		t.Errorf("Parse() sequence = %d, want between 0 and %d", sequence, MaxSequence)
	}

	// Timestamp should be recent (within last second)
	parsedTime := time.UnixMilli(timestamp)
	if time.Since(parsedTime) > time.Second {
		t.Errorf("Parse() timestamp = %v, want recent time", parsedTime)
	}
}

func TestParseTime(t *testing.T) {
	sf, err := New(1)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	id, err := sf.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	parsedTime := ParseTime(id)

	// Parsed time should be within last second
	if time.Since(parsedTime) > time.Second {
		t.Errorf("ParseTime() = %v, want recent time", parsedTime)
	}
}

func TestMonotonic(t *testing.T) {
	sf, err := New(1)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	var lastID int64
	for i := 0; i < 1000; i++ {
		id, err := sf.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}
		if id <= lastID {
			t.Errorf("Generate() ID not monotonic: %d <= %d", id, lastID)
		}
		lastID = id
	}
}

func BenchmarkGenerate(b *testing.B) {
	sf, err := New(1)
	if err != nil {
		b.Fatalf("New() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sf.Generate()
	}
}

func BenchmarkGenerateParallel(b *testing.B) {
	sf, err := New(1)
	if err != nil {
		b.Fatalf("New() failed: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sf.Generate()
		}
	})
}
