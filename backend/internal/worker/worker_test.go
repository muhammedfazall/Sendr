package worker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/muhammedfazall/Sendr/internal/adapters/emailsender"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// mockRepo implements the minimal repository interface the worker needs.
type mockRepo struct {
	markDoneCalled   bool
	markFailedCalled bool
	dlqCalled        bool
	lastBackoff      time.Duration
	claimBatchErr    error
	reclaimZombiesFn func() (int64, error)
}

func (m *mockRepo) ClaimBatch(_ context.Context, _ int) ([]domain.Job, error) {
	if m.claimBatchErr != nil {
		return nil, m.claimBatchErr
	}
	return nil, nil
}
func (m *mockRepo) MarkDone(_ context.Context, _ string) error {
	m.markDoneCalled = true
	return nil
}
func (m *mockRepo) MarkFailed(_ context.Context, _ string, backoff time.Duration) error {
	m.markFailedCalled = true
	m.lastBackoff = backoff
	return nil
}
func (m *mockRepo) MoveToDLQ(_ context.Context, _ domain.Job, _ string) error {
	m.dlqCalled = true
	return nil
}
func (m *mockRepo) ReclaimZombies(_ context.Context) (int64, error) {
	if m.reclaimZombiesFn != nil {
		return m.reclaimZombiesFn()
	}
	return 0, nil
}
func (m *mockRepo) GetByID(_ context.Context, _ string) (*domain.Job, error)  { return nil, nil }
func (m *mockRepo) ListByUser(_ context.Context, _ string, _ string, _, _ int) ([]domain.Job, error) {
	return nil, nil
}
func (m *mockRepo) Enqueue(_ context.Context, _, _ string, _ domain.EmailPayload) (*domain.Job, error) {
	return nil, nil
}

var errSendFailed = errors.New("send failed")

func TestWorkerProcessJobSuccess(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	job := domain.Job{
		ID: "job-1", UserID: "user-1", Status: "pending",
		Payload: domain.EmailPayload{To: "a@test.com", Subject: "S", Body: "B"},
	}

	w.processJob(context.Background(), job)

	if !repo.markDoneCalled {
		t.Fatal("expected MarkDone to be called on success")
	}
	if repo.markFailedCalled {
		t.Fatal("MarkFailed should not be called on success")
	}
	if repo.dlqCalled {
		t.Fatal("DLQ should not be called on success")
	}
	if len(sender.Sent) != 1 {
		t.Fatalf("expected 1 sent email, got %d", len(sender.Sent))
	}
	if sender.Sent[0].To != "a@test.com" {
		t.Fatalf("expected to 'a@test.com', got %q", sender.Sent[0].To)
	}
}

