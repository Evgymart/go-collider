package main

import (
	"collider/internal/config"
	"collider/internal/handlers"
	"collider/internal/middleware"
	"collider/internal/router"
	"collider/internal/stores"

	"log"
	"net/http"

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
			log.Fatal(err)
		}
	}(db)

	log.Println("Successful db connection!")
	eventStore := stores.NewEventStore(db)
	statsStore := stores.NewStatsStore(db)

	h := handlers.NewHandlers(eventStore, statsStore)

	mux := router.New(h)
	loggedMux := middleware.Logging(mux)
	serverAddr := ":" + cfg.ServerPort

	err = http.ListenAndServe(serverAddr, loggedMux)
	if err != nil {
		log.Fatal(err)
	}
}
