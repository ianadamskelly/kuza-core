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
	pingErr       error
	organizations []database.Organization
	createErr     error
	loginErr      error
	authErr       error
	authUser      database.AuthUser
}

func (store fakeStore) Ping(context.Context) error {
	return store.pingErr
}

func (store fakeStore) ListOrganizations(context.Context) ([]database.Organization, error) {
	return store.organizations, nil
}

func (store fakeStore) CreateOrganization(_ context.Context, input database.CreateOrganizationParams) (database.Organization, error) {
	if store.createErr != nil {
		return database.Organization{}, store.createErr
	}
	return database.Organization{
		ID:   "org_1",
		Name: input.Name,
		Slug: input.Slug,
		Kind: input.Kind,
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

func TestListOrganizationsRequiresDatabase(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/organizations", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestListOrganizations(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		organizations: []database.Organization{
			{ID: "org_1", Name: "Kuza Kizazi", Slug: "kuza-kizazi", Kind: "school"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/organizations", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateOrganization(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{
			Memberships: []database.Membership{{Role: "owner"}},
		},
	})
	body := bytes.NewBufferString(`{"name":"Example School","slug":"example-school","kind":"school"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations", body)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestCreateOrganizationRequiresAuth(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{})
	body := bytes.NewBufferString(`{"name":"Example School","slug":"example-school","kind":"school"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestCreateOrganizationRequiresOwner(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), fakeStore{
		authUser: database.AuthUser{
			Memberships: []database.Membership{{Role: "admin"}},
		},
	})
	body := bytes.NewBufferString(`{"name":"Example School","slug":"example-school","kind":"school"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/organizations", body)
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
