package main

import (
	"collider/internal/stores"
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

	fmt.Printf("Truncated in %.2f seconds\n", time.Since(start).Seconds())
}
