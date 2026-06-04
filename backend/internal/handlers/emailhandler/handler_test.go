package emailhandler

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
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type mockEmailSvc struct {
	sendFn func(ctx context.Context, fullKey string, payload domain.EmailPayload) (*domain.Job, error)
}

func (m *mockEmailSvc) Send(ctx context.Context, fullKey string, payload domain.EmailPayload) (*domain.Job, error) {
	return m.sendFn(ctx, fullKey, payload)
}

type mockJobReader struct {
	getByIDFn     func(ctx context.Context, jobID string) (*domain.Job, error)
	listByUserFn  func(ctx context.Context, userID, status string, limit, offset int) ([]domain.Job, error)
}

func (m *mockJobReader) GetByID(ctx context.Context, jobID string) (*domain.Job, error) {
	return m.getByIDFn(ctx, jobID)
}
func (m *mockJobReader) ListByUser(ctx context.Context, userID, status string, limit, offset int) ([]domain.Job, error) {
	return m.listByUserFn(ctx, userID, status, limit, offset)
}

func emailWithClaims(r *http.Request, userID string) *http.Request {
	claims := jwt.MapClaims{"user_id": userID}
	ctx := context.WithValue(r.Context(), middleware.UserClaimsKey, claims)
	return r.WithContext(ctx)
}

func chiCtxEmail(r *http.Request, params map[string]string) *http.Request {
	chiC := chi.NewRouteContext()
	for k, v := range params {
		chiC.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiC))
}

func TestSendSuccess(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return &domain.Job{ID: "job-123", Status: "pending", CreatedAt: time.Now()}, nil
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendInvalidBody(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendMissingFields(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	body := `{"to":"","subject":"","body":""}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendSubjectTooLong(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"` + strings.Repeat("A", 999) + `","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendBodyTooLong(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hi","body":"` + strings.Repeat("B", 50001) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendInvalidEmail(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	body := `{"to":"not-an-email","subject":"Hi","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendHTMLNotSupported(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hi","body":"World","html":true}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendRateLimited(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, constants.ErrRateLimitExceeded
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("X-RateLimit-Reset") == "" {
		t.Fatal("expected X-RateLimit-Reset header")
	}
}

func TestSendInvalidAPIKey(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, constants.ErrAPIKeyInvalid
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSendAPIKeyRevoked(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, constants.ErrAPIKeyRevoked
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSendAPIKeyNotFound(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, constants.ErrAPIKeyNotFound
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSendUserNotFound(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, constants.ErrUserNotFound
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSendGenericError(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, errors.New("unexpected error")
		},
	}
	h := New(email, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"Hello","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSendWhitespaceOnlySubject(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	body := `{"to":"a@b.com","subject":"   ","body":"World"}`
	req := httptest.NewRequest(http.MethodPost, "/emails/send", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer mk_live_test.secret")
	w := httptest.NewRecorder()

	h.Send().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetJobSuccess(t *testing.T) {
	now := time.Now()
	jobs := &mockJobReader{
		getByIDFn: func(_ context.Context, _ string) (*domain.Job, error) {
			return &domain.Job{ID: "job-1", Status: "sent", Retries: 0, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails/job-1", nil)
	req = chiCtxEmail(req, map[string]string{"id": "job-1"})
	w := httptest.NewRecorder()

	h.GetJob().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetJobMissingID(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	req := httptest.NewRequest(http.MethodGet, "/emails/", nil)
	req = chiCtxEmail(req, map[string]string{"id": ""})
	w := httptest.NewRecorder()

	h.GetJob().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetJobNotFound(t *testing.T) {
	jobs := &mockJobReader{
		getByIDFn: func(_ context.Context, _ string) (*domain.Job, error) {
			return nil, errors.New("not found")
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails/job-1", nil)
	req = chiCtxEmail(req, map[string]string{"id": "job-1"})
	w := httptest.NewRecorder()

	h.GetJob().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListSuccess(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _, _ string, _, _ int) ([]domain.Job, error) {
			return []domain.Job{{ID: "job-1", Status: "sent"}}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails?status=sent&limit=10&offset=0", nil)
	req = emailWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListMissingClaims(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListInvalidStatus(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{})

	req := httptest.NewRequest(http.MethodGet, "/emails?status=invalid", nil)
	req = emailWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListDefaultPagination(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _, _ string, limit, offset int) ([]domain.Job, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("expected limit=20, offset=0, got limit=%d, offset=%d", limit, offset)
			}
			return []domain.Job{}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)
	req = emailWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListExceedsMaxLimit(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _, _ string, limit, offset int) ([]domain.Job, error) {
			if limit > 100 {
				t.Fatalf("expected limit capped at 100, got %d", limit)
			}
			return []domain.Job{}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails?limit=999", nil)
	req = emailWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListInvalidQueryParams(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _, _ string, limit, offset int) ([]domain.Job, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("expected defaults, got limit=%d, offset=%d", limit, offset)
			}
			return []domain.Job{}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails?limit=abc&offset=xyz", nil)
	req = emailWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListRepositoryError(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _, _ string, _, _ int) ([]domain.Job, error) {
			return nil, errors.New("db error")
		},
	}
	h := New(&mockEmailSvc{}, jobs)

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)
	req = emailWithClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.List().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
