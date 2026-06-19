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

type ProjectStore interface {
	ListProjects(context.Context) ([]database.Project, error)
	CreateProject(context.Context, database.CreateProjectParams) (database.Project, error)
}

type AuthStore interface {
	Login(context.Context, database.LoginParams, time.Duration) (database.Session, error)
	Authenticate(context.Context, string) (database.AuthUser, error)
	AuthenticateProjectAPIKey(context.Context, string) (database.ProjectAPIKey, error)
}

type UserStore interface {
	ListUsers(context.Context) ([]database.User, error)
	CreateUser(context.Context, database.CreateUserParams) (database.User, error)
	ListProjectMembers(context.Context, string) ([]database.ProjectMember, error)
	AddMembership(context.Context, string, database.AddMembershipParams) (database.ProjectMember, error)
}

type ProjectDataStore interface {
	ListProjectTables(context.Context, string) ([]database.ProjectTable, error)
	CreateProjectTable(context.Context, string, database.CreateProjectTableParams) (database.ProjectTable, error)
	GetProjectTableAccess(context.Context, string, string) (database.ProjectTableAccess, error)
	ListProjectRecords(context.Context, string, string) ([]database.ProjectRecord, error)
	CreateProjectRecord(context.Context, string, string, string, database.CreateProjectRecordParams) (database.ProjectRecord, error)
	UpdateProjectRecord(context.Context, string, string, string, database.UpdateProjectRecordParams) (database.ProjectRecord, error)
	DeleteProjectRecord(context.Context, string, string, string) error
}

type ProjectAPIKeyStore interface {
	ListProjectAPIKeys(context.Context, string) ([]database.ProjectAPIKey, error)
	CreateProjectAPIKey(context.Context, string, string, database.CreateProjectAPIKeyParams) (database.CreatedProjectAPIKey, error)
}

type StorageStore interface {
	ListFiles(context.Context, string) ([]database.File, error)
	CreateFileIntent(context.Context, string, string, string, string, database.CreateFileIntentParams) (database.FileIntent, error)
	GetFile(context.Context, string, string) (database.File, error)
}

type AuditStore interface {
	CreateAuditEvent(context.Context, database.CreateAuditEventParams) error
	ListAuditEvents(context.Context, string) ([]database.AuditEvent, error)
}

type ObjectSigner interface {
	PresignUpload(context.Context, database.File) (database.StorageOperation, error)
	PresignDownload(context.Context, database.File) (database.StorageOperation, error)
}

type Server struct {
	cfg           config.Config
	logger        *slog.Logger
	mux           *http.ServeMux
	healthChecker HealthChecker
	projectStore  ProjectStore
	authStore     AuthStore
	userStore     UserStore
	dataStore     ProjectDataStore
	apiKeyStore   ProjectAPIKeyStore
	storageStore  StorageStore
	auditStore    AuditStore
	objectSigner  ObjectSigner
}

