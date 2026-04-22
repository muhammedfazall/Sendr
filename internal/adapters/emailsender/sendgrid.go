package emailsender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// SendGridSender sends emails via the SendGrid v3 REST API.
// No SDK — just net/http. Keeps the dependency tree lean.
type SendGridSender struct {
	apiKey    string
	fromEmail string
	fromName  string
	client    *http.Client
}

func NewSendGrid(apiKey, fromEmail, fromName string) *SendGridSender {
	return &SendGridSender{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		fromName:  fromName,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Send delivers an email. Returns a RetryableError on 5xx/network failures,
// and a plain error on 4xx (bad request — retrying won't help).
func (s *SendGridSender) Send(ctx context.Context, to, subject, body string) error {
	payload := map[string]any{
		"personalizations": []map[string]any{
			{"to": []map[string]string{{"email": to}}},
		},
		"from":    map[string]string{"email": s.fromEmail, "name": s.fromName},
		"subject": subject,
		"content": []map[string]string{{"type": "text/plain", "value": body}},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendgrid payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.sendgrid.com/v3/mail/send", bytes.NewReader(b))
	if err != nil {
		return &RetryableError{Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return &RetryableError{Err: err} // network failure — safe to retry
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &RetryableError{Err: fmt.Errorf("sendgrid %d", resp.StatusCode)}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid bad request: %d (won't retry)", resp.StatusCode)
	}
	return nil
}

// RetryableError signals a transient failure — the worker will back off and retry.
type RetryableError struct{ Err error }

func (e *RetryableError) Error() string { return e.Err.Error() }

// IsRetryable reports whether an error should trigger a worker retry.
// errors.As walks the chain so wrapped RetryableErrors are also caught.
func IsRetryable(err error) bool {
	var r *RetryableError
	return errors.As(err, &r)
}

// Sender is the interface the worker depends on.
// Using an interface here means tests swap in MockSender with zero changes.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}