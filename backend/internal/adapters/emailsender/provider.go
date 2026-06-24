package emailsender

import (
	"context"
	"fmt"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/pkg/config"
)

// NewSender creates the appropriate email sender based on configuration.
// This is the single point of provider selection — add new providers here.
func NewSender(cfg *config.Config) (Sender, error) {
	switch cfg.EmailProvider {
	case config.ProviderSendGrid:
		return NewSendGrid(cfg.SendGridKey, cfg.FromEmail, cfg.FromName), nil

	case config.ProviderMailgun:
		return NewMailgun(cfg.MailgunDomain, cfg.MailgunKey, cfg.MailgunBaseURL, cfg.FromEmail, cfg.FromName), nil

	case config.ProviderSMTP:
		return NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail, cfg.FromName), nil

	case config.ProviderMock:
		return &MockSender{}, nil

	default:
		return nil, fmt.Errorf("unknown email provider: %q", cfg.EmailProvider)
	}
}

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

func (s *MailgunSender) Send(ctx context.Context, email *domain.EmailPayload) error {
	return fmt.Errorf("MailgunSender: not yet implemented")
}

// compile-time check: *SMTPSender implements Sender.
var _ Sender = (*SMTPSender)(nil)

// SMTPSender sends emails via an SMTP relay.
type SMTPSender struct {
	host      string
	port      string
	user      string
	pass      string
	fromEmail string
	fromName  string
}

func NewSMTP(host, port, user, pass, fromEmail, fromName string) *SMTPSender {
	return &SMTPSender{
		host:      host,
		port:      port,
		user:      user,
		pass:      pass,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (s *SMTPSender) Send(ctx context.Context, email *domain.EmailPayload) error {
	return fmt.Errorf("SMTPSender: not yet implemented")
}
