package emailsender

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// compile-time check: *MailgunSender implements Sender.
var _ Sender = (*MailgunSender)(nil)

// MailgunSender sends emails via the Mailgun REST API.
type MailgunSender struct {
	domain    string
	apiKey    string
	baseURL   string
	fromEmail string
	fromName  string
}

func NewMailgun(domain, apiKey, baseURL, fromEmail, fromName string) *MailgunSender {
	if baseURL == "" {
		baseURL = "https://api.mailgun.net/v3"
	}
	return &MailgunSender{
		domain:    domain,
		apiKey:    apiKey,
		baseURL:   baseURL,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (s *MailgunSender) Send(ctx context.Context, email *domain.EmailPayload) (*SendResult, error) {
	url := s.baseURL + "/" + s.domain + "/messages"

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	from := s.fromEmail
	if s.fromName != "" {
		from = s.fromName + " <" + s.fromEmail + ">"
	}
	if email.From != nil && email.From.Email != "" {
		from = email.From.Email
		if email.From.Name != "" {
			from = email.From.Name + " <" + email.From.Email + ">"
		}
	}
	if err := w.WriteField("from", from); err != nil {
		return nil, fmt.Errorf("mailgun write from: %w", err)
	}

	for _, addr := range email.To {
		if err := w.WriteField("to", addr); err != nil {
			return nil, fmt.Errorf("mailgun write to: %w", err)
		}
	}

	for _, addr := range email.CC {
		if err := w.WriteField("cc", addr); err != nil {
			return nil, fmt.Errorf("mailgun write cc: %w", err)
		}
	}

	for _, addr := range email.BCC {
		if err := w.WriteField("bcc", addr); err != nil {
			return nil, fmt.Errorf("mailgun write bcc: %w", err)
		}
	}

	if err := w.WriteField("subject", email.Subject); err != nil {
		return nil, fmt.Errorf("mailgun write subject: %w", err)
	}

	if email.TextBody != "" {
		if err := w.WriteField("text", email.TextBody); err != nil {
			return nil, fmt.Errorf("mailgun write text: %w", err)
		}
	}
	if email.HTMLBody != "" {
		if err := w.WriteField("html", email.HTMLBody); err != nil {
			return nil, fmt.Errorf("mailgun write html: %w", err)
		}
	}

	for k, v := range email.Headers {
		if err := w.WriteField("h:"+k, v); err != nil {
			return nil, fmt.Errorf("mailgun write header: %w", err)
		}
	}

	for k, v := range email.Tags {
		if err := w.WriteField("o:tag", k+":"+v); err != nil {
			return nil, fmt.Errorf("mailgun write tag: %w", err)
		}
	}

	for _, a := range email.Attachments {
		decoded, err := base64.StdEncoding.DecodeString(a.Content)
		if err != nil {
			return nil, fmt.Errorf("mailgun decode attachment %s: %w", a.Filename, err)
		}

		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", a.ContentType)
		h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.Filename))

		part, err := w.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("mailgun create attachment part: %w", err)
		}
		if _, err := io.Copy(part, bytes.NewReader(decoded)); err != nil {
			return nil, fmt.Errorf("mailgun write attachment %s: %w", a.Filename, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("mailgun close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("mailgun new request: %w", err)}
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetBasicAuth("api", s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("mailgun request: %w", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &RetryableError{Err: fmt.Errorf("mailgun read response: %w", err)}
	}

	if resp.StatusCode >= 500 {
		return nil, &RetryableError{Err: fmt.Errorf("mailgun %d: %s", resp.StatusCode, string(body))}
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("mailgun bad request: %d %s (won't retry)", resp.StatusCode, string(body))
	}

	var mgResp struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &mgResp); err != nil {
		return nil, fmt.Errorf("mailgun parse response: %w", err)
	}

	return &SendResult{ProviderMessageID: mgResp.ID}, nil
}
