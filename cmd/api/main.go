package main

import (
	"collider/internal/cache"
	"collider/internal/config"
	"collider/internal/handlers"
	"collider/internal/middleware"
	"collider/internal/router"
	"collider/internal/stores"

	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/jmoiron/sqlx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting server on port %s", cfg.ServerPort)

	db, err := stores.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	defer func(db *sqlx.DB) {
		err := db.Close()
		if err != nil {
			log.Printf("error: failed to close database: %v", err)
		}
	}(db)

	log.Println("Successful db connection!")

	numCPU := runtime.NumCPU()
	maxOpenConns := numCPU * 4
	maxIdleConns := numCPU * 2
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(1 * time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)
	log.Printf("Connection pool configured: maxOpen=%d, maxIdle=%d", maxOpenConns, maxIdleConns)
	log.Printf("Snowflake node ID: %d", cfg.SnowflakeNodeID)

	eventStore := stores.NewEventStore(db, cfg.SnowflakeNodeID)
	statsStore := stores.NewStatsStore(db)

	var c *cache.Cache
	if cfg.CacheURL != "" {
		c = cache.NewFromURL(cfg.CacheURL, cfg.CachePoolSize)
		defer c.Close()
	} else {
		log.Println("cache: disabled (no CACHE_URL set)")
	}

	h := handlers.NewHandlers(eventStore, statsStore, c)

	mux := router.New(h)
	loggedMux := middleware.Logging(mux)
	serverAddr := ":" + cfg.ServerPort

	err = http.ListenAndServe(serverAddr, loggedMux)
	if err != nil {
		log.Fatal(err)
	}
}
