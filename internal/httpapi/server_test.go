package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kuza-core/internal/config"
	"kuza-core/internal/database"
)

type fakeStore struct {
	pingErr   error
	projects  []database.Project
	createErr error
	loginErr  error
	authErr   error
	authUser  database.AuthUser
	users     []database.User
	members   []database.ProjectMember
	tables    []database.ProjectTable
	records   []database.ProjectRecord
	apiKeys   []database.ProjectAPIKey
	access    database.ProjectTableAccess
	files     []database.File
}

func (store fakeStore) Ping(context.Context) error {
	return store.pingErr
}

func (store fakeStore) ListProjects(context.Context) ([]database.Project, error) {
	return store.projects, nil
}

func (store fakeStore) CreateProject(_ context.Context, input database.CreateProjectParams) (database.Project, error) {
	if store.createErr != nil {
		return database.Project{}, store.createErr
	}
	return database.Project{
		ID:       "project_1",
		Name:     input.Name,
		Slug:     input.Slug,
		Template: input.Template,
	}, nil
}

func (store fakeStore) Login(_ context.Context, _ database.LoginParams, _ time.Duration) (database.Session, error) {
	if store.loginErr != nil {
		return database.Session{}, store.loginErr
	}
	return database.Session{
		Token: "token",
		User:  store.authUser,
	}, nil
}

func (store fakeStore) Authenticate(context.Context, string) (database.AuthUser, error) {
	if store.authErr != nil {
		return database.AuthUser{}, store.authErr
	}
	return store.authUser, nil
}

func (store fakeStore) AuthenticateProjectAPIKey(context.Context, string) (database.ProjectAPIKey, error) {
	if store.authErr != nil {
		return database.ProjectAPIKey{}, store.authErr
	}
	return database.ProjectAPIKey{ID: "key_1", ProjectID: "project_1", Name: "Client"}, nil
}

func (store fakeStore) ListUsers(context.Context) ([]database.User, error) {
	return store.users, nil
}

func (store fakeStore) CreateUser(_ context.Context, input database.CreateUserParams) (database.User, error) {
	return database.User{
		ID:          "user_1",
		Email:       input.Email,
		DisplayName: input.DisplayName,
	}, nil
}

func (store fakeStore) ListProjectMembers(context.Context, string) ([]database.ProjectMember, error) {
	return store.members, nil
}

func (store fakeStore) AddMembership(_ context.Context, _ string, input database.AddMembershipParams) (database.ProjectMember, error) {
	return database.ProjectMember{
		UserID: input.UserID,
		Role:   input.Role,
	}, nil
}

func (store fakeStore) ListProjectTables(context.Context, string) ([]database.ProjectTable, error) {
	return store.tables, nil
}

func (store fakeStore) CreateProjectTable(_ context.Context, projectID string, input database.CreateProjectTableParams) (database.ProjectTable, error) {
	return database.ProjectTable{
		ID:        "table_1",
		ProjectID: projectID,
		Name:      input.Name,
		Schema:    input.Schema,
	}, nil
}

func (store fakeStore) GetProjectTableAccess(context.Context, string, string) (database.ProjectTableAccess, error) {
	if store.access.ReadAccess != "" || store.access.WriteAccess != "" {
		return store.access, nil
	}
	return database.ProjectTableAccess{
		ReadAccess:  "project_members",
		WriteAccess: "project_members",
	}, nil
}

func (store fakeStore) ListProjectRecords(context.Context, string, string) ([]database.ProjectRecord, error) {
	return store.records, nil
}

func (store fakeStore) CreateProjectRecord(_ context.Context, projectID, _ string, createdByUserID string, input database.CreateProjectRecordParams) (database.ProjectRecord, error) {
	return database.ProjectRecord{
		ID:              "record_1",
		ProjectID:       projectID,
		TableID:         "table_1",
		Data:            input.Data,
		CreatedByUserID: &createdByUserID,
	}, nil
}

func (store fakeStore) UpdateProjectRecord(_ context.Context, projectID, _ string, recordID string, input database.UpdateProjectRecordParams) (database.ProjectRecord, error) {
	return database.ProjectRecord{
		ID:        recordID,
		ProjectID: projectID,
		TableID:   "table_1",
		Data:      input.Data,
	}, nil
}

