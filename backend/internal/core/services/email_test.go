package services

import (
	"context"
	"errors"
	"testing"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

func TestEmailServiceSendSuccess(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("free")
	fullKey, _ := mock.keySvc.addKey(user.ID)

	job, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{
		To:      "test@example.com",
		Subject: "Hello",
		Body:    "World",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if job == nil {
		t.Fatal("expected non-nil job")
	}
	if job.Payload.To != "test@example.com" {
		t.Fatalf("expected To 'test@example.com', got %q", job.Payload.To)
	}
}

func TestEmailServiceSendInvalidKey(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	_, err := svc.Send(context.Background(), "mk_live_bad.key", domain.EmailPayload{
		To: "test@example.com", Subject: "Hi", Body: "Body",
	})
	if err == nil {
		t.Fatal("expected error for invalid key, got nil")
	}
}

func TestEmailServiceSendRateLimited(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("free")
	fullKey, _ := mock.keySvc.addKey(user.ID)

	mock.limiter.allow = false

	_, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{
		To: "test@example.com", Subject: "Hi", Body: "Body",
	})
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}
}

func TestEmailServiceSendUnlimitedPlan(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("max") // unlimited plan
	fullKey, _ := mock.keySvc.addKey(user.ID)

	mock.limiter.allow = false // would block if checked

	job, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{
		To: "test@example.com", Subject: "Unlimited", Body: "Test",
	})
	if err != nil {
		t.Fatalf("Send for unlimited plan returned error: %v", err)
	}
	if job == nil {
		t.Fatal("expected non-nil job")
	}
}

func TestEmailServiceSendUserNotFound(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	// Create a key for a user that doesn't have a plan set up
	user := &domain.User{ID: "user-no-plan", Email: "noplan@test.com", Name: "No Plan"}
	mock.users.users[user.ID] = user
	fullKey, _ := mock.keySvc.addKey(user.ID)

	_, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{
		To: "test@example.com", Subject: "Hi", Body: "Body",
	})
	if err == nil {
		t.Fatal("expected error for user with no plan, got nil")
	}
}

func TestEmailServiceSendEnqueueError(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("free")
	fullKey, _ := mock.keySvc.addKey(user.ID)

	mock.jobs.enqueueErr = errors.New("queue full")

	_, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{
		To: "test@example.com", Subject: "Hi", Body: "Body",
	})
	if err == nil {
		t.Fatal("expected error from Enqueue, got nil")
	}
}

func TestEmailServiceEmptyPayload(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("free")
	fullKey, _ := mock.keySvc.addKey(user.ID)

	job, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{})
	if err != nil {
		t.Fatalf("Send with empty payload returned error: %v", err)
	}
	if job == nil {
		t.Fatal("expected non-nil job even with empty payload")
	}
}

func TestEmailServiceMultipleSends(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("pro")
	fullKey, _ := mock.keySvc.addKey(user.ID)

	payloads := []domain.EmailPayload{
		{To: "a@test.com", Subject: "A", Body: "A body"},
		{To: "b@test.com", Subject: "B", Body: "B body"},
		{To: "c@test.com", Subject: "C", Body: "C body"},
	}

	for i, p := range payloads {
		job, err := svc.Send(context.Background(), fullKey, p)
		if err != nil {
			t.Fatalf("send %d returned error: %v", i, err)
		}
		if job.Payload.To != p.To {
			t.Fatalf("send %d: expected To %q, got %q", i, p.To, job.Payload.To)
		}
	}
}

func TestEmailServiceRateLimitRejected(t *testing.T) {
	mock := newMockDeps()
	svc := NewEmailService(mock.keySvc, mock.jobs, mock.users, mock.limiter)

	user := mock.addUserWithPlan("free")
	fullKey, _ := mock.keySvc.addKey(user.ID)

	mock.limiter.allow = false

	_, err := svc.Send(context.Background(), fullKey, domain.EmailPayload{
		To: "test@example.com", Subject: "Hi", Body: "Body",
	})
	if err == nil {
		t.Fatal("expected error when rate limit exceeded, got nil")
	}
}
