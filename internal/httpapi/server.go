package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"kuza-core/internal/config"
)

type Server struct {
	cfg    config.Config
	logger *slog.Logger
	mux    *http.ServeMux
}

func NewServer(cfg config.Config, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	server := &Server{
		cfg:    cfg,
		logger: logger,
		mux:    http.NewServeMux(),
	}
	server.routes()

	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.HandleFunc("GET /v1", s.index)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "kuza-core-api",
	})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	status := "degraded"
	if s.cfg.DatabaseURL != "" {
		status = "ready"
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": status,
	})
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "Kuza Core",
		"version": "v1",
		"modules": []string{
			"identity",
			"schools",
			"content",
			"storage",
			"audit",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
