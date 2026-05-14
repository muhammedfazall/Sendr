package webhookhandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/core/ports"
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

func (h *Handler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify webhook signature
		receivedSig := r.Header.Get("X-Razorpay-Signature")
		mac := hmac.New(sha256.New, []byte(h.cfg.RazorpayWebhookSecret))
		mac.Write(body)
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(expectedSig), []byte(receivedSig)) {
			log.Println("webhook: invalid signature")
			w.WriteHeader(http.StatusUnauthorized)
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
			log.Printf("webhook: unmarshal error: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if event.Event != "payment.captured" {
			w.WriteHeader(http.StatusOK) // Acknowledge but ignore
			return
		}

		orderID := event.Payload.Payment.Entity.OrderID
		paymentID := event.Payload.Payment.Entity.ID

		// Find our payment record
		payment, err := h.payments.FindByOrderID(r.Context(), orderID)
		if err != nil {
			log.Printf("webhook: payment not found for order %s: %v", orderID, err)
			w.WriteHeader(http.StatusOK)
			return
		}

		// If already paid, skip (idempotent)
		if payment.Status == "paid" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Mark paid and upgrade plan
		if err := h.payments.MarkPaid(r.Context(), orderID, paymentID, "webhook"); err != nil {
			log.Printf("webhook: mark paid failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := h.users.UpdatePlan(r.Context(), payment.UserID, payment.PlanName); err != nil {
			log.Printf("webhook: upgrade plan failed: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("webhook: upgraded user %s to %s via webhook", payment.UserID, payment.PlanName)
		w.WriteHeader(http.StatusOK)
	}
}