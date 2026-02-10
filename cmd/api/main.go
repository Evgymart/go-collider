package main

import (
	"collider/database"
	"collider/handlers"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

func main() {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		log.Fatal("SERVER_PORT environment variable is required")
	}

	log.Printf("Starting server on port %s", serverPort)

	db, err := database.Connect(databaseUrl)
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
	eventStore := database.NewEventStore(db)
	statsStore := database.NewStatsStore(db)

	handlers := handlers.NewHandlers(eventStore, statsStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetEventsPaginated(w, r)
		case http.MethodPost:
			handlers.CreateEvent(w, r)
		default:
			respondWithError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		}
	})

	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/users/") || !strings.HasSuffix(path, "/events") {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handlers.GetUserEventsPaginated(w, r)
		default:
			respondWithError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
		}
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondWithError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}

		handlers.GetStats(w, r)
	})

	loggedMux := loggingMiddleware(mux)
	serverAddr := ":" + serverPort

	err = http.ListenAndServe(serverAddr, loggedMux)
	if err != nil {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func respondWithError(w http.ResponseWriter, statusCode int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
