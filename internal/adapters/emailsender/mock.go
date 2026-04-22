package emailsender

import "context"

// MockSender captures sent emails in memory. Use in tests — no real HTTP calls.
type MockSender struct {
	Sent  []SentEmail
	ErrFn func(to string) error // inject per-recipient errors
}

type SentEmail struct{ To, Subject, Body string }

func (m *MockSender) Send(_ context.Context, to, subject, body string) error {
	if m.ErrFn != nil {
		if err := m.ErrFn(to); err != nil {
			return err
		}
	}
	m.Sent = append(m.Sent, SentEmail{to, subject, body})
	return nil
}