func NewServer(cfg config.Config, logger *slog.Logger, store interface {
	HealthChecker
	ProjectStore
	AuthStore
	UserStore
	ProjectDataStore
	ProjectAPIKeyStore
	StorageStore
	AuditStore
}, objectSigners ...ObjectSigner) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	var healthChecker HealthChecker
	var projectStore ProjectStore
	var authStore AuthStore
	var userStore UserStore
	var dataStore ProjectDataStore
	var apiKeyStore ProjectAPIKeyStore
	var storageStore StorageStore
	var auditStore AuditStore
	if store != nil {
		healthChecker = store
		projectStore = store
		authStore = store
		userStore = store
		dataStore = store
		apiKeyStore = store
		storageStore = store
		auditStore = store
	}

	server := &Server{
		cfg:           cfg,
		logger:        logger,
		mux:           http.NewServeMux(),
		healthChecker: healthChecker,
		projectStore:  projectStore,
		authStore:     authStore,
		userStore:     userStore,
		dataStore:     dataStore,
		apiKeyStore:   apiKeyStore,
		storageStore:  storageStore,
		auditStore:    auditStore,
	}
	if len(objectSigners) > 0 {
		server.objectSigner = objectSigners[0]
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
	s.mux.HandleFunc("GET /v1/users", s.listUsers)
	s.mux.HandleFunc("POST /v1/users", s.createUser)
	s.mux.HandleFunc("GET /v1/projects", s.listProjects)
	s.mux.HandleFunc("POST /v1/projects", s.createProject)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/members", s.listProjectMembers)
	s.mux.HandleFunc("POST /v1/projects/{projectID}/members", s.addProjectMember)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/api-keys", s.listProjectAPIKeys)
	s.mux.HandleFunc("POST /v1/projects/{projectID}/api-keys", s.createProjectAPIKey)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/tables", s.listProjectTables)
	s.mux.HandleFunc("POST /v1/projects/{projectID}/tables", s.createProjectTable)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/tables/{tableName}/records", s.listProjectRecords)
	s.mux.HandleFunc("POST /v1/projects/{projectID}/tables/{tableName}/records", s.createProjectRecord)
	s.mux.HandleFunc("PATCH /v1/projects/{projectID}/tables/{tableName}/records/{recordID}", s.updateProjectRecord)
	s.mux.HandleFunc("DELETE /v1/projects/{projectID}/tables/{tableName}/records/{recordID}", s.deleteProjectRecord)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/files", s.listFiles)
	s.mux.HandleFunc("POST /v1/projects/{projectID}/files", s.createFileIntent)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/files/{fileID}", s.getFile)
	s.mux.HandleFunc("GET /v1/projects/{projectID}/audit-events", s.listAuditEvents)
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
			"projects",
			"data",
			"storage",
			"audit",
		},
	})
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	if s.projectStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	projects, err := s.projectStore.ListProjects(r.Context())
	if err != nil {
		s.logger.Error("list projects", "error", err)
		writeError(w, http.StatusInternalServerError, "list projects")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
	})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !user.HasRole("owner") {
		writeError(w, http.StatusForbidden, "owner role required")
		return
	}

	if s.projectStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateProjectParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Template = strings.TrimSpace(input.Template)

	project, err := s.projectStore.CreateProject(r.Context(), input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create project", "error", err)
		writeError(w, http.StatusInternalServerError, "create project")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   project.ID,
		ActorUserID: user.User.ID,
		Action:      "project.create",
		TargetType:  "project",
		TargetID:    project.ID,
		Metadata: map[string]any{
			"name":     project.Name,
			"slug":     project.Slug,
			"template": project.Template,
		},
	})

	writeJSON(w, http.StatusCreated, project)
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

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !user.HasRole("owner") {
		writeError(w, http.StatusForbidden, "owner role required")
		return
	}
	if s.userStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	users, err := s.userStore.ListUsers(r.Context())
	if err != nil {
		s.logger.Error("list users", "error", err)
		writeError(w, http.StatusInternalServerError, "list users")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	if !user.HasRole("owner") {
		writeError(w, http.StatusForbidden, "owner role required")
		return
	}
	if s.userStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateUserParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	created, err := s.userStore.CreateUser(r.Context(), input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create user", "error", err)
		writeError(w, http.StatusInternalServerError, "create user")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	projectID := r.PathValue("projectID")
	if !user.IsProjectMember(projectID) {
		writeError(w, http.StatusForbidden, "project membership required")
		return
	}
	if s.userStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	members, err := s.userStore.ListProjectMembers(r.Context(), projectID)
	if err != nil {
		s.logger.Error("list project members", "error", err)
		writeError(w, http.StatusInternalServerError, "list project members")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (s *Server) addProjectMember(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return
	}
	projectID := r.PathValue("projectID")
	if !user.HasProjectRole(projectID, "owner", "admin") {
		writeError(w, http.StatusForbidden, "project owner or admin role required")
		return
	}
	if s.userStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.AddMembershipParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	member, err := s.userStore.AddMembership(r.Context(), projectID, input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("add project member", "error", err)
		writeError(w, http.StatusInternalServerError, "add project member")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   projectID,
		ActorUserID: user.User.ID,
		Action:      "membership.add",
		TargetType:  "membership",
		TargetID:    member.UserID,
		Metadata: map[string]any{
			"user_id": member.UserID,
			"email":   member.Email,
			"role":    member.Role,
		},
	})

	writeJSON(w, http.StatusCreated, member)
}

func (s *Server) listProjectAPIKeys(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireProjectRole(w, r, "owner", "admin", "developer")
	if !ok {
		return
	}
	if s.apiKeyStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	keys, err := s.apiKeyStore.ListProjectAPIKeys(r.Context(), r.PathValue("projectID"))
	if err != nil {
		s.logger.Error("list project api keys", "error", err)
		writeError(w, http.StatusInternalServerError, "list project api keys")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"api_keys": keys})
}

