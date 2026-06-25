package worker

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/muhammedfazall/Sendr/internal/adapters/emailsender"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// eventBus defines the publish side of the event bus the worker needs.
type eventBus interface {
	Publish(event domain.Event)
}

// jobRepository defines the repo methods the worker needs.
type jobRepository interface {
	MarkDone(ctx context.Context, jobID string) error
	MarkFailed(ctx context.Context, jobID string, backoff time.Duration) error
	MoveToDLQ(ctx context.Context, job domain.Job, errMsg string) error
	ReclaimZombies(ctx context.Context) (int64, error)
	ClaimBatch(ctx context.Context, batchSize int) ([]domain.Job, error)
	SetProviderMessageID(ctx context.Context, jobID, providerMessageID string) error
}

// backoffSchedule maps retry attempt (0-indexed) to how long to wait before
// the next try. Capped at the last entry for attempts beyond len-1.
var backoffSchedule = []time.Duration{
	10 * time.Second,
	60 * time.Second,
	300 * time.Second,
}

// Worker polls the job queue and processes jobs concurrently.
type Worker struct {
	repo              jobRepository
	sender            emailsender.Sender
	bus               eventBus
	log               *slog.Logger
	unsubscribeBase   string
	unsubscribeSecret string
}

func New(repo jobRepository, sender emailsender.Sender, bus eventBus, log *slog.Logger, unsubscribeBase, unsubscribeSecret string) *Worker {
	return &Worker{
		repo:              repo,
		sender:            sender,
		bus:               bus,
		log:               log,
		unsubscribeBase:   unsubscribeBase,
		unsubscribeSecret: unsubscribeSecret,
	}
}

// Run starts the poll loop and blocks until ctx is cancelled.
// Intended to be called in a goroutine from main.
func (w *Worker) Run(ctx context.Context) {
	pollTick := time.NewTicker(2 * time.Second)
	zombieTick := time.NewTicker(60 * time.Second)
	// Semaphore: caps concurrent in-flight jobs at 10.
	sem := make(chan struct{}, 10)

	defer pollTick.Stop()
	defer zombieTick.Stop()

	w.log.Info("worker started")

	for {
		select {
		case <-ctx.Done():
			w.log.Info("worker shutting down — draining in-flight jobs")
			// Fill the semaphore to capacity: blocks until all 10 slots are returned,
			// meaning every running goroutine has finished.
			for i := 0; i < cap(sem); i++ {
				sem <- struct{}{}
			}
			w.log.Info("worker stopped cleanly")
			return

		case <-zombieTick.C:
			n, err := w.repo.ReclaimZombies(ctx)
			if err != nil {
				w.log.Error("zombie reclaim failed", "err", err)
				continue
			}
			if n > 0 {
				w.log.Info("reclaimed zombie jobs", "count", n)
			}

		case <-pollTick.C:
			jobs, err := w.repo.ClaimBatch(ctx, 10)
			if err != nil {
				w.log.Error("claim batch failed", "err", err)
				continue
			}
			for _, j := range jobs {
				sem <- struct{}{} // acquire slot (blocks if 10 already running)
				go func(job domain.Job) {
					defer func() { <-sem }() // release slot when done
					w.processJob(ctx, job)
				}(j)
			}
		}
	}
}

// processJob sends the email and updates the job status.
// It is the only place that decides retry vs DLQ.
func (w *Worker) processJob(ctx context.Context, j domain.Job) {
	log := w.log.With("job_id", j.ID, "to", j.Payload.To)

	w.attachUnsubscribeHeader(&j.Payload)

	result, err := w.sender.Send(ctx, &j.Payload)
	if err == nil {
		if result != nil && result.ProviderMessageID != "" {
			if setErr := w.repo.SetProviderMessageID(ctx, j.ID, result.ProviderMessageID); setErr != nil {
				log.Error("set provider message id failed", "err", setErr)
			}
		}
		w.publish(domain.EventEmailSent, j)
		if markErr := w.repo.MarkDone(ctx, j.ID); markErr != nil {
			log.Error("mark done failed", "err", markErr)
		} else {
			log.Info("email sent", "provider_message_id", result.ProviderMessageID)
		}
		return
	}

	log.Warn("send failed", "err", err, "retries", j.Retries)

	w.publish(domain.EventEmailFailed, j)

	// Non-retryable error OR retries exhausted → straight to DLQ
	if !emailsender.IsRetryable(err) || j.Retries >= j.MaxRetries-1 {
		if dlqErr := w.repo.MoveToDLQ(ctx, j, err.Error()); dlqErr != nil {
			log.Error("move to DLQ failed", "err", dlqErr)
		} else {
			log.Error("job moved to DLQ", "reason", err.Error())
		}
		return
	}

	// Retryable — schedule with backoff
	idx := j.Retries
	if idx >= len(backoffSchedule) {
		idx = len(backoffSchedule) - 1
	}
	backoff := backoffSchedule[idx]

	if retryErr := w.repo.MarkFailed(ctx, j.ID, backoff); retryErr != nil {
		log.Error("mark failed error", "err", retryErr)
	} else {
		log.Info("scheduled retry", "backoff", backoff, "attempt", j.Retries+1)
	}
}

// publish sends an event to the bus if one is configured.
func (w *Worker) publish(typ domain.EventType, j domain.Job) {
	if w.bus == nil {
		return
	}
	w.bus.Publish(domain.NewEvent(typ, j))
}

// attachUnsubscribeHeader adds a List-Unsubscribe header to every recipient
// so mail clients show an unsubscribe button and providers don't flag as spam.
func (w *Worker) attachUnsubscribeHeader(payload *domain.EmailPayload) {
	if w.unsubscribeSecret == "" || w.unsubscribeBase == "" {
		return
	}
	if payload.Headers == nil {
		payload.Headers = make(map[string]string)
	}
	var links string
	for _, to := range payload.To {
		tok := unsubscribeToken(to, w.unsubscribeSecret)
		links += fmt.Sprintf("<%s/unsubscribe?email=%s&token=%s>,", w.unsubscribeBase, to, tok)
	}
	if len(links) > 0 {
		links = links[:len(links)-1] // trim trailing comma
	}
	payload.Headers["List-Unsubscribe"] = links
}

func unsubscribeToken(email, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(email))
	return hex.EncodeToString(mac.Sum(nil))
}