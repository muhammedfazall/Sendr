package webhookhandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

type mockPaymentRepo struct {
	findByOrderIDFn func(ctx context.Context, orderID string) (*domain.Payment, error)
	markPaidFn      func(ctx context.Context, orderID, paymentID, signature string) error
}

func (m *mockPaymentRepo) Create(ctx context.Context, payment *domain.Payment) error { return nil }
func (m *mockPaymentRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return m.findByOrderIDFn(ctx, orderID)
}
func (m *mockPaymentRepo) MarkPaid(ctx context.Context, orderID, paymentID, signature string) error {
	return m.markPaidFn(ctx, orderID, paymentID, signature)
}
func (m *mockPaymentRepo) MarkFailed(ctx context.Context, orderID string) error { return nil }

type mockUserRepoWebhook struct {
	updatePlanFn func(ctx context.Context, userID, planName string) error
}

func (m *mockUserRepoWebhook) UpdatePlan(ctx context.Context, userID, planName string) error {
	return m.updatePlanFn(ctx, userID, planName)
}
func (m *mockUserRepoWebhook) Upsert(ctx context.Context, _, _, _ string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepoWebhook) FindByID(ctx context.Context, _ string) (*domain.User, error) {
	return nil, nil
}
func (m *mockUserRepoWebhook) FindWithPlan(ctx context.Context, _ string) (*domain.User, *domain.Plan, error) {
	return nil, nil, nil
}
func (m *mockUserRepoWebhook) UpdateProfile(ctx context.Context, _, _ string) (*domain.User, error) {
	return nil, nil
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func webhookEventBody(event string, orderID, paymentID string) []byte {
	data := map[string]interface{}{
		"event": event,
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       paymentID,
					"order_id": orderID,
					"status":   "captured",
				},
			},
		},
	}
	b, _ := json.Marshal(data)
	return b
}

func TestWebhookPaymentCapturedSuccess(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	payments := &mockPaymentRepo{
		findByOrderIDFn: func(_ context.Context, orderID string) (*domain.Payment, error) {
			return &domain.Payment{
				RazorpayOrderID: orderID, UserID: "user-1", PlanName: "premium", Status: "created",
			}, nil
		},
		markPaidFn: func(_ context.Context, _, _, _ string) error { return nil },
	}
	users := &mockUserRepoWebhook{
		updatePlanFn: func(_ context.Context, _, _ string) error { return nil },
	}
	h := New(payments, users, cfg)

	body := webhookEventBody("payment.captured", "order_premium", "pay_123")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWebhookInvalidSignature(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	h := New(&mockPaymentRepo{}, &mockUserRepoWebhook{}, cfg)

	body := webhookEventBody("payment.captured", "order_1", "pay_1")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", "invalid_sig")
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWebhookMissingSignature(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	h := New(&mockPaymentRepo{}, &mockUserRepoWebhook{}, cfg)

	body := webhookEventBody("payment.captured", "order_1", "pay_1")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWebhookInvalidBody(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	h := New(&mockPaymentRepo{}, &mockUserRepoWebhook{}, cfg)

	body := []byte("not json")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWebhookWrongEventIgnored(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	h := New(&mockPaymentRepo{}, &mockUserRepoWebhook{}, cfg)

	body := webhookEventBody("payment.failed", "order_1", "pay_1")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWebhookPaymentNotFound(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	payments := &mockPaymentRepo{
		findByOrderIDFn: func(_ context.Context, orderID string) (*domain.Payment, error) {
			return nil, errors.New("not found")
		},
	}
	h := New(payments, &mockUserRepoWebhook{}, cfg)

	body := webhookEventBody("payment.captured", "order_nonexistent", "pay_1")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWebhookAlreadyPaid(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	payments := &mockPaymentRepo{
		findByOrderIDFn: func(_ context.Context, orderID string) (*domain.Payment, error) {
			return &domain.Payment{
				RazorpayOrderID: orderID, UserID: "user-1", PlanName: "premium", Status: "paid",
			}, nil
		},
	}
	h := New(payments, &mockUserRepoWebhook{}, cfg)

	body := webhookEventBody("payment.captured", "order_premium", "pay_1")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWebhookMarkPaidError(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	payments := &mockPaymentRepo{
		findByOrderIDFn: func(_ context.Context, orderID string) (*domain.Payment, error) {
			return &domain.Payment{
				RazorpayOrderID: orderID, UserID: "user-1", PlanName: "premium", Status: "created",
			}, nil
		},
		markPaidFn: func(_ context.Context, _, _, _ string) error {
			return errors.New("db error")
		},
	}
	h := New(payments, &mockUserRepoWebhook{}, cfg)

	body := webhookEventBody("payment.captured", "order_premium", "pay_1")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestWebhookUpdatePlanError(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	payments := &mockPaymentRepo{
		findByOrderIDFn: func(_ context.Context, orderID string) (*domain.Payment, error) {
			return &domain.Payment{
				RazorpayOrderID: orderID, UserID: "user-1", PlanName: "premium", Status: "created",
			}, nil
		},
		markPaidFn: func(_ context.Context, _, _, _ string) error { return nil },
	}
	users := &mockUserRepoWebhook{
		updatePlanFn: func(_ context.Context, _, _ string) error {
			return errors.New("db error")
		},
	}
	h := New(payments, users, cfg)

	body := webhookEventBody("payment.captured", "order_premium", "pay_1")
	sig := signBody(body, "whsec_test")

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", strings.NewReader(string(body)))
	req.Header.Set("X-Razorpay-Signature", sig)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestWebhookReadBodyError(t *testing.T) {
	cfg := &config.Config{RazorpayWebhookSecret: "whsec_test"}
	h := New(&mockPaymentRepo{}, &mockUserRepoWebhook{}, cfg)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", nil)
	w := httptest.NewRecorder()

	h.Handle().ServeHTTP(w, req)

	// With nil body, io.ReadAll returns empty body, HMAC doesn't match empty string
	// since no signature provided, so it's 401
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for nil body, got %d", w.Code)
	}
}
