package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redis/go-redis/v9"
)

type mockDB struct {
	err error
}

func (m *mockDB) Ping(_ context.Context) error {
	return m.err
}

type mockRDB struct {
	err error
}

func (m *mockRDB) Ping(_ context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	cmd.SetErr(m.err)
	cmd.SetVal("PONG")
	return cmd
}

func TestHealthCheckAllOK(t *testing.T) {
	s := NewService(&mockDB{}, &mockRDB{})
	h := NewHandler(s)

	result := s.Check(context.Background())
	if result.Status != "ok" {
		t.Fatalf("expected status ok, got %s", result.Status)
	}
	if result.DB != "ok" {
		t.Fatalf("expected db ok, got %s", result.DB)
	}
	if result.Redis != "ok" {
		t.Fatalf("expected redis ok, got %s", result.Redis)
	}

	// Also verify through the HTTP handler
	handler := h.Check()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body HealthStatus
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" || body.DB != "ok" || body.Redis != "ok" {
		t.Fatalf("expected all ok, got %+v", body)
	}
}

func TestHealthCheckDBError(t *testing.T) {
	s := NewService(&mockDB{err: context.DeadlineExceeded}, &mockRDB{})
	h := NewHandler(s)

	result := s.Check(context.Background())
	if result.Status != "degraded" {
		t.Fatalf("expected status degraded, got %s", result.Status)
	}
	if result.DB != "error" {
		t.Fatalf("expected db error, got %s", result.DB)
	}
	if result.Redis != "ok" {
		t.Fatalf("expected redis ok, got %s", result.Redis)
	}

	handler := h.Check()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHealthCheckRedisError(t *testing.T) {
	s := NewService(&mockDB{}, &mockRDB{err: context.DeadlineExceeded})

	result := s.Check(context.Background())
	if result.Status != "degraded" {
		t.Fatalf("expected status degraded, got %s", result.Status)
	}
	if result.DB != "ok" {
		t.Fatalf("expected db ok, got %s", result.DB)
	}
	if result.Redis != "error" {
		t.Fatalf("expected redis error, got %s", result.Redis)
	}
}

func TestHealthCheckBothError(t *testing.T) {
	s := NewService(&mockDB{err: context.DeadlineExceeded}, &mockRDB{err: context.Canceled})

	result := s.Check(context.Background())
	if result.Status != "degraded" {
		t.Fatalf("expected status degraded, got %s", result.Status)
	}
	if result.DB != "error" {
		t.Fatalf("expected db error, got %s", result.DB)
	}
	if result.Redis != "error" {
		t.Fatalf("expected redis error, got %s", result.Redis)
	}
}

func TestHealthHandlerGETOnly(t *testing.T) {
	s := NewService(&mockDB{}, &mockRDB{})
	h := NewHandler(s)

	// Handler should work with GET
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Check().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET, got %d", w.Code)
	}
}

func TestHealthHandlerResponseStructure(t *testing.T) {
	s := NewService(&mockDB{}, &mockRDB{})
	h := NewHandler(s)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.Check().ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	for _, k := range []string{"status", "db", "redis"} {
		if _, ok := body[k]; !ok {
			t.Fatalf("missing key %q in response", k)
		}
	}
}

func TestHealthServiceCheckCalledMultipleTimes(t *testing.T) {
	s := NewService(&mockDB{}, &mockRDB{})

	for i := 0; i < 5; i++ {
		result := s.Check(context.Background())
		if result.Status != "ok" {
			t.Fatalf("iteration %d: expected ok, got %s", i, result.Status)
		}
	}
}