func (s *Server) createProjectAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireProjectRole(w, r, "owner", "admin", "developer")
	if !ok {
		return
	}
	if s.apiKeyStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateProjectAPIKeyParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	key, err := s.apiKeyStore.CreateProjectAPIKey(r.Context(), r.PathValue("projectID"), user.User.ID, input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create project api key", "error", err)
		writeError(w, http.StatusInternalServerError, "create project api key")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   r.PathValue("projectID"),
		ActorUserID: user.User.ID,
		Action:      "api_key.create",
		TargetType:  "api_key",
		TargetID:    key.ID,
		Metadata: map[string]any{
			"name":         key.Name,
			"token_prefix": key.TokenPrefix,
		},
	})

	writeJSON(w, http.StatusCreated, key)
}

func (s *Server) listProjectTables(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireProjectMember(w, r)
	if !ok {
		return
	}
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	tables, err := s.dataStore.ListProjectTables(r.Context(), r.PathValue("projectID"))
	if err != nil {
		s.logger.Error("list project tables", "error", err)
		writeError(w, http.StatusInternalServerError, "list project tables")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

func (s *Server) createProjectTable(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireProjectRole(w, r, "owner", "admin", "developer")
	if !ok {
		return
	}
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateProjectTableParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	table, err := s.dataStore.CreateProjectTable(r.Context(), r.PathValue("projectID"), input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create project table", "error", err)
		writeError(w, http.StatusInternalServerError, "create project table")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   r.PathValue("projectID"),
		ActorUserID: user.User.ID,
		Action:      "table.create",
		TargetType:  "project_table",
		TargetID:    table.ID,
		Metadata: map[string]any{
			"name":         table.Name,
			"read_access":  table.ReadAccess,
			"write_access": table.WriteAccess,
		},
	})

	writeJSON(w, http.StatusCreated, table)
}

func (s *Server) listProjectRecords(w http.ResponseWriter, r *http.Request) {
	ok := s.requireTableAccess(w, r, "read")
	if !ok {
		return
	}
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	records, err := s.dataStore.ListProjectRecords(r.Context(), r.PathValue("projectID"), r.PathValue("tableName"))
	if err != nil {
		s.logger.Error("list project records", "error", err)
		writeError(w, http.StatusInternalServerError, "list project records")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"records": records})
}

