package main

import (
	"collider/internal/cache"
	"collider/internal/stores"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	start := time.Now()

	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := stores.Connect(databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Truncating tables...")

	query := `truncate table users, event_types, events restart identity cascade`
	_, err = db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}

	// Invalidate caches
	invalidateCaches()

	fmt.Printf("Truncated in %.2f seconds\n", time.Since(start).Seconds())
}

func invalidateCaches() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "dragonfly:6379"
	}

	c := cache.NewFromURL(redisURL, 10)
	if c == nil {
		return
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Invalidate all caches
	c.InvalidateEvents(ctx)
	c.InvalidateStats(ctx)

	log.Println("Cache invalidated")
}
