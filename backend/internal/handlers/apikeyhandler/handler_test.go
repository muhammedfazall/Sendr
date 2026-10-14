package apikeyhandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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

func runHandler(method, target, body string, claims jwt.MapClaims, params gin.Params, h gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Params = params
	if claims != nil {
		c.Set(middleware.UserClaimsKey, claims)
	}
	h(c)
	c.Writer.WriteHeaderNow()
	return w
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

	w := runHandler(http.MethodPost, "/apikeys", `{"name":"test-key"}`, jwt.MapClaims{"user_id": "user-1"}, nil, h.Create)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAPIKeyMissingClaims(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	w := runHandler(http.MethodPost, "/apikeys", `{"name":"test"}`, nil, nil, h.Create)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateAPIKeyInvalidBody(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	w := runHandler(http.MethodPost, "/apikeys", `not json`, jwt.MapClaims{"user_id": "user-1"}, nil, h.Create)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateAPIKeyEmptyName(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	w := runHandler(http.MethodPost, "/apikeys", `{"name":""}`, jwt.MapClaims{"user_id": "user-1"}, nil, h.Create)

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

	w := runHandler(http.MethodPost, "/apikeys", `{"name":"key"}`, jwt.MapClaims{"user_id": "user-1"}, nil, h.Create)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListAPIKeysSuccess(t *testing.T) {
	mock := &mockAPIKeySvc{
		listFn: func(_ context.Context, _ string) ([]domain.APIKey, error) {
			return []domain.APIKey{{ID: "key-1", Name: "test", Prefix: "sendr_test", CreatedAt: time.Now()}}, nil
		},
	}
	h := New(mock)

	w := runHandler(http.MethodGet, "/apikeys", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListAPIKeysMissingClaims(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	w := runHandler(http.MethodGet, "/apikeys", "", nil, nil, h.List)

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

	w := runHandler(http.MethodGet, "/apikeys", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestRevokeAPIKeySuccess(t *testing.T) {
	mock := &mockAPIKeySvc{
		revokeFn: func(_ context.Context, _, _ string) error { return nil },
	}
	h := New(mock)

	w := runHandler(http.MethodDelete, "/apikeys/key-1", "", jwt.MapClaims{"user_id": "user-1"}, gin.Params{{Key: "id", Value: "key-1"}}, h.Revoke)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestRevokeAPIKeyMissingClaims(t *testing.T) {
	h := New(&mockAPIKeySvc{})

	w := runHandler(http.MethodDelete, "/apikeys/key-1", "", nil, gin.Params{{Key: "id", Value: "key-1"}}, h.Revoke)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRevokeAPIKeyNotFound(t *testing.T) {
	mock := &mockAPIKeySvc{
		revokeFn: func(_ context.Context, _, _ string) error { return errors.New("not found") },
	}
	h := New(mock)

	w := runHandler(http.MethodDelete, "/apikeys/key-1", "", jwt.MapClaims{"user_id": "user-1"}, gin.Params{{Key: "id", Value: "key-1"}}, h.Revoke)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