func (s *Server) createProjectRecord(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireTableActor(w, r, "write")
	if !ok {
		return
	}
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateProjectRecordParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	record, err := s.dataStore.CreateProjectRecord(r.Context(), r.PathValue("projectID"), r.PathValue("tableName"), actor.userID, input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create project record", "error", err)
		writeError(w, http.StatusInternalServerError, "create project record")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   r.PathValue("projectID"),
		ActorUserID: actor.userID,
		Action:      "record.create",
		TargetType:  "project_record",
		TargetID:    record.ID,
		Metadata: map[string]any{
			"table": r.PathValue("tableName"),
		},
	})

	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) updateProjectRecord(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireTableActor(w, r, "write")
	if !ok {
		return
	}
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.UpdateProjectRecordParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	record, err := s.dataStore.UpdateProjectRecord(r.Context(), r.PathValue("projectID"), r.PathValue("tableName"), r.PathValue("recordID"), input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("update project record", "error", err)
		writeError(w, http.StatusInternalServerError, "update project record")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   r.PathValue("projectID"),
		ActorUserID: actor.userID,
		Action:      "record.update",
		TargetType:  "project_record",
		TargetID:    record.ID,
		Metadata: map[string]any{
			"table": r.PathValue("tableName"),
		},
	})

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) deleteProjectRecord(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireTableActor(w, r, "write")
	if !ok {
		return
	}
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	if err := s.dataStore.DeleteProjectRecord(r.Context(), r.PathValue("projectID"), r.PathValue("tableName"), r.PathValue("recordID")); err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("delete project record", "error", err)
		writeError(w, http.StatusInternalServerError, "delete project record")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   r.PathValue("projectID"),
		ActorUserID: actor.userID,
		Action:      "record.delete",
		TargetType:  "project_record",
		TargetID:    r.PathValue("recordID"),
		Metadata: map[string]any{
			"table": r.PathValue("tableName"),
		},
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireProjectMember(w, r)
	if !ok {
		return
	}
	if s.storageStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	files, err := s.storageStore.ListFiles(r.Context(), r.PathValue("projectID"))
	if err != nil {
		s.logger.Error("list files", "error", err)
		writeError(w, http.StatusInternalServerError, "list files")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (s *Server) createFileIntent(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireProjectActor(w, r)
	if !ok {
		return
	}
	if s.storageStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	var input database.CreateFileIntentParams
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	intent, err := s.storageStore.CreateFileIntent(r.Context(), r.PathValue("projectID"), actor.userID, s.cfg.StorageBucket, s.cfg.PublicURL, input)
	if err != nil {
		if errors.Is(err, database.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Error("create file intent", "error", err)
		writeError(w, http.StatusInternalServerError, "create file intent")
		return
	}
	if err := s.signFileIntent(r.Context(), &intent); err != nil {
		s.logger.Error("sign file intent", "error", err)
		writeError(w, http.StatusInternalServerError, "sign file intent")
		return
	}
	s.recordAudit(r.Context(), database.CreateAuditEventParams{
		ProjectID:   r.PathValue("projectID"),
		ActorUserID: actor.userID,
		Action:      "file.intent.create",
		TargetType:  "file",
		TargetID:    intent.File.ID,
		Metadata: map[string]any{
			"file_name":  intent.File.FileName,
			"mime_type":  intent.File.MimeType,
			"byte_size":  intent.File.ByteSize,
			"access":     intent.File.Access,
			"object_key": intent.File.ObjectKey,
		},
	})

	writeJSON(w, http.StatusCreated, intent)
}

func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireProjectMember(w, r)
	if !ok {
		return
	}
	if s.storageStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	file, err := s.storageStore.GetFile(r.Context(), r.PathValue("projectID"), r.PathValue("fileID"))
	if err != nil {
		s.logger.Error("get file", "error", err)
		writeError(w, http.StatusInternalServerError, "get file")
		return
	}

	writeJSON(w, http.StatusOK, file)
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireProjectRole(w, r, "owner", "admin", "developer")
	if !ok {
		return
	}
	if s.auditStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return
	}

	events, err := s.auditStore.ListAuditEvents(r.Context(), r.PathValue("projectID"))
	if err != nil {
		s.logger.Error("list audit events", "error", err)
		writeError(w, http.StatusInternalServerError, "list audit events")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"audit_events": events})
}

func (s *Server) signFileIntent(ctx context.Context, intent *database.FileIntent) error {
	if s.objectSigner == nil {
		return nil
	}

	upload, err := s.objectSigner.PresignUpload(ctx, intent.File)
	if err != nil {
		return err
	}
	download, err := s.objectSigner.PresignDownload(ctx, intent.File)
	if err != nil {
		return err
	}
	intent.Upload = upload
	intent.Download = download
	return nil
}

func (s *Server) recordAudit(ctx context.Context, input database.CreateAuditEventParams) {
	if s.auditStore == nil {
		return
	}
	if err := s.auditStore.CreateAuditEvent(ctx, input); err != nil {
		s.logger.Error("record audit event", "error", err, "action", input.Action)
	}
}

