package paymenthandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type mockPaymentSvc struct {
	createOrderFn    func(ctx context.Context, userID, planName string) (map[string]interface{}, error)
	verifyPaymentFn  func(ctx context.Context, userID, orderID, paymentID, signature string) error
	getPlansFn       func(ctx context.Context) ([]domain.Plan, error)
}

func (m *mockPaymentSvc) CreateOrder(ctx context.Context, userID, planName string) (map[string]interface{}, error) {
	return m.createOrderFn(ctx, userID, planName)
}
func (m *mockPaymentSvc) VerifyPayment(ctx context.Context, userID, orderID, paymentID, signature string) error {
	return m.verifyPaymentFn(ctx, userID, orderID, paymentID, signature)
}
func (m *mockPaymentSvc) GetPlans(ctx context.Context) ([]domain.Plan, error) {
	return m.getPlansFn(ctx)
}

func withClaims(r *http.Request, userID string) *http.Request {
	claims := jwt.MapClaims{"user_id": userID}
	ctx := context.WithValue(r.Context(), middleware.UserClaimsKey, claims)
	return r.WithContext(ctx)
}

func TestCreateOrderSuccess(t *testing.T) {
	mock := &mockPaymentSvc{
		createOrderFn: func(_ context.Context, _, _ string) (map[string]interface{}, error) {
			return map[string]interface{}{"id": "order_xyz", "amount": 999}, nil
		},
	}
	h := New(mock)

	body := `{"plan_name":"premium"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.CreateOrder().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "order_xyz" {
		t.Fatalf("expected order id 'order_xyz', got %v", resp["id"])
	}
}

func TestCreateOrderMissingClaims(t *testing.T) {
	h := New(&mockPaymentSvc{})

	body := `{"plan_name":"premium"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateOrder().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateOrderInvalidBody(t *testing.T) {
	h := New(&mockPaymentSvc{})

	req := httptest.NewRequest(http.MethodPost, "/payments/orders", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.CreateOrder().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOrderMissingPlan(t *testing.T) {
	h := New(&mockPaymentSvc{})

	body := `{"plan_name":""}`
	req := httptest.NewRequest(http.MethodPost, "/payments/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.CreateOrder().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOrderServiceError(t *testing.T) {
	mock := &mockPaymentSvc{
		createOrderFn: func(_ context.Context, _, _ string) (map[string]interface{}, error) {
			return nil, errors.New("plan not found")
		},
	}
	h := New(mock)

	body := `{"plan_name":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.CreateOrder().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyPaymentSuccess(t *testing.T) {
	mock := &mockPaymentSvc{
		verifyPaymentFn: func(_ context.Context, _, _, _, _ string) error {
			return nil
		},
	}
	h := New(mock)

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVerifyPaymentMissingClaims(t *testing.T) {
	h := New(&mockPaymentSvc{})

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestVerifyPaymentInvalidBody(t *testing.T) {
	h := New(&mockPaymentSvc{})

	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyPaymentMissingFields(t *testing.T) {
	h := New(&mockPaymentSvc{})

	body := `{"razorpay_order_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyPaymentNotFound(t *testing.T) {
	mock := &mockPaymentSvc{
		verifyPaymentFn: func(_ context.Context, _, _, _, _ string) error {
			return constants.ErrPaymentNotFound
		},
	}
	h := New(mock)

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestVerifyPaymentInvalidSignature(t *testing.T) {
	mock := &mockPaymentSvc{
		verifyPaymentFn: func(_ context.Context, _, _, _, _ string) error {
			return constants.ErrPaymentInvalidSignature
		},
	}
	h := New(mock)

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyPaymentUserMismatch(t *testing.T) {
	mock := &mockPaymentSvc{
		verifyPaymentFn: func(_ context.Context, _, _, _, _ string) error {
			return constants.ErrPaymentUserMismatch
		},
	}
	h := New(mock)

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-2")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestVerifyPaymentAlreadyProcessed(t *testing.T) {
	mock := &mockPaymentSvc{
		verifyPaymentFn: func(_ context.Context, _, _, _, _ string) error {
			return constants.ErrPaymentAlreadyProcessed
		},
	}
	h := New(mock)

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestVerifyPaymentUnknownError(t *testing.T) {
	mock := &mockPaymentSvc{
		verifyPaymentFn: func(_ context.Context, _, _, _, _ string) error {
			return errors.New("some unknown error")
		},
	}
	h := New(mock)

	body := `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withClaims(req, "user-1")
	w := httptest.NewRecorder()

	h.VerifyPayment().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListPlansSuccess(t *testing.T) {
	mock := &mockPaymentSvc{
		getPlansFn: func(_ context.Context) ([]domain.Plan, error) {
			return []domain.Plan{
				{Name: "free", PricePaise: 0},
				{Name: "premium", PricePaise: 999},
			}, nil
		},
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	w := httptest.NewRecorder()

	h.ListPlans().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var plans []domain.Plan
	if err := json.NewDecoder(w.Body).Decode(&plans); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}
}

func TestListPlansServiceError(t *testing.T) {
	mock := &mockPaymentSvc{
		getPlansFn: func(_ context.Context) ([]domain.Plan, error) {
			return nil, errors.New("db error")
		},
	}
	h := New(mock)

	req := httptest.NewRequest(http.MethodGet, "/plans", nil)
	w := httptest.NewRecorder()

	h.ListPlans().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
