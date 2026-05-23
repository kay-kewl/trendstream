package api

import (
	"encoding/json"
	"net/http"
	"time"
)

type HealthHandler struct {
	ServiceName string
	StartedAt   time.Time
}

func NewHealthHandler(serviceName string, startedAt time.Time) *HealthHandler {
	return &HealthHandler{
		ServiceName: serviceName,
		StartedAt:   startedAt,
	}
}

func (h *HealthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /readyz", h.Readyz)
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":       "ok",
		"service":      h.ServiceName,
		"started_at":   h.StartedAt.UTC().Format(time.RFC3339),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":       "ready",
		"service":      h.ServiceName,
		"started_at":   h.StartedAt.UTC().Format(time.RFC3339),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(payload)
}