type tableActor struct {
	userID string
}

func (s *Server) requireTableAccess(w http.ResponseWriter, r *http.Request, operation string) bool {
	_, ok := s.requireTableActor(w, r, operation)
	return ok
}

func (s *Server) requireTableActor(w http.ResponseWriter, r *http.Request, operation string) (tableActor, bool) {
	if s.dataStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return tableActor{}, false
	}

	projectID := r.PathValue("projectID")
	access, err := s.dataStore.GetProjectTableAccess(r.Context(), projectID, r.PathValue("tableName"))
	if err != nil {
		s.logger.Error("get project table access", "error", err)
		writeError(w, http.StatusInternalServerError, "get project table access")
		return tableActor{}, false
	}

	policy := access.ReadAccess
	if operation == "write" {
		policy = access.WriteAccess
	}
	if policy == "public" {
		return tableActor{}, true
	}

	if policy == "api_key" && apiKeyToken(r) != "" {
		token := apiKeyToken(r)
		key, ok := s.authenticateProjectAPIKey(w, r, token)
		if !ok {
			return tableActor{}, false
		}
		if key.ProjectID != projectID {
			writeError(w, http.StatusForbidden, "api key is not scoped to this project")
			return tableActor{}, false
		}
		return tableActor{}, true
	}

	user, ok := s.requireAuth(w, r)
	if !ok {
		return tableActor{}, false
	}
	if !user.IsProjectMember(projectID) {
		writeError(w, http.StatusForbidden, "project membership required")
		return tableActor{}, false
	}

	return tableActor{userID: user.User.ID}, true
}

func (s *Server) requireProjectActor(w http.ResponseWriter, r *http.Request) (tableActor, bool) {
	projectID := r.PathValue("projectID")
	if token := apiKeyToken(r); token != "" {
		key, ok := s.authenticateProjectAPIKey(w, r, token)
		if !ok {
			return tableActor{}, false
		}
		if key.ProjectID != projectID {
			writeError(w, http.StatusForbidden, "api key is not scoped to this project")
			return tableActor{}, false
		}
		return tableActor{}, true
	}

	user, ok := s.requireAuth(w, r)
	if !ok {
		return tableActor{}, false
	}
	if !user.IsProjectMember(projectID) {
		writeError(w, http.StatusForbidden, "project membership required")
		return tableActor{}, false
	}
	return tableActor{userID: user.User.ID}, true
}

func (s *Server) requireProjectMember(w http.ResponseWriter, r *http.Request) (database.AuthUser, bool) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return database.AuthUser{}, false
	}
	if !user.IsProjectMember(r.PathValue("projectID")) {
		writeError(w, http.StatusForbidden, "project membership required")
		return database.AuthUser{}, false
	}
	return user, true
}

func (s *Server) requireProjectRole(w http.ResponseWriter, r *http.Request, roles ...string) (database.AuthUser, bool) {
	user, ok := s.requireAuth(w, r)
	if !ok {
		return database.AuthUser{}, false
	}
	if !user.HasProjectRole(r.PathValue("projectID"), roles...) {
		writeError(w, http.StatusForbidden, "project role required")
		return database.AuthUser{}, false
	}
	return user, true
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

func (s *Server) authenticateProjectAPIKey(w http.ResponseWriter, r *http.Request, token string) (database.ProjectAPIKey, bool) {
	if s.authStore == nil {
		writeError(w, http.StatusServiceUnavailable, "database not configured")
		return database.ProjectAPIKey{}, false
	}

	key, err := s.authStore.AuthenticateProjectAPIKey(r.Context(), token)
	if err != nil {
		if errors.Is(err, database.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return database.ProjectAPIKey{}, false
		}
		s.logger.Error("authenticate project api key", "error", err)
		writeError(w, http.StatusInternalServerError, "authenticate project api key")
		return database.ProjectAPIKey{}, false
	}

	return key, true
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func apiKeyToken(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Kuza-API-Key"))
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
