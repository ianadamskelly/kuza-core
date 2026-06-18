package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"kuza-core/internal/config"
)

type HealthChecker interface {
	Ping(context.Context) error
}

type Server struct {
	cfg           config.Config
	logger        *slog.Logger
	mux           *http.ServeMux
	healthChecker HealthChecker
}

func NewServer(cfg config.Config, logger *slog.Logger, healthChecker HealthChecker) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	server := &Server{
		cfg:           cfg,
		logger:        logger,
		mux:           http.NewServeMux(),
		healthChecker: healthChecker,
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
	if s.healthChecker == nil {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "degraded",
			"reason": "database not configured",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.healthChecker.Ping(ctx); err != nil {
		s.logger.Warn("readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"reason": "database unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
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
