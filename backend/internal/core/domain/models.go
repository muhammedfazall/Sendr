package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

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

// EmailAddress represents a named email contact.
type EmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Attachment holds a file to include with the email.
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // base64-encoded
	ContentType string `json:"content_type"`
}

// EmailPayload is the data needed to send a single email.
type EmailPayload struct {
	From        *EmailAddress    `json:"from,omitempty"`
	To          []string         `json:"to"`
	CC          []string         `json:"cc,omitempty"`
	BCC         []string         `json:"bcc,omitempty"`
	Subject     string           `json:"subject"`
	TextBody    string           `json:"text_body,omitempty"`
	HTMLBody    string           `json:"html_body,omitempty"`
	Attachments []Attachment     `json:"attachments,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// UnmarshalJSON handles backward-compat with the legacy single-string
// "to" field and "body"/"html" fields used by previously queued jobs.
func (p *EmailPayload) UnmarshalJSON(data []byte) error {
	type alias struct {
		From        *EmailAddress    `json:"from,omitempty"`
		To          any              `json:"to"`
		CC          []string         `json:"cc,omitempty"`
		BCC         []string         `json:"bcc,omitempty"`
		Subject     string           `json:"subject"`
		TextBody    string           `json:"text_body,omitempty"`
		HTMLBody    string           `json:"html_body,omitempty"`
		Body        string           `json:"body,omitempty"`
		HTML        bool             `json:"html,omitempty"`
		Attachments []Attachment     `json:"attachments,omitempty"`
		Headers     map[string]string `json:"headers,omitempty"`
		Tags        map[string]string `json:"tags,omitempty"`
	}
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return fmt.Errorf("email payload: %w", err)
	}

	p.From = a.From
	p.CC = a.CC
	p.BCC = a.BCC
	p.Subject = a.Subject
	p.TextBody = a.TextBody
	p.HTMLBody = a.HTMLBody
	p.Attachments = a.Attachments
	p.Headers = a.Headers
	p.Tags = a.Tags

	// Handle "to" as either a string or []string
	switch v := a.To.(type) {
	case string:
		if v != "" {
			p.To = []string{v}
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				p.To = append(p.To, s)
			}
		}
	}

	// Handle legacy "body" -> TextBody / HTMLBody mapping
	if a.Body != "" && a.TextBody == "" {
		p.TextBody = a.Body
	}
	if a.HTML && a.HTMLBody == "" {
		p.HTMLBody = a.Body
	}

	return nil
}

// Template is a user-defined email template rendered with Go html/template.
type Template struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	Name             string    `json:"name"`
	SubjectTemplate  string    `json:"subject_template"`
	HTMLTemplate     string    `json:"html_template"`
	TextTemplate     string    `json:"text_template"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EmailEvent represents a delivery event from SendGrid (open, click, bounce, etc.).
type EmailEvent struct {
	ID          string          `json:"id"`
	Email       string          `json:"email"`
	EventType   string          `json:"event_type"`
	SGEventID   string          `json:"sg_event_id,omitempty"`
	SGMessageID string          `json:"sg_message_id,omitempty"`
	JobID       string          `json:"job_id,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Unsubscription records a recipient who has opted out.
type Unsubscription struct {
	Email     string    `json:"email"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
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