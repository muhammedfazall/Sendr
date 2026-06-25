package webhookhandler

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
)

// jobLookupper finds a job by its provider-side message ID.
type jobLookupper interface {
	FindByProviderMessageID(ctx context.Context, providerMessageID string) (*domain.Job, error)
}

// SendGridHandler receives event notifications from SendGrid.
type SendGridHandler struct {
	events ports.EmailEventRepository
	jobs   jobLookupper
}

func NewSendGridHandler(events ports.EmailEventRepository, jobs jobLookupper) *SendGridHandler {
	return &SendGridHandler{events: events, jobs: jobs}
}

// Handle processes a SendGrid event POST.
// SendGrid sends a JSON array of event objects.
func (h *SendGridHandler) Handle() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("sendgrid webhook: read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var raw []map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			log.Printf("sendgrid webhook: unmarshal: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		for _, ev := range raw {
			h.storeEvent(r, ev)
		}

		w.WriteHeader(http.StatusOK)
	}
}

// storeEvent extracts, links to a job, and persists a single SendGrid event.
func (h *SendGridHandler) storeEvent(r *http.Request, raw map[string]any) {
	email, _ := raw["email"].(string)
	eventType, _ := raw["event"].(string)
	sgEventID, _ := raw["sg_event_id"].(string)
	sgMessageID, _ := raw["sg_message_id"].(string)

	ts, _ := raw["timestamp"].(float64)
	timestamp := time.Unix(int64(ts), 0)

	meta, _ := json.Marshal(raw)

	ev := &domain.EmailEvent{
		ID:          uuid.NewString(),
		Email:       email,
		EventType:   eventType,
		SGEventID:   sgEventID,
		SGMessageID: sgMessageID,
		Timestamp:   timestamp,
		Metadata:    meta,
	}

	// Try to link this event to a job via SendGrid's message ID.
	if sgMessageID != "" && h.jobs != nil {
		job, err := h.jobs.FindByProviderMessageID(r.Context(), sgMessageID)
		if err == nil && job != nil {
			ev.JobID = job.ID
		}
	}

	if err := h.events.Store(r.Context(), ev); err != nil {
		log.Printf("sendgrid webhook: store event %s: %v", sgEventID, err)
	}
}
