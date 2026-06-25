package emaileventrepo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
	"github.com/muhammedfazall/Sendr/internal/core/ports"
)

// PostgresEmailEventRepository implements ports.EmailEventRepository.
type PostgresEmailEventRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresEmailEventRepository {
	return &PostgresEmailEventRepository{db: db}
}

// Store inserts a delivery event.
func (r *PostgresEmailEventRepository) Store(ctx context.Context, event *domain.EmailEvent) error {
	meta, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	const q = `
		INSERT INTO email_events (id, email, event_type, sg_event_id, sg_message_id, job_id, timestamp, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = r.db.Exec(ctx, q,
		event.ID, event.Email, event.EventType,
		nullOrString(event.SGEventID), nullOrString(event.SGMessageID),
		nullOrString(event.JobID), event.Timestamp, meta,
	)
	if err != nil {
		return fmt.Errorf("store email event: %w", err)
	}
	return nil
}

// GetStats returns aggregate event counts for a user's emails.
func (r *PostgresEmailEventRepository) GetStats(ctx context.Context, userID string) (*ports.EmailEventStats, error) {
	const q = `
		SELECT
			COALESCE(SUM((ee.event_type = 'delivered')::int), 0),
			COALESCE(SUM((ee.event_type = 'open')::int), 0),
			COALESCE(SUM((ee.event_type = 'click')::int), 0),
			COALESCE(SUM((ee.event_type = 'bounce')::int), 0),
			COALESCE(SUM((ee.event_type = 'spamreport')::int), 0)
		FROM email_events ee
		JOIN jobs j ON j.id = ee.job_id
		WHERE j.user_id = $1`

	var stats ports.EmailEventStats
	err := r.db.QueryRow(ctx, q, userID).Scan(
		&stats.Delivered,
		&stats.Opens,
		&stats.Clicks,
		&stats.Bounces,
		&stats.Spam,
	)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	// Count sent jobs
	const sentQ = `SELECT COUNT(*) FROM jobs WHERE user_id = $1 AND status = 'sent'`
	err = r.db.QueryRow(ctx, sentQ, userID).Scan(&stats.Sent)
	if err != nil {
		return nil, fmt.Errorf("get sent count: %w", err)
	}

	return &stats, nil
}

func nullOrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
