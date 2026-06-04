package mehandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/middleware"
)

type mockUserRepo struct {
	findWithPlanFn  func(ctx context.Context, id string) (*domain.User, *domain.Plan, error)
	updateProfileFn func(ctx context.Context, userID, name string) (*domain.User, error)
}

func (m *mockUserRepo) FindWithPlan(ctx context.Context, id string) (*domain.User, *domain.Plan, error) {
	return m.findWithPlanFn(ctx, id)
}
func (m *mockUserRepo) UpdateProfile(ctx context.Context, userID, name string) (*domain.User, error) {
	return m.updateProfileFn(ctx, userID, name)
}
func (m *mockUserRepo) Upsert(ctx context.Context, googleID, email, name string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) UpdatePlan(ctx context.Context, userID, planName string) error {
	return nil
}

type mockLimiter struct {
	getCountFn func(ctx context.Context, userID string) (int, error)
	checkFn    func(ctx context.Context, userID string, limit int) (bool, int, error)
}

func (m *mockLimiter) GetCount(ctx context.Context, userID string) (int, error) {
	return m.getCountFn(ctx, userID)
}
func (m *mockLimiter) Check(ctx context.Context, userID string, limit int) (bool, int, error) {
	return m.checkFn(ctx, userID, limit)
}

func meWithClaims(r *http.Request, userID string) *http.Request {
	claims := jwt.MapClaims{"user_id": userID}
	ctx := context.WithValue(r.Context(), middleware.UserClaimsKey, claims)
	return r.WithContext(ctx)
}

func TestGetMeSuccess(t *testing.T) {
	now := time.Now()
	users := &mockUserRepo{
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: now},
				&domain.Plan{Name: "free", DailyLimit: 100, MaxAPIKeys: 2, RateWaitSecs: 10}, nil
		},
	}
	limiter := &mockLimiter{
		getCountFn: func(_ context.Context, _ string) (int, error) { return 5, nil },
	}
	h := New(users, limiter)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetMeMissingClaims(t *testing.T) {
	h := New(&mockUserRepo{}, &mockLimiter{})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetMeFindWithPlanError(t *testing.T) {
	users := &mockUserRepo{
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return nil, nil, errors.New("db error")
		},
	}
	h := New(users, &mockLimiter{})

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestGetMeUsageSnapshotWithLimit(t *testing.T) {
	now := time.Now()
	users := &mockUserRepo{
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: now},
				&domain.Plan{Name: "premium", DailyLimit: 1000, MaxAPIKeys: 10, RateWaitSecs: 5}, nil
		},
	}
	limiter := &mockLimiter{
		getCountFn: func(_ context.Context, _ string) (int, error) { return 0, nil },
	}
	h := New(users, limiter)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetMeUsageSnapshotExhausted(t *testing.T) {
	now := time.Now()
	users := &mockUserRepo{
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", CreatedAt: now},
				&domain.Plan{Name: "free", DailyLimit: 100, MaxAPIKeys: 2, RateWaitSecs: 10}, nil
		},
	}
	limiter := &mockLimiter{
		getCountFn: func(_ context.Context, _ string) (int, error) { return 999, nil },
	}
	h := New(users, limiter)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetMeUsageSnapshotUnlimited(t *testing.T) {
	now := time.Now()
	users := &mockUserRepo{
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", CreatedAt: now},
				&domain.Plan{Name: "unlimited", DailyLimit: -1, MaxAPIKeys: -1, RateWaitSecs: 0}, nil
		},
	}
	limiter := &mockLimiter{
		getCountFn: func(_ context.Context, _ string) (int, error) { return 0, nil },
	}
	h := New(users, limiter)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetMeUsageSnapshotGetCountError(t *testing.T) {
	now := time.Now()
	users := &mockUserRepo{
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", CreatedAt: now},
				&domain.Plan{Name: "free", DailyLimit: 100, MaxAPIKeys: 2, RateWaitSecs: 10}, nil
		},
	}
	limiter := &mockLimiter{
		getCountFn: func(_ context.Context, _ string) (int, error) { return 0, errors.New("redis down") },
	}
	h := New(users, limiter)

	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Get().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpdateMeSuccess(t *testing.T) {
	now := time.Now()
	users := &mockUserRepo{
		updateProfileFn: func(_ context.Context, _, name string) (*domain.User, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", Name: name, CreatedAt: now}, nil
		},
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", Name: "Alice", CreatedAt: now},
				&domain.Plan{Name: "free", DailyLimit: 100, MaxAPIKeys: 2, RateWaitSecs: 10}, nil
		},
	}
	h := New(users, &mockLimiter{getCountFn: func(_ context.Context, _ string) (int, error) { return 0, nil }})

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"name":"Bob"}`))
	req.Header.Set("Content-Type", "application/json")
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMeMissingClaims(t *testing.T) {
	h := New(&mockUserRepo{}, &mockLimiter{})

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"name":"Bob"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateMeInvalidBody(t *testing.T) {
	h := New(&mockUserRepo{}, &mockLimiter{})

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateMeNameTooShort(t *testing.T) {
	h := New(&mockUserRepo{}, &mockLimiter{})

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"name":"A"}`))
	req.Header.Set("Content-Type", "application/json")
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateMeNameTooLong(t *testing.T) {
	h := New(&mockUserRepo{}, &mockLimiter{})

	longName := strings.Repeat("A", 81)
	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"name":"`+longName+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateMeUpdateProfileError(t *testing.T) {
	users := &mockUserRepo{
		updateProfileFn: func(_ context.Context, _, _ string) (*domain.User, error) {
			return nil, errors.New("db error")
		},
	}
	h := New(users, &mockLimiter{})

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"name":"Bob"}`))
	req.Header.Set("Content-Type", "application/json")
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestUpdateMeFindWithPlanAfterUpdateError(t *testing.T) {
	users := &mockUserRepo{
		updateProfileFn: func(_ context.Context, _, name string) (*domain.User, error) {
			return &domain.User{ID: "user-1", Email: "a@b.com", Name: name}, nil
		},
		findWithPlanFn: func(_ context.Context, _ string) (*domain.User, *domain.Plan, error) {
			return nil, nil, errors.New("db error on refetch")
		},
	}
	h := New(users, &mockLimiter{})

	req := httptest.NewRequest(http.MethodPatch, "/me", strings.NewReader(`{"name":"Bob"}`))
	req.Header.Set("Content-Type", "application/json")
	req = meWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.Update().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
