package webhookhandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

type Handler struct {
	payments ports.PaymentRepository
	users    ports.UserRepository
	cfg      *config.Config
}

func New(payments ports.PaymentRepository, users ports.UserRepository, cfg *config.Config) *Handler {
	return &Handler{payments: payments, users: users, cfg: cfg}
}

func (h *Handler) Handle(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	// Verify the HMAC signature
	receivedSig := c.GetHeader("X-Razorpay-Signature")
	mac := hmac.New(sha256.New, []byte(h.cfg.RazorpayWebhookSecret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expectedSig), []byte(receivedSig)) {
		middleware.Logger(c).Warn("webhook: invalid signature")
		c.Status(http.StatusUnauthorized)
		return
	}

	// Parse the event
	var event struct {
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity struct {
					ID      string `json:"id"`
					OrderID string `json:"order_id"`
					Status  string `json:"status"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		middleware.Logger(c).Error("webhook: unmarshal error", "err", err)
		c.Status(http.StatusBadRequest)
		return
	}

	if event.Event != "payment.captured" {
		c.Status(http.StatusOK) // Acknowledge but ignore
		return
	}

	orderID := event.Payload.Payment.Entity.OrderID
	paymentID := event.Payload.Payment.Entity.ID

	// Find our payment record
	payment, err := h.payments.FindByOrderID(c.Request.Context(), orderID)
	if err != nil {
		middleware.Logger(c).Warn("webhook: payment not found for order", "order_id", orderID, "err", err)
		c.Status(http.StatusOK)
		return
	}

	// If already paid, skip (idempotent)
	if payment.Status == "paid" {
		c.Status(http.StatusOK)
		return
	}

	// Mark paid and upgrade plan
	if err := h.payments.MarkPaid(c.Request.Context(), orderID, paymentID, "webhook"); err != nil {
		middleware.Logger(c).Error("webhook: mark paid failed", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if err := h.users.UpdatePlan(c.Request.Context(), payment.UserID, payment.PlanName); err != nil {
		middleware.Logger(c).Error("webhook: upgrade plan failed", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	middleware.Logger(c).Info("webhook: upgraded user", "user_id", payment.UserID, "plan", payment.PlanName)
	c.Status(http.StatusOK)
}
