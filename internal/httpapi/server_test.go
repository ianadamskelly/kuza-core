package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kuza-core/internal/config"
)

type healthCheckerFunc func(context.Context) error

func (fn healthCheckerFunc) Ping(ctx context.Context) error {
	return fn(ctx)
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
	handler := NewServer(config.Config{}, slog.Default(), healthCheckerFunc(func(context.Context) error {
		return nil
	}))
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestReadyWithUnhealthyDatabase(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default(), healthCheckerFunc(func(context.Context) error {
		return errors.New("offline")
	}))
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
