package webhookhandler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
	"github.com/muhammedfazall/Sendr/internal/middleware"
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

// Handle receives SendGrid event notifications.
func (h *SendGridHandler) Handle(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		middleware.Logger(c).Error("sendgrid webhook: read body", "err", err)
		c.Status(http.StatusBadRequest)
		return
	}

	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		middleware.Logger(c).Error("sendgrid webhook: unmarshal error", "err", err)
		c.Status(http.StatusBadRequest)
		return
	}

	for _, ev := range raw {
		h.storeEvent(c, ev)
	}

	c.Status(http.StatusOK)
}

// storeEvent extracts, links to a job, and persists a single SendGrid event.
func (h *SendGridHandler) storeEvent(c *gin.Context, raw map[string]any) {
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
		job, err := h.jobs.FindByProviderMessageID(c.Request.Context(), sgMessageID)
		if err == nil && job != nil {
			ev.JobID = job.ID
		}
	}

	if err := h.events.Store(c.Request.Context(), ev); err != nil {
		middleware.Logger(c).Error("sendgrid webhook: store event", "sg_event_id", sgEventID, "err", err)
	}
}
