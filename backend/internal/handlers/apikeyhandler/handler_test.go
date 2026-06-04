package apikeyhandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/middleware"
)

type mockAPIKeySvc struct {
	createFn func(ctx context.Context, userID, name string) (string, *domain.APIKey, error)
	listFn   func(ctx context.Context, userID string) ([]domain.APIKey, error)
	revokeFn func(ctx context.Context, keyID, userID string) error
}

func (m *mockAPIKeySvc) Create(ctx context.Context, userID, name string) (string, *domain.APIKey, error) {
	return m.createFn(ctx, userID, name)
}
func (m *mockAPIKeySvc) List(ctx context.Context, userID string) ([]domain.APIKey, error) {
	return m.listFn(ctx, userID)
}
func (m *mockAPIKeySvc) Revoke(ctx context.Context, keyID, userID string) error {
	return m.revokeFn(ctx, keyID, userID)
}
func (m *mockAPIKeySvc) Validate(ctx context.Context, fullKey string) (*domain.APIKey, error) {
	return nil, nil
}

func apikeyWithClaims(r *http.Request, userID string) *http.Request {
	claims := jwt.MapClaims{"user_id": userID}
	ctx := context.WithValue(r.Context(), middleware.UserClaimsKey, claims)
	return r.WithContext(ctx)
}

func chiCtx(r *http.Request, params map[string]string) *http.Request {
	chiC := chi.NewRouteContext()
	for k, v := range params {
		chiC.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiC))
}

func TestCreateAPIKeySuccess(t *testing.T) {
	mock := &mockAPIKeySvc{
		createFn: func(_ context.Context, _, name string) (string, *domain.APIKey, error) {
			return "sendr_" + name + "_secret", &domain.APIKey{
				ID: "key-1", Name: name, Prefix: "sendr_" + name, CreatedAt: time.Now(),
			}, nil
		},
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(`{"name":"test-key"}`))
	req.Header.Set("Content-Type", "application/json")
	req = apikeyWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Create().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAPIKeyMissingClaims(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	req := httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateAPIKeyInvalidBody(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	req := httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req = apikeyWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Create().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKeyEmptyName(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	req := httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	req = apikeyWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Create().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKeyServiceError(t *testing.T) {
	mock := &mockAPIKeySvc{
		createFn: func(_ context.Context, _, _ string) (string, *domain.APIKey, error) {
			return "", nil, errors.New("create failed")
		},
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodPost, "/apikeys", strings.NewReader(`{"name":"key"}`))
	req.Header.Set("Content-Type", "application/json")
	req = apikeyWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Create().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListAPIKeysSuccess(t *testing.T) {
	mock := &mockAPIKeySvc{
		listFn: func(_ context.Context, _ string) ([]domain.APIKey, error) {
			return []domain.APIKey{{ID: "key-1", Name: "test", Prefix: "sendr_t", CreatedAt: time.Now()}}, nil
		},
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/apikeys", nil)
	req = apikeyWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListAPIKeysMissingClaims(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	req := httptest.NewRequest(http.MethodGet, "/apikeys", nil)
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListAPIKeysServiceError(t *testing.T) {
	mock := &mockAPIKeySvc{
		listFn: func(_ context.Context, _ string) ([]domain.APIKey, error) {
			return nil, errors.New("list failed")
		},
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/apikeys", nil)
	req = apikeyWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRevokeAPIKeySuccess(t *testing.T) {
	mock := &mockAPIKeySvc{
		revokeFn: func(_ context.Context, _, _ string) error { return nil },
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodDelete, "/apikeys/key-1", nil)
	req = apikeyWithClaims(req, "user-1")
	req = chiCtx(req, map[string]string{"id": "key-1"})
	w := httptest.NewRecorder()

	h.Revoke().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestRevokeAPIKeyMissingClaims(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	req := httptest.NewRequest(http.MethodDelete, "/apikeys/key-1", nil)
	w := httptest.NewRecorder()

	h.Revoke().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRevokeAPIKeyNotFound(t *testing.T) {
	mock := &mockAPIKeySvc{
		revokeFn: func(_ context.Context, _, _ string) error { return errors.New("not found") },
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodDelete, "/apikeys/key-1", nil)
	req = apikeyWithClaims(req, "user-1")
	req = chiCtx(req, map[string]string{"id": "key-1"})
	w := httptest.NewRecorder()

	h.Revoke().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
