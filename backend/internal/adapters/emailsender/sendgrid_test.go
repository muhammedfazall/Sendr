package emailsender

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestIsRetryableWrapsRetryableError(t *testing.T) {
	inner := errors.New("inner error")
	rerr := &RetryableError{Err: inner}

	if !IsRetryable(rerr) {
		t.Fatal("IsRetryable should return true for RetryableError")
	}

	if IsRetryable(inner) {
		t.Fatal("IsRetryable should return false for plain error")
	}
}

func TestIsRetryableWithWrappedError(t *testing.T) {
	inner := &RetryableError{Err: errors.New("network timeout")}
	wrapped := fmt.Errorf("send failed: %w", inner)

	if !IsRetryable(wrapped) {
		t.Fatal("IsRetryable should unwrap and find RetryableError")
	}
}

func TestIsRetryableNilError(t *testing.T) {
	if IsRetryable(nil) {
		t.Fatal("IsRetryable(nil) should return false")
	}
}

func TestRetryableErrorImplementsError(t *testing.T) {
	rerr := &RetryableError{Err: errors.New("test error")}
	if rerr.Error() != "test error" {
		t.Fatalf("expected 'test error', got %q", rerr.Error())
	}
}

func TestNewSendGridConfig(t *testing.T) {
	sender := NewSendGrid("test-key", "from@test.com", "Test Sender")
	if sender.apiKey != "test-key" {
		t.Fatalf("expected apiKey 'test-key', got %q", sender.apiKey)
	}
	if sender.fromEmail != "from@test.com" {
		t.Fatalf("expected fromEmail 'from@test.com', got %q", sender.fromEmail)
	}
	if sender.fromName != "Test Sender" {
		t.Fatalf("expected fromName 'Test Sender', got %q", sender.fromName)
	}
	if sender.client.Timeout != 30*time.Second {
		t.Fatalf("expected 30s timeout, got %s", sender.client.Timeout)
	}
}

func TestNewSendGridHTTPClient(t *testing.T) {
	sender := NewSendGrid("k", "f@t.com", "F")
	if sender.client.Timeout != 30*time.Second {
		t.Fatal("expected 30s HTTP client timeout")
	}
	if sender.client.Transport != nil {
		t.Fatal("expected nil Transport (uses http.DefaultTransport)")
	}
}

func TestMockSender(t *testing.T) {
	sender := &MockSender{}

	err := sender.Send(context.Background(), "a@test.com", "Subject A", "Body A")
	if err != nil {
		t.Fatalf("mock Send: %v", err)
	}
	err = sender.Send(context.Background(), "b@test.com", "Subject B", "Body B")
	if err != nil {
		t.Fatalf("mock Send: %v", err)
	}

	if len(sender.Sent) != 2 {
		t.Fatalf("expected 2 sent emails, got %d", len(sender.Sent))
	}
	if sender.Sent[0].To != "a@test.com" || sender.Sent[0].Subject != "Subject A" {
		t.Fatalf("unexpected first email: %+v", sender.Sent[0])
	}
	if sender.Sent[1].To != "b@test.com" || sender.Sent[1].Body != "Body B" {
		t.Fatalf("unexpected second email: %+v", sender.Sent[1])
	}
}

func TestMockSenderWithErrorFn(t *testing.T) {
	sender := &MockSender{
		ErrFn: func(to string) error {
			if to == "fail@test.com" {
				return errors.New("injected error")
			}
			return nil
		},
	}

	err := sender.Send(context.Background(), "ok@test.com", "S", "B")
	if err != nil {
		t.Fatalf("expected no error for ok@test.com, got %v", err)
	}

	err = sender.Send(context.Background(), "fail@test.com", "S", "B")
	if err == nil {
		t.Fatal("expected error for fail@test.com, got nil")
	}

	if len(sender.Sent) != 1 {
		t.Fatalf("expected 1 successful send, got %d", len(sender.Sent))
	}
}

func TestMockSenderCleanState(t *testing.T) {
	sender := &MockSender{}
	if len(sender.Sent) != 0 {
		t.Fatal("new mock should have empty Sent slice")
	}
}

func TestMockSenderInterface(t *testing.T) {
	var s Sender = &MockSender{}
	if s == nil {
		t.Fatal("MockSender should implement Sender interface")
	}
}
