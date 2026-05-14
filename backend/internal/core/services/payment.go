package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/pkg/config"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	razorpay "github.com/razorpay/razorpay-go"
)

type paymentService struct {
	payments ports.PaymentRepository
	plans    ports.PlanRepository
	users    ports.UserRepository
	rzClient *razorpay.Client
	cfg      *config.Config
}

func NewPaymentService(
	payments ports.PaymentRepository,
	plans ports.PlanRepository,
	users ports.UserRepository,
	cfg *config.Config,
) ports.PaymentService {
	client := razorpay.NewClient(cfg.RazorpayKeyID, cfg.RazorpayKeySecret)
	return &paymentService{
		payments: payments,
		plans:    plans,
		users:    users,
		rzClient: client,
		cfg:      cfg,
	}
}

// CreateOrder creates a Razorpay order and stores it in the DB.
func (s *paymentService) CreateOrder(ctx context.Context, userID, planName string) (map[string]interface{}, error) {
	// 1. Validate: can't "buy" free plan
	if planName == "free" {
		return nil, fmt.Errorf("cannot purchase the free plan")
	}

	// 2. Check user isn't already on this plan
	_, currentPlan, err := s.users.FindWithPlan(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if currentPlan.Name == planName {
		return nil, fmt.Errorf("you are already on the %s plan", planName)
	}

	// 3. Look up the target plan to get the price
	plan, err := s.plans.FindByName(ctx, planName)
	if err != nil {
		return nil, fmt.Errorf("plan not found: %w", err)
	}
	if plan.PricePaise <= 0 {
		return nil, fmt.Errorf("plan %s has no price configured", planName)
	}

	// 4. Create Razorpay order
	orderData := map[string]interface{}{
		"amount":   plan.PricePaise,
		"currency": "INR",
		"receipt":  fmt.Sprintf("sendr_%s_%s", userID[:8], planName),
		"notes": map[string]interface{}{
			"user_id":   userID,
			"plan_name": planName,
		},
	}

	rzOrder, err := s.rzClient.Order.Create(orderData, nil)
	if err != nil {
		return nil, fmt.Errorf("razorpay create order: %w", err)
	}

	orderID, _ := rzOrder["id"].(string)

	// 5. Store in our DB
	payment := &domain.Payment{
		UserID:          userID,
		RazorpayOrderID: orderID,
		PlanName:        planName,
		AmountPaise:     plan.PricePaise,
		Currency:        "INR",
		Status:          "created",
	}
	if err := s.payments.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("store payment: %w", err)
	}

	// 6. Return data the frontend needs to open Razorpay Checkout
	return map[string]interface{}{
		"order_id": orderID,
		"amount":   plan.PricePaise,
		"currency": "INR",
		"key_id":   s.cfg.RazorpayKeyID,
	}, nil
}

// VerifyPayment verifies the Razorpay signature and upgrades the user's plan.
func (s *paymentService) VerifyPayment(ctx context.Context, userID, orderID, paymentID, signature string) error {
	// 1. Verify signature: HMAC-SHA256(order_id + "|" + payment_id, secret)
	message := orderID + "|" + paymentID
	mac := hmac.New(sha256.New, []byte(s.cfg.RazorpayKeySecret))
	mac.Write([]byte(message))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(signature)) {
		return constants.ErrPaymentInvalidSignature
	}

	// 2. Load the payment record to get the plan name
	payment, err := s.payments.FindByOrderID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("load payment: %w", err)
	}

	// 3. Security: ensure the payment belongs to this user
	if payment.UserID != userID {
		return constants.ErrPaymentUserMismatch
	}

	// Repeated Razorpay callbacks should be safe. If the same payment was
	// already recorded, retry the plan update and return success.
	if payment.Status == "paid" {
		if payment.RazorpayPaymentID != nil && *payment.RazorpayPaymentID != paymentID {
			return constants.ErrPaymentAlreadyProcessed
		}
		if err := s.users.UpdatePlan(ctx, userID, payment.PlanName); err != nil {
			return fmt.Errorf("upgrade plan: %w", err)
		}
		return nil
	}

	// 4. Mark payment as paid in our DB
	if err := s.payments.MarkPaid(ctx, orderID, paymentID, signature); err != nil {
		return fmt.Errorf("mark paid: %w", err)
	}

	// 5. Upgrade the user's plan
	if err := s.users.UpdatePlan(ctx, userID, payment.PlanName); err != nil {
		return fmt.Errorf("upgrade plan: %w", err)
	}

	return nil
}

// GetPlans returns all available plans.
func (s *paymentService) GetPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.ListAll(ctx)
}
