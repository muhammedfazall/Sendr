package emailsender

import (
	"context"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// MockSender captures sent emails in memory. Use in tests — no real HTTP calls.
type MockSender struct {
	Sent  []*domain.EmailPayload
	ErrFn func(to string) error // inject per-recipient errors
}

func (m *MockSender) Send(_ context.Context, email *domain.EmailPayload) (*SendResult, error) {
	if m.ErrFn != nil {
		for _, to := range email.To {
			if err := m.ErrFn(to); err != nil {
				return nil, err
			}
		}
	}
	m.Sent = append(m.Sent, email)
	return &SendResult{}, nil
}