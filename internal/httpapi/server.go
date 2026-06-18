package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"kuza-core/internal/config"
	"kuza-core/internal/database"
)

type HealthChecker interface {
	Ping(context.Context) error
}

type OrganizationStore interface {
	ListOrganizations(context.Context) ([]database.Organization, error)
	CreateOrganization(context.Context, database.CreateOrganizationParams) (database.Organization, error)
}

type Server struct {
	cfg           config.Config
	logger        *slog.Logger
	mux           *http.ServeMux
	healthChecker HealthChecker
	orgStore      OrganizationStore
}

func NewServer(cfg config.Config, logger *slog.Logger, store interface {
	HealthChecker
	OrganizationStore
}) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	var healthChecker HealthChecker
	var orgStore OrganizationStore
	if store != nil {
		healthChecker = store
		orgStore = store
	}

	server := &Server{
		cfg:           cfg,
		logger:        logger,
		mux:           http.NewServeMux(),
		healthChecker: healthChecker,
		orgStore:      orgStore,
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
	s.mux.HandleFunc("GET /v1/organizations", s.listOrganizations)
	s.mux.HandleFunc("POST /v1/organizations", s.createOrganization)
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

func (s *Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	if s.orgStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	organizations, err := s.orgStore.ListOrganizations(r.Context())
	if err != nil {
		s.logger.Error("list organizations", "error", err)
		writeError(w, http.StatusInternalServerError, "list organizations")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"organizations": organizations,
	})
}

func (s *Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	if s.orgStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateOrganizationParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Kind = strings.TrimSpace(input.Kind)

	organization, err := s.orgStore.CreateOrganization(r.Context(), input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create organization", "error", err)
		writeError(w, http.StatusInternalServerError, "create organization")
		return
	}

	writeJSON(w, http.StatusCreated, organization)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