func (store fakeStore) DeleteProjectRecord(context.Context, string, string, string) error {
	return nil
}

func (store fakeStore) ListProjectAPIKeys(context.Context, string) ([]database.ProjectAPIKey, error) {
	return store.apiKeys, nil
}

func (store fakeStore) CreateProjectAPIKey(_ context.Context, projectID, createdByUserID string, input database.CreateProjectAPIKeyParams) (database.CreatedProjectAPIKey, error) {
	return database.CreatedProjectAPIKey{
		ProjectAPIKey: database.ProjectAPIKey{
			ID:              "key_1",
			ProjectID:       projectID,
			Name:            input.Name,
			TokenPrefix:     "abc123",
			CreatedByUserID: &createdByUserID,
		},
		Token: "token",
	}, nil
}

func (store fakeStore) ListFiles(context.Context, string) ([]database.File, error) {
	return store.files, nil
}

func (store fakeStore) CreateFileIntent(_ context.Context, projectID, ownerUserID, bucket, _ string, input database.CreateFileIntentParams) (database.FileIntent, error) {
	owner := &ownerUserID
	if ownerUserID == "" {
		owner = nil
	}
	file := database.File{
		ID:          "file_1",
		ProjectID:   projectID,
		OwnerUserID: owner,
		Bucket:      bucket,
		ObjectKey:   "projects/project_1/file.pdf",
		FileName:    input.FileName,
		MimeType:    input.MimeType,
		ByteSize:    input.ByteSize,
		Access:      input.Access,
	}
	return database.FileIntent{
		File: file,
		Upload: database.StorageOperation{
			Method: "PUT",
			URL:    "http://localhost:8080/upload",
		},
	}, nil
}

func (store fakeStore) GetFile(context.Context, string, string) (database.File, error) {
	return database.File{ID: "file_1", ProjectID: "project_1", FileName: "file.pdf"}, nil
}

func TestHealth(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected ok health status, got %q", body["status"])
	}
}

func TestReadyDegradedWithoutDatabaseURL(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("expected degraded readiness status, got %q", body["status"])
	}
}

