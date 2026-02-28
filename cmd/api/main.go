package main

import (
	"collider/internal/cache"
	"collider/internal/config"
	"collider/internal/handlers"
	"collider/internal/middleware"
	"collider/internal/queue"
	"collider/internal/repositories"
	"collider/internal/router"
	"collider/internal/stores"

	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
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

	// Create repositories
	eventRepository := repositories.NewEventRepository(eventStore, c)
	statsRepository := repositories.NewStatsRepository(statsStore, eventStore, c)

	queueConfig := queue.DefaultConfig()
	eventQueue, err := queue.NewEventQueue(eventStore, queueConfig)
	if err != nil {
		log.Fatalf("Failed to create event queue: %v", err)
	}

	if err := eventQueue.Start(); err != nil {
		log.Fatalf("Failed to start event queue: %v", err)
	}
	log.Println("event queue: started")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			stats := eventQueue.GetStats()
			log.Printf("queue stats: depth=%d/%d, enqueued=%d, flushed=%d, failed=%d, dlq=%d",
				stats.CurrentDepth, stats.MaxDepth, stats.TotalEnqueued,
				stats.TotalFlushed, stats.TotalFailed, stats.DeadLetterCount)
		}
	}()

	h := handlers.NewHandlers(eventRepository, statsRepository)
	h.SetEventQueue(eventQueue)

	mux := router.New(h)
	loggedMux := middleware.Logging(mux)
	serverAddr := ":" + cfg.ServerPort

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      loggedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("server: listening on %s", serverAddr)
		serverErr <- server.ListenAndServe()
	}()

	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("server error: %v", err)
	case sig := <-sigChan:
		log.Printf("server: received signal %v, initiating graceful shutdown", sig)
	}

	shutdownTimeout := 30 * time.Second
	log.Println("server: shutting down HTTP server...")
	ctx, cancel := context.WithTimeout(shutdownCtx, shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("server: HTTP shutdown error: %v", err)
	}

	log.Println("server: shutting down event queue...")
	queueShutdownTimeout := 30 * time.Second
	if err := eventQueue.Shutdown(queueShutdownTimeout); err != nil {
		log.Printf("server: event queue shutdown error: %v", err)
	}

	stats := eventQueue.GetStats()
	log.Printf("server: final queue stats: enqueued=%d, flushed=%d, failed=%d",
		stats.TotalEnqueued, stats.TotalFlushed, stats.TotalFailed)

	if c != nil {
		c.Close()
	}

	log.Println("server: shutdown complete")
}
