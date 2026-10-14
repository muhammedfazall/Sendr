package paymenthandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
)

type mockPaymentSvc struct {
	createOrderFn   func(ctx context.Context, userID, planName string) (map[string]interface{}, error)
	verifyPaymentFn func(ctx context.Context, userID, orderID, paymentID, signature string) error
	getPlansFn      func(ctx context.Context) ([]domain.Plan, error)
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

func runHandler(method, target, body string, claims jwt.MapClaims, h gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	if claims != nil {
		c.Set(middleware.UserClaimsKey, claims)
	}
	h(c)
	c.Writer.WriteHeaderNow()
	return w
}

func TestCreateOrderSuccess(t *testing.T) {
	mock := &mockPaymentSvc{
		createOrderFn: func(_ context.Context, _ string, _ string) (map[string]interface{}, error) {
			return map[string]interface{}{"id": "order_xyz", "amount": 999}, nil
		},
	}
	h := New(mock)

	w := runHandler(http.MethodPost, "/payments/orders", `{"plan_name":"premium"}`, jwt.MapClaims{"user_id": "user-1"}, h.CreateOrder)

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

	w := runHandler(http.MethodPost, "/payments/orders", `{"plan_name":"premium"}`, nil, h.CreateOrder)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCreateOrderInvalidBody(t *testing.T) {
	h := New(&mockPaymentSvc{})

	w := runHandler(http.MethodPost, "/payments/orders", "not json", jwt.MapClaims{"user_id": "user-1"}, h.CreateOrder)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOrderMissingPlan(t *testing.T) {
	h := New(&mockPaymentSvc{})

	w := runHandler(http.MethodPost, "/payments/orders", `{"plan_name":""}`, jwt.MapClaims{"user_id": "user-1"}, h.CreateOrder)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateOrderServiceError(t *testing.T) {
	mock := &mockPaymentSvc{
		createOrderFn: func(_ context.Context, _ string, _ string) (map[string]interface{}, error) {
			return nil, errors.New("plan not found")
		},
	}
	h := New(mock)

	w := runHandler(http.MethodPost, "/payments/orders", `{"plan_name":"nonexistent"}`, jwt.MapClaims{"user_id": "user-1"}, h.CreateOrder)

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

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`, jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVerifyPaymentMissingClaims(t *testing.T) {
	h := New(&mockPaymentSvc{})

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`, nil, h.VerifyPayment)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestVerifyPaymentInvalidBody(t *testing.T) {
	h := New(&mockPaymentSvc{})

	w := runHandler(http.MethodPost, "/payments/verify", "not json", jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyPaymentMissingFields(t *testing.T) {
	h := New(&mockPaymentSvc{})

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":""}`, jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

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

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`, jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

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

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"wrong"}`, jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

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

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`, jwt.MapClaims{"user_id": "user-2"}, h.VerifyPayment)

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

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`, jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

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

	w := runHandler(http.MethodPost, "/payments/verify", `{"razorpay_order_id":"ord_1","razorpay_payment_id":"pay_1","razorpay_signature":"sig_1"}`, jwt.MapClaims{"user_id": "user-1"}, h.VerifyPayment)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListPlansSuccess(t *testing.T) {
	mock := &mockPaymentSvc{
		getPlansFn: func(_ context.Context) ([]domain.Plan, error) {
			return []domain.Plan{{Name: "free"}, {Name: "premium"}}, nil
		},
	}
	h := New(mock)

	w := runHandler(http.MethodGet, "/plans", "", nil, h.ListPlans)

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

	w := runHandler(http.MethodGet, "/plans", "", nil, h.ListPlans)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