func TestReadyWithHealthyDatabase(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestReadyWithUnhealthyDatabase(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{pingErr: errors.New("offline")})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestIndex(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/v1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestListProjectsRequiresDatabase(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestListProjects(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		projects: []database.Project{
			{ID: "project_1", Name: "CV Builder", Slug: "cv-builder", Template: "blank"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateProject(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{
			Memberships: []database.Membership{{Role: "owner"}},
		},
	})
	body := bytes.NewBufferString(`{"name":"CV Builder","slug":"cv-builder","template":"blank"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestCreateProjectRequiresAuth(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{})
	body := bytes.NewBufferString(`{"name":"CV Builder","slug":"cv-builder","template":"blank"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateProjectRequiresOwner(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{
			Memberships: []database.Membership{{Role: "admin"}},
		},
	})
	body := bytes.NewBufferString(`{"name":"CV Builder","slug":"cv-builder","template":"blank"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestLogin(t *testing.T) {
	handler := NewServer(config.Config{SessionTTLHours: 24}, slog.Default(), fakeStore{})
	body := bytes.NewBufferString(`{"email":"owner@example.com","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLoginRejectsBadCredentials(t *testing.T) {
	handler := NewServer(config.Config{SessionTTLHours: 24}, slog.Default(), fakeStore{loginErr: database.ErrUnauthorized})
	body := bytes.NewBufferString(`{"email":"owner@example.com","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestMe(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{User: database.User{ID: "user_1", Email: "owner@example.com"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestListUsersRequiresOwner(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{Role: "admin"}}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestListUsers(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{Role: "owner"}}},
		users:    []database.User{{ID: "user_1", Email: "owner@example.com", DisplayName: "Owner"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateUser(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{Role: "owner"}}},
	})
	body := bytes.NewBufferString(`{"email":"builder@example.com","display_name":"Builder","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/users", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestListProjectMembersRequiresMembership(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_2", Role: "developer"}}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/members", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestListProjectMembers(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "developer"}}},
		members:  []database.ProjectMember{{UserID: "user_1", Email: "builder@example.com", Role: "developer"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/members", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAddProjectMemberRequiresAdmin(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "developer"}}},
	})
	body := bytes.NewBufferString(`{"user_id":"user_2","role":"developer"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/members", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestAddProjectMember(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "admin"}}},
	})
	body := bytes.NewBufferString(`{"user_id":"user_2","role":"developer"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/members", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestListProjectAPIKeys(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "developer"}}},
		apiKeys:  []database.ProjectAPIKey{{ID: "key_1", ProjectID: "project_1", Name: "Client"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/api-keys", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateProjectAPIKey(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{
			User:        database.User{ID: "user_1"},
			Memberships: []database.Membership{{ProjectID: "project_1", Role: "developer"}},
		},
	})
	body := bytes.NewBufferString(`{"name":"Client"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/api-keys", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestListProjectTables(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
		tables:   []database.ProjectTable{{ID: "table_1", ProjectID: "project_1", Name: "profiles"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/tables", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateProjectTableRequiresDeveloper(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
	})
	body := bytes.NewBufferString(`{"name":"profiles","schema":{"fields":{"name":"text"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/tables", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestCreateProjectTable(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "developer"}}},
	})
	body := bytes.NewBufferString(`{"name":"profiles","schema":{"fields":{"name":"text"}}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/tables", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestListProjectRecords(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{User: database.User{ID: "user_1"}, Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
		records:  []database.ProjectRecord{{ID: "record_1", ProjectID: "project_1", TableID: "table_1"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/tables/profiles/records", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateProjectRecord(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{User: database.User{ID: "user_1"}, Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
	})
	body := bytes.NewBufferString(`{"data":{"name":"Ian","headline":"Educator"}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/tables/profiles/records", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestUpdateProjectRecord(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{User: database.User{ID: "user_1"}, Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
	})
	body := bytes.NewBufferString(`{"data":{"name":"Ian","headline":"Founder"}}`)
	req := httptest.NewRequest(http.MethodPatch, "/v1/projects/project_1/tables/profiles/records/record_1", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestDeleteProjectRecord(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{User: database.User{ID: "user_1"}, Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
	})
	req := httptest.NewRequest(http.MethodDelete, "/v1/projects/project_1/tables/profiles/records/record_1", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}

func TestListProjectRecordsWithAPIKeyPolicy(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		access: database.ProjectTableAccess{ReadAccess: "api_key", WriteAccess: "api_key"},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/tables/profiles/records", nil)
	req.Header.Set("X-Kuza-API-Key", "key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestListProjectRecordsRejectsAPIKeyForMemberPolicy(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		access: database.ProjectTableAccess{ReadAccess: "project_members", WriteAccess: "project_members"},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/tables/profiles/records", nil)
	req.Header.Set("X-Kuza-API-Key", "key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestListFiles(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
		files:    []database.File{{ID: "file_1", ProjectID: "project_1", FileName: "cv.pdf"}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/files", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateFileIntentWithSession(t *testing.T) {
	handler := NewServer(config.Config{StorageBucket: "kuza-core", PublicURL: "http://localhost:8080"}, slog.Default(), fakeStore{
		authUser: database.AuthUser{User: database.User{ID: "user_1"}, Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
	})
	body := bytes.NewBufferString(`{"file_name":"cv.pdf","mime_type":"application/pdf","byte_size":2048,"access":"api_key"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/files", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestCreateFileIntentWithAPIKey(t *testing.T) {
	handler := NewServer(config.Config{StorageBucket: "kuza-core", PublicURL: "http://localhost:8080"}, slog.Default(), fakeStore{})
	body := bytes.NewBufferString(`{"file_name":"avatar.png","mime_type":"image/png","byte_size":1024,"access":"public"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/project_1/files", body)
	req.Header.Set("X-Kuza-API-Key", "key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestGetFile(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{Memberships: []database.Membership{{ProjectID: "project_1", Role: "member"}}},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/projects/project_1/files/file_1", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
