package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/pkg/config"
	razorpay "github.com/razorpay/razorpay-go"
)

// testCfg returns a minimal config usable by payment service tests.
func testCfg() *config.Config {
	return &config.Config{
		RazorpayKeyID:         "rzp_test_key",
		RazorpayKeySecret:     "test_secret",
		RazorpayWebhookSecret: "webhook_secret",
	}
}

func newTestPaymentService(m *mockDeps, cfg *config.Config) *paymentService {
	return &paymentService{
		payments: m.payments,
		plans:    m.plans,
		users:    m.users,
		rzClient: razorpay.NewClient(cfg.RazorpayKeyID, cfg.RazorpayKeySecret),
		cfg:      cfg,
	}
}

func TestPaymentServiceGetPlans(t *testing.T) {
	mock := newMockDeps()
	svc := newTestPaymentService(mock, testCfg())

	plans, err := svc.GetPlans(context.Background())
	if err != nil {
		t.Fatalf("GetPlans returned error: %v", err)
	}
	if len(plans) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(plans))
	}

	planMap := make(map[string]*domain.Plan)
	for i, p := range plans {
		planMap[p.Name] = &plans[i]
	}

	if planMap["free"].DailyLimit != 5 {
		t.Fatalf("free plan: expected daily limit 5, got %d", planMap["free"].DailyLimit)
	}
	if planMap["pro"].PricePaise != 29900 {
		t.Fatalf("pro plan: expected price 29900, got %d", planMap["pro"].PricePaise)
	}
	if planMap["max"].MaxAPIKeys != -1 {
		t.Fatalf("max plan: expected unlimited api keys, got %d", planMap["max"].MaxAPIKeys)
	}
}

func TestPaymentServiceCreateOrderFreePlan(t *testing.T) {
	mock := newMockDeps()
	svc := newTestPaymentService(mock, testCfg())

	user := mock.addUserWithPlan("free")

	_, err := svc.CreateOrder(context.Background(), user.ID, "free")
	if err == nil {
		t.Fatal("expected error purchasing free plan, got nil")
	}
}

func TestPaymentServiceCreateOrderAlreadyOnPlan(t *testing.T) {
	mock := newMockDeps()
	svc := newTestPaymentService(mock, testCfg())

	user := mock.addUserWithPlan("pro")

	_, err := svc.CreateOrder(context.Background(), user.ID, "pro")
	if err == nil {
		t.Fatal("expected error when already on plan, got nil")
	}
}

func TestPaymentServiceCreateOrderNonExistentPlan(t *testing.T) {
	mock := newMockDeps()
	svc := newTestPaymentService(mock, testCfg())

	user := mock.addUserWithPlan("free")

	_, err := svc.CreateOrder(context.Background(), user.ID, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent plan, got nil")
	}
}

func TestPaymentServiceVerifyPaymentSignatureMismatch(t *testing.T) {
	mock := newMockDeps()
	svc := newTestPaymentService(mock, testCfg())

	err := svc.VerifyPayment(context.Background(), "user-id", "order-id", "payment-id", "bad-signature")
	if err == nil {
		t.Fatal("expected error for bad signature, got nil")
	}
}

func TestPaymentServiceVerifyPaymentUserMismatch(t *testing.T) {
	mock := newMockDeps()
	cfg := testCfg()
	svc := newTestPaymentService(mock, cfg)

	user := mock.addUserWithPlan("free")

	// Create a valid signature
	mac := hmac.New(sha256.New, []byte(cfg.RazorpayKeySecret))
	mac.Write([]byte("order-1|pay-1"))
	validSig := hex.EncodeToString(mac.Sum(nil))

	// Store a payment with a different user ID
	otherUser := &domain.User{ID: "user-other", Email: "other@test.com", Name: "Other"}
	mock.users.users[otherUser.ID] = otherUser

	mock.payments.Create(context.Background(), &domain.Payment{
		UserID:          otherUser.ID,
		RazorpayOrderID: "order-1",
		PlanName:        "pro",
		AmountPaise:     29900,
		Currency:        "INR",
		Status:          "created",
	})

	// user tries to verify a payment that belongs to otherUser
	err := svc.VerifyPayment(context.Background(), user.ID, "order-1", "pay-1", validSig)
	if err == nil {
		t.Fatal("expected error for user mismatch, got nil")
	}
}

func TestPaymentServiceVerifyPaymentSuccess(t *testing.T) {
	mock := newMockDeps()
	cfg := testCfg()
	svc := newTestPaymentService(mock, cfg)

	user := mock.addUserWithPlan("free")

	// Store a payment order
	mock.payments.Create(context.Background(), &domain.Payment{
		UserID:          user.ID,
		RazorpayOrderID: "order-success",
		PlanName:        "pro",
		AmountPaise:     29900,
		Currency:        "INR",
		Status:          "created",
	})

	// Create a valid signature
	mac := hmac.New(sha256.New, []byte(cfg.RazorpayKeySecret))
	mac.Write([]byte("order-success|pay-123"))
	validSig := hex.EncodeToString(mac.Sum(nil))

	err := svc.VerifyPayment(context.Background(), user.ID, "order-success", "pay-123", validSig)
	if err != nil {
		t.Fatalf("VerifyPayment returned error: %v", err)
	}

	// Verify the payment was marked paid
	payment, _ := mock.payments.FindByOrderID(context.Background(), "order-success")
	if payment.Status != "paid" {
		t.Fatalf("expected payment status 'paid', got %q", payment.Status)
	}

	// Verify the user was upgraded
	_, plan, _ := mock.users.FindWithPlan(context.Background(), user.ID)
	if plan.Name != "pro" {
		t.Fatalf("expected user plan 'pro', got %q", plan.Name)
	}
}

