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

type AuthStore interface {
	Login(context.Context, database.LoginParams, time.Duration) (database.Session, error)
	Authenticate(context.Context, string) (database.AuthUser, error)
}

type Server struct {
	cfg           config.Config
	logger        *slog.Logger
	mux           *http.ServeMux
	healthChecker HealthChecker
	orgStore      OrganizationStore
	authStore     AuthStore
}

func NewServer(cfg config.Config, logger *slog.Logger, store interface {
	HealthChecker
	OrganizationStore
	AuthStore
}) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	var healthChecker HealthChecker
	var orgStore OrganizationStore
	var authStore AuthStore
	if store != nil {
		healthChecker = store
		orgStore = store
		authStore = store
	}

	server := &Server{
		cfg:           cfg,
		logger:        logger,
		mux:           http.NewServeMux(),
		healthChecker: healthChecker,
		orgStore:      orgStore,
		authStore:     authStore,
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
	s.mux.HandleFunc("POST /v1/auth/login", s.login)
	s.mux.HandleFunc("GET /v1/auth/me", s.me)
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
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !user.HasRole("owner") {
		writeError(w, http.StatusForbidden, "owner role required")
		return
	}

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

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.authStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.LoginParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	session, err := s.authStore.Login(r.Context(), input, time.Duration(s.cfg.SessionTTLHours)*time.Hour)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, database.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		s.logger.Error("login", "error", err)
		writeError(w, http.StatusInternalServerError, "login")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (s *Server) requireAuth(w http.ResponseWriter, r *http.Request) (database.AuthUser, bool) {
	if s.authStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return database.AuthUser{}, false
	}

	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "bearer token required")
		return database.AuthUser{}, false
	}

	user, err := s.authStore.Authenticate(r.Context(), token)
	if err != nil {
		if errors.Is(err, database.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "invalid bearer token")
			return database.AuthUser{}, false
		}
		s.logger.Error("authenticate", "error", err)
		writeError(w, http.StatusInternalServerError, "authenticate")
		return database.AuthUser{}, false
	}

	return user, true
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
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
