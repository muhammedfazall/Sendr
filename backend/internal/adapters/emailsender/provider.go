package emailsender

import (
	"fmt"

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


