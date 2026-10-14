package paymenthandler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
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
func (h *Handler) CreateOrder(c *gin.Context) {
	type request struct {
		PlanName string `json:"plan_name"`
	}

	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}
	userID, _ := claims["user_id"].(string)
	if userID == "" {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "invalid token")
		return
	}

	var req request
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.PlanName == "" {
		response.Error(c, http.StatusBadRequest, "missing_plan", "plan_name is required")
		return
	}

	order, err := h.payments.CreateOrder(c.Request.Context(), userID, req.PlanName)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "order_failed", err.Error())
		return
	}

	response.JSON(c, http.StatusCreated, order)
}

// POST /payments/verify — verify Razorpay payment and upgrade plan
func (h *Handler) VerifyPayment(c *gin.Context) {
	type request struct {
		OrderID   string `json:"razorpay_order_id"`
		PaymentID string `json:"razorpay_payment_id"`
		Signature string `json:"razorpay_signature"`
	}

	claims, ok := middleware.GetClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "missing token")
		return
	}
	userID, _ := claims["user_id"].(string)

	var req request
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}
	if req.OrderID == "" || req.PaymentID == "" || req.Signature == "" {
		response.Error(c, http.StatusBadRequest, "missing_fields", "all payment fields required")
		return
	}

	if err := h.payments.VerifyPayment(c.Request.Context(), userID, req.OrderID, req.PaymentID, req.Signature); err != nil {
		writeVerifyError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Plan upgraded successfully",
	})
}

func writeVerifyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, constants.ErrPaymentNotFound):
		response.Error(c, http.StatusNotFound, "payment_not_found", "payment order was not found")
	case errors.Is(err, constants.ErrPaymentInvalidSignature):
		response.Error(c, http.StatusBadRequest, "invalid_signature", "payment signature is invalid")
	case errors.Is(err, constants.ErrPaymentUserMismatch):
		response.Error(c, http.StatusForbidden, "payment_forbidden", "payment does not belong to this user")
	case errors.Is(err, constants.ErrPaymentAlreadyProcessed):
		response.Error(c, http.StatusConflict, "payment_already_processed", "payment has already been processed")
	default:
		response.Error(c, http.StatusBadRequest, "verify_failed", "payment verification failed")
	}
}

// GET /plans — list all available plans (public)
func (h *Handler) ListPlans(c *gin.Context) {
	plans, err := h.payments.GetPlans(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "fetch_failed", "could not load plans")
		return
	}
	response.JSON(c, http.StatusOK, plans)
}