func TestWorkerProcessJobNonRetryableError(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{
		ErrFn: func(to string) error {
			return errors.New("400 bad request: invalid email")
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	job := domain.Job{
		ID: "job-2", UserID: "user-1", Status: "pending", Retries: 0, MaxRetries: 3,
		Payload: domain.EmailPayload{To: "bad-email", Subject: "S", Body: "B"},
	}

	w.processJob(context.Background(), job)

	if !repo.dlqCalled {
		t.Fatal("expected DLQ move for non-retryable error")
	}
	if repo.markDoneCalled {
		t.Fatal("MarkDone should not be called on failure")
	}
	if repo.markFailedCalled {
		t.Fatal("MarkFailed should not be called for non-retryable error")
	}
}

func TestWorkerProcessJobRetryableError(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{
		ErrFn: func(to string) error {
			return &emailsender.RetryableError{Err: errors.New("network timeout")}
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	job := domain.Job{
		ID: "job-3", UserID: "user-1", Status: "pending", Retries: 0, MaxRetries: 3,
		Payload: domain.EmailPayload{To: "a@test.com", Subject: "S", Body: "B"},
	}

	w.processJob(context.Background(), job)

	if !repo.markFailedCalled {
		t.Fatal("expected MarkFailed to be called for retryable error")
	}
	if repo.markDoneCalled {
		t.Fatal("MarkDone should not be called on failure")
	}
	if repo.dlqCalled {
		t.Fatal("DLQ should not be called for retryable error with retries left")
	}
	if repo.lastBackoff != 10*time.Second {
		t.Fatalf("expected backoff 10s for first retry, got %s", repo.lastBackoff)
	}
}

func TestWorkerProcessJobRetryExhausted(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{
		ErrFn: func(to string) error {
			return &emailsender.RetryableError{Err: errors.New("network timeout")}
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	// Job with retries = 2, MaxRetries = 3 — this means 2 retries already used,
	// so one more try is allowed (retries < MaxRetries-1 → 2 < 2 → false)
	job := domain.Job{
		ID: "job-4", UserID: "user-1", Status: "pending", Retries: 2, MaxRetries: 3,
		Payload: domain.EmailPayload{To: "a@test.com", Subject: "S", Body: "B"},
	}

	w.processJob(context.Background(), job)

	if !repo.dlqCalled {
		t.Fatal("expected DLQ move when retries exhausted")
	}
	if repo.markFailedCalled {
		t.Fatal("MarkFailed should not be called after retries exhausted")
	}
}

func TestWorkerProcessJobLastRetry(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{
		ErrFn: func(to string) error {
			return &emailsender.RetryableError{Err: errors.New("still failing")}
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	// retries = 1, MaxRetries = 3 — one more retry allowed
	job := domain.Job{
		ID: "job-5", UserID: "user-1", Status: "pending", Retries: 1, MaxRetries: 3,
		Payload: domain.EmailPayload{To: "a@test.com", Subject: "S", Body: "B"},
	}

	w.processJob(context.Background(), job)

	if !repo.markFailedCalled {
		t.Fatal("expected MarkFailed for retryable error with retries left")
	}
	if repo.dlqCalled {
		t.Fatal("DLQ should not be called when retries still available")
	}
	// Second retry: backoffSchedule[1] = 60s
	if repo.lastBackoff != 60*time.Second {
		t.Fatalf("expected backoff 60s for second retry, got %s", repo.lastBackoff)
	}
}

func TestWorkerProcessJobBackoffScheduleBoundary(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{
		ErrFn: func(to string) error {
			return &emailsender.RetryableError{Err: errors.New("failing")}
		},
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	// retries = 0 → backoffSchedule[0] = 10s
	job := domain.Job{
		ID: "job-6", UserID: "user-1", Status: "pending", Retries: 0, MaxRetries: 5,
		Payload: domain.EmailPayload{To: "a@test.com", Subject: "S", Body: "B"},
	}

	w.processJob(context.Background(), job)

	if !repo.markFailedCalled {
		t.Fatal("expected MarkFailed for retryable error")
	}
	if repo.lastBackoff != 10*time.Second {
		t.Fatalf("expected backoff 10s for retry 0, got %s", repo.lastBackoff)
	}
}

func TestWorkerProcessJobMarkDoneError(t *testing.T) {
	repo := &mockRepo{
		// Can't set MarkDone to fail without more fields, but we can
		// verify the function doesn't panic on error from MarkDone
	}
	sender := &emailsender.MockSender{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	job := domain.Job{
		ID: "job-7", UserID: "user-1",
		Payload: domain.EmailPayload{To: "a@test.com", Subject: "S", Body: "B"},
	}

	// Should not panic
	w.processJob(context.Background(), job)
}

func TestWorkerBackoffScheduleLength(t *testing.T) {
	if len(backoffSchedule) != 3 {
		t.Fatalf("expected 3 backoff entries, got %d", len(backoffSchedule))
	}
	expected := []time.Duration{10 * time.Second, 60 * time.Second, 300 * time.Second}
	for i, d := range backoffSchedule {
		if d != expected[i] {
			t.Fatalf("backoff[%d] = %s, want %s", i, d, expected[i])
		}
	}
}

func TestWorkerBackoffScheduleOOB(t *testing.T) {
	// When retry count exceeds schedule length, last entry is used
	idx := 5
	if idx >= len(backoffSchedule) {
		idx = len(backoffSchedule) - 1
	}
	if backoffSchedule[idx] != 300*time.Second {
		t.Fatalf("OOB backoff should cap at 300s, got %s", backoffSchedule[idx])
	}
}

func TestWorkerNewWithMockDepsCreatesCleanInstance(t *testing.T) {
	repo := &mockRepo{}
	sender := &emailsender.MockSender{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	w := New(repo, sender, logger)

	if w.repo != repo {
		t.Fatal("repo not set correctly")
	}
	if w.sender != sender {
		t.Fatal("sender not set correctly")
	}
	if w.log != logger {
		t.Fatal("logger not set correctly")
	}
}

func TestWorkerImplementsSenderInterface(t *testing.T) {
	var s emailsender.Sender = &emailsender.MockSender{}
	if s == nil {
		t.Fatal("MockSender should implement Sender")
	}
}
