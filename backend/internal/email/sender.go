package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Sender is the interface the worker depends on — not SendGrid directly.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// RetryableError signals a transient failure — worker will back off and retry.
type RetryableError struct{ Err error }

func (e *RetryableError) Error() string { return e.Err.Error() }

func IsRetryable(err error) bool {
	// var r *RetryableError
	// errors.As would be cleaner but avoids importing errors for one call
	_, ok := err.(*RetryableError)
	return ok
}

// SendGridSender implements Sender using the SendGrid v3 REST API.
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
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.sendgrid.com/v3/mail/send", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return &RetryableError{Err: err} // network failure — always retry
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return &RetryableError{Err: fmt.Errorf("sendgrid %d", resp.StatusCode)}
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid bad request: %d", resp.StatusCode) // non-retryable
	}
	return nil
}