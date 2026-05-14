package paymenthandler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/constants"
	"github.com/muhammedfazall/Sendr/pkg/response"
)

type Handler struct {
	payments ports.PaymentService
}

func New(payments ports.PaymentService) *Handler {
	return &Handler{payments: payments}
}

// POST /payments/orders — create a Razorpay order for a plan upgrade
func (h *Handler) CreateOrder() http.HandlerFunc {
	type request struct {
		PlanName string `json:"plan_name"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
			return
		}
		userID, _ := claims["user_id"].(string)
		if userID == "" {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
		if req.PlanName == "" {
			response.Error(w, http.StatusBadRequest, "missing_plan", "plan_name is required")
			return
		}

		order, err := h.payments.CreateOrder(r.Context(), userID, req.PlanName)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "order_failed", err.Error())
			return
		}

		response.JSON(w, http.StatusCreated, order)
	}
}

// POST /payments/verify — verify Razorpay payment and upgrade plan
func (h *Handler) VerifyPayment() http.HandlerFunc {
	type request struct {
		OrderID   string `json:"razorpay_order_id"`
		PaymentID string `json:"razorpay_payment_id"`
		Signature string `json:"razorpay_signature"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized", "missing token")
			return
		}
		userID, _ := claims["user_id"].(string)

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
		if req.OrderID == "" || req.PaymentID == "" || req.Signature == "" {
			response.Error(w, http.StatusBadRequest, "missing_fields", "all payment fields required")
			return
		}

		if err := h.payments.VerifyPayment(r.Context(), userID, req.OrderID, req.PaymentID, req.Signature); err != nil {
			writeVerifyError(w, err)
			return
		}

		response.JSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Plan upgraded successfully",
		})
	}
}

func writeVerifyError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, constants.ErrPaymentNotFound):
		response.Error(w, http.StatusNotFound, "payment_not_found", "payment order was not found")
	case errors.Is(err, constants.ErrPaymentInvalidSignature):
		response.Error(w, http.StatusBadRequest, "invalid_signature", "payment signature is invalid")
	case errors.Is(err, constants.ErrPaymentUserMismatch):
		response.Error(w, http.StatusForbidden, "payment_forbidden", "payment does not belong to this user")
	case errors.Is(err, constants.ErrPaymentAlreadyProcessed):
		response.Error(w, http.StatusConflict, "payment_already_processed", "payment has already been processed")
	default:
		response.Error(w, http.StatusBadRequest, "verify_failed", "payment verification failed")
	}
}

// GET /plans — list all available plans (public)
func (h *Handler) ListPlans() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plans, err := h.payments.GetPlans(r.Context())
		if err != nil {
			response.Error(w, http.StatusInternalServerError, "fetch_failed", "could not load plans")
			return
		}
		response.JSON(w, http.StatusOK, plans)
	}
}
