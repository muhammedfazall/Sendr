package domain

import "time"

// User is the authenticated principal.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	GoogleID  string    `json:"google_id,omitempty"`
	PlanID    string    `json:"plan_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Plan represents a billing tier and its limits.
type Plan struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	DailyLimit   int       `json:"daily_limit"`
	MaxAPIKeys   int       `json:"max_api_keys"`
	RateWaitSecs int       `json:"rate_wait_secs"`
	PricePaise   int       `json:"price_paise"`
	CreatedAt    time.Time `json:"created_at"`
}

// APIKey is the domain representation of an API credential.
// Hashed is never transmitted — only stored and compared.
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id,omitempty"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	Hashed    string    `json:"-"` // SHA-256 of the secret half — never exposed
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// Job represents a queued email send task.
type Job struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	APIKeyID    string       `json:"api_key_id"`
	Payload     EmailPayload `json:"payload,omitempty"`
	Status      string       `json:"status"`
	Retries     int          `json:"retries"`
	MaxRetries  int          `json:"max_retries"`
	RunAt       time.Time    `json:"run_at"`
	LockedUntil *time.Time   `json:"locked_until,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// EmailPayload is the data needed to send a single email.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
	HTML    bool   `json:"html"`
}

// Payment tracks a Razorpay payment for a plan upgrade.
type Payment struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	RazorpayOrderID    string    `json:"razorpay_order_id"`
	RazorpayPaymentID  *string    `json:"razorpay_payment_id,omitempty"`
	RazorpaySignature  *string    `json:"-"`
	PlanName           string    `json:"plan_name"`
	AmountPaise        int       `json:"amount_paise"`
	Currency           string    `json:"currency"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}