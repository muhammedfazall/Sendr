package emailsender

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
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
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Send delivers an email. Returns a RetryableError on 5xx/network failures,
// and a plain error on 4xx (bad request — retrying won't help).
func (s *SendGridSender) Send(ctx context.Context, email *domain.EmailPayload) error {
	from := map[string]string{"email": s.fromEmail, "name": s.fromName}
	if email.From != nil && email.From.Email != "" {
		from = map[string]string{"email": email.From.Email}
		if email.From.Name != "" {
			from["name"] = email.From.Name
		}
	}

	personalization := map[string]any{}
	if len(email.To) > 0 {
		to := make([]map[string]string, len(email.To))
		for i, addr := range email.To {
			to[i] = map[string]string{"email": addr}
		}
		personalization["to"] = to
	}
	if len(email.CC) > 0 {
		cc := make([]map[string]string, len(email.CC))
		for i, addr := range email.CC {
			cc[i] = map[string]string{"email": addr}
		}
		personalization["cc"] = cc
	}
	if len(email.BCC) > 0 {
		bcc := make([]map[string]string, len(email.BCC))
		for i, addr := range email.BCC {
			bcc[i] = map[string]string{"email": addr}
		}
		personalization["bcc"] = bcc
	}
	if len(email.Headers) > 0 {
		personalization["headers"] = email.Headers
	}

	payload := map[string]any{
		"personalizations": []any{personalization},
		"from":             from,
		"subject":          email.Subject,
	}

	var content []map[string]string
	if email.HTMLBody != "" {
		content = append(content, map[string]string{"type": "text/html", "value": email.HTMLBody})
	}
	if email.TextBody != "" {
		content = append(content, map[string]string{"type": "text/plain", "value": email.TextBody})
	}
	if len(content) > 0 {
		payload["content"] = content
	}

	if len(email.Attachments) > 0 {
		atts := make([]map[string]string, len(email.Attachments))
		for i, a := range email.Attachments {
			atts[i] = map[string]string{
				"filename": a.Filename,
				"content":  a.Content,
				"type":     a.ContentType,
			}
		}
		payload["attachments"] = atts
	}

	if len(email.Tags) > 0 {
		var categories []string
		for k, v := range email.Tags {
			categories = append(categories, k+":"+v)
		}
		payload["categories"] = categories
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
	Send(ctx context.Context, email *domain.EmailPayload) error
}
