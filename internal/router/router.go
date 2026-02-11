package router

import (
	"collider/internal/handlers"

	"net/http"
	"strings"
)

func New(h *handlers.Handlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/events", methodHandler(map[string]http.HandlerFunc{
		http.MethodGet:  h.GetEventsPaginated,
		http.MethodPost: h.CreateEvent,
	}))

	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/users/") || !strings.HasSuffix(path, "/events") {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.GetUserEventsPaginated(w, r)
		default:
			methodNotAllowed(w)
		}
	})

	mux.HandleFunc("/stats", methodHandler(map[string]http.HandlerFunc{
		http.MethodGet: h.GetStats,
	}))

	return mux
}

func methodHandler(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h, ok := handlers[r.Method]; ok {
			h(w, r)
			return
		}
		methodNotAllowed(w)
	}
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte(`{"error":"method not allowed"}`))
}
