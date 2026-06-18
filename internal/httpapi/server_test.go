package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"kuza-core/internal/config"
)

func TestHealth(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default())
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
	handler := NewServer(config.Config{}, slog.Default())
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

func TestIndex(t *testing.T) {
	handler := NewServer(config.Config{}, slog.Default())
	req := httptest.NewRequest(http.MethodGet, "/v1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
