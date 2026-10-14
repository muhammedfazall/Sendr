package emailhandler

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
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type mockEmailSvc struct {
	sendFn func(ctx context.Context, fullKey string, payload domain.EmailPayload) (*domain.Job, error)
}

func (m *mockEmailSvc) Send(ctx context.Context, fullKey string, payload domain.EmailPayload) (*domain.Job, error) {
	return m.sendFn(ctx, fullKey, payload)
}

type mockJobReader struct {
	getByIDFn    func(ctx context.Context, jobID string) (*domain.Job, error)
	listByUserFn func(ctx context.Context, userID, status string, limit, offset int) ([]domain.Job, error)
}

func (m *mockJobReader) GetByID(ctx context.Context, jobID string) (*domain.Job, error) {
	return m.getByIDFn(ctx, jobID)
}
func (m *mockJobReader) ListByUser(ctx context.Context, userID, status string, limit, offset int) ([]domain.Job, error) {
	return m.listByUserFn(ctx, userID, status, limit, offset)
}

func runHandler(method, target, body, auth string, claims jwt.MapClaims, params gin.Params, h gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
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

func sendReq(h *Handler, body string) *httptest.ResponseRecorder {
	return runHandler(http.MethodPost, "/emails/send", body, "Bearer mk_live_test.secret", nil, nil, h.Send)
}

func TestSendSuccess(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return &domain.Job{ID: "job-123", Status: "pending", CreatedAt: time.Now()}, nil
		},
	}
	h := New(email, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hello","text_body":"World"}`)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSendInvalidBody(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := sendReq(h, `not json`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendMissingFields(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":[],"subject":"","text_body":""}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendSubjectTooLong(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"`+strings.Repeat("A", 999)+`","text_body":"World"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendBodyTooLong(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hi","text_body":"`+strings.Repeat("B", 50001)+`"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSendInvalidEmail(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["not-an-email"],"subject":"Hi","text_body":"World"}`)

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
	h := New(email, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hello","text_body":"World"}`)

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
	h := New(email, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hello","text_body":"World"}`)

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
	h := New(email, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hello","text_body":"World"}`)

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
	h := New(email, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hello","text_body":"World"}`)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSendGenericError(t *testing.T) {
	email := &mockEmailSvc{
		sendFn: func(_ context.Context, _ string, _ domain.EmailPayload) (*domain.Job, error) {
			return nil, errors.New("boom")
		},
	}
	h := New(email, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"Hello","text_body":"World"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestSendWhitespaceOnlySubject(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := sendReq(h, `{"to":["a@b.com"],"subject":"   ","text_body":"World"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetJobSuccess(t *testing.T) {
	jobs := &mockJobReader{
		getByIDFn: func(_ context.Context, _ string) (*domain.Job, error) {
			return &domain.Job{ID: "job-1", Status: "sent", CreatedAt: time.Now()}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs, nil, nil)

	w := runHandler(http.MethodGet, "/emails/job-1", "", "", nil, gin.Params{{Key: "id", Value: "job-1"}}, h.GetJob)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetJobMissingID(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := runHandler(http.MethodGet, "/emails/", "", "", nil, gin.Params{{Key: "id", Value: ""}}, h.GetJob)

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
	h := New(&mockEmailSvc{}, jobs, nil, nil)

	w := runHandler(http.MethodGet, "/emails/job-1", "", "", nil, gin.Params{{Key: "id", Value: "job-1"}}, h.GetJob)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListSuccess(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _ string, _ string, _ int, _ int) ([]domain.Job, error) {
			return []domain.Job{{ID: "job-1", Status: "sent"}}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs, nil, nil)

	w := runHandler(http.MethodGet, "/emails?status=sent&limit=10&offset=0", "", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListMissingClaims(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := runHandler(http.MethodGet, "/emails", "", "", nil, nil, h.List)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListInvalidStatus(t *testing.T) {
	h := New(&mockEmailSvc{}, &mockJobReader{}, nil, nil)

	w := runHandler(http.MethodGet, "/emails?status=invalid", "", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListDefaultPagination(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _ string, _ string, limit, offset int) ([]domain.Job, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("expected limit=20, offset=0, got limit=%d, offset=%d", limit, offset)
			}
			return []domain.Job{}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs, nil, nil)

	w := runHandler(http.MethodGet, "/emails", "", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListExceedsMaxLimit(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _ string, _ string, limit, offset int) ([]domain.Job, error) {
			if limit > 100 {
				t.Fatalf("expected limit capped at 100, got %d", limit)
			}
			return []domain.Job{}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs, nil, nil)

	w := runHandler(http.MethodGet, "/emails?limit=999", "", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestListInvalidQueryParams(t *testing.T) {
	jobs := &mockJobReader{
		listByUserFn: func(_ context.Context, _ string, _ string, limit, offset int) ([]domain.Job, error) {
			if limit != 20 || offset != 0 {
				t.Fatalf("expected defaults, got limit=%d, offset=%d", limit, offset)
			}
			return []domain.Job{}, nil
		},
	}
	h := New(&mockEmailSvc{}, jobs, nil, nil)

	w := runHandler(http.MethodGet, "/emails?limit=abc&offset=xyz", "", "", jwt.MapClaims{"user_id": "user-1"}, nil, h.List)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
