package server

import (
	"encoding/json"
	"net/http"
	"time"
)

const serviceName = "stellatlas-service"

type healthResponse struct {
	Service string `json:"service"`
	Status  string `json:"status"`
}

type statusResponse struct {
	Service     string `json:"service"`
	Product     string `json:"product"`
	Role        string `json:"role"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
}

// NewHandler builds the HTTP handler for StellAtlas service.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", onlyMethod(http.MethodGet, handleHealth))
	mux.HandleFunc("/api/stellatlas/v1/status", onlyMethod(http.MethodGet, handleStatus))
	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Service: serviceName,
		Status:  "ok",
	})
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, statusResponse{
		Service:     serviceName,
		Product:     "StellAtlas",
		Role:        "Configuration Management Database service",
		Description: "Manages configuration items, asset inventory, topology relationships, and lifecycle metadata for the Stell platform.",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

func onlyMethod(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
