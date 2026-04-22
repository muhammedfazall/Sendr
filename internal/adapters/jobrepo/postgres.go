package jobrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// PostgresJobRepository implements ports.JobRepository.
type PostgresJobRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresJobRepository {
	return &PostgresJobRepository{db: db}
}

// Enqueue inserts a new job with status=pending and returns it.
func (r *PostgresJobRepository) Enqueue(ctx context.Context, userID, apiKeyID string, payload domain.EmailPayload) (*domain.Job, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	const q = `
		INSERT INTO jobs (user_id, api_key_id, payload, max_retries)
		VALUES ($1, $2, $3, 3)
		RETURNING id, status, retries, max_retries, run_at, created_at, updated_at`

	var j domain.Job
	j.UserID = userID
	j.APIKeyID = apiKeyID
	j.Payload = payload

	err = r.db.QueryRow(ctx, q, userID, apiKeyID, data).
		Scan(&j.ID, &j.Status, &j.Retries, &j.MaxRetries, &j.RunAt, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("enqueue job: %w", err)
	}

	return &j, nil
}

// ClaimBatch atomically claims up to batchSize pending jobs.
// Uses SELECT FOR UPDATE SKIP LOCKED — safe to call from multiple workers concurrently.
func (r *PostgresJobRepository) ClaimBatch(ctx context.Context, batchSize int) ([]domain.Job, error) {
	const q = `
		UPDATE jobs SET
			status       = 'processing',
			locked_until = now() + interval '30 seconds',
			updated_at   = now()
		WHERE id IN (
			SELECT id FROM jobs
			WHERE  status = 'pending'
			AND    run_at <= now()
			AND    (locked_until IS NULL OR locked_until < now())
			ORDER  BY run_at ASC
			LIMIT  $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, user_id, api_key_id, payload, retries, max_retries`

	rows, err := r.db.Query(ctx, q, batchSize)
	if err != nil {
		return nil, fmt.Errorf("claim batch: %w", err)
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		var raw []byte
		if err := rows.Scan(&j.ID, &j.UserID, &j.APIKeyID, &raw, &j.Retries, &j.MaxRetries); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		if err := json.Unmarshal(raw, &j.Payload); err != nil {
			return nil, fmt.Errorf("unmarshal payload: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// MarkDone sets the job status to sent.
func (r *PostgresJobRepository) MarkDone(ctx context.Context, jobID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE jobs SET status='sent', locked_until=NULL, updated_at=now() WHERE id=$1`,
		jobID)
	return err
}

// MarkFailed resets the job to pending with an exponential backoff run_at,
// incrementing the retry counter. The caller decides the backoff duration.
func (r *PostgresJobRepository) MarkFailed(ctx context.Context, jobID string, backoff time.Duration) error {
	_, err := r.db.Exec(ctx,
		`UPDATE jobs SET
			status       = 'pending',
			retries      = retries + 1,
			locked_until = NULL,
			run_at       = now() + $2,
			updated_at   = now()
		WHERE id = $1`,
		jobID, backoff)
	return err
}

// MoveToDLQ inserts the job into the dlq table and marks it as failed.
// Runs in a transaction so both writes succeed or neither does.
func (r *PostgresJobRepository) MoveToDLQ(ctx context.Context, job domain.Job, errMsg string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO dlq (job_id, payload, error_message)
		 SELECT id, payload, $2 FROM jobs WHERE id = $1`,
		job.ID, errMsg)
	if err != nil {
		return fmt.Errorf("insert dlq: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE jobs SET status='failed', locked_until=NULL, updated_at=now() WHERE id=$1`,
		job.ID)
	if err != nil {
		return fmt.Errorf("mark job failed: %w", err)
	}

	return tx.Commit(ctx)
}

// ReclaimZombies resets jobs stuck in 'processing' with an expired lock.
// Call this on a ticker (every ~60s) to recover from crashed workers.
func (r *PostgresJobRepository) ReclaimZombies(ctx context.Context) (int64, error) {
	res, err := r.db.Exec(ctx,
		`UPDATE jobs SET
			status       = 'pending',
			locked_until = NULL,
			updated_at   = now()
		WHERE status       = 'processing'
		AND   locked_until < now()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected(), nil
}

// GetByID returns a single job by ID for status polling.
func (r *PostgresJobRepository) GetByID(ctx context.Context, jobID string) (*domain.Job, error) {
	var j domain.Job
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, api_key_id, status, retries, max_retries, run_at, created_at, updated_at
		 FROM jobs WHERE id = $1`,
		jobID,
	).Scan(&j.ID, &j.UserID, &j.APIKeyID, &j.Status, &j.Retries, &j.MaxRetries,
		&j.RunAt, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &j, nil
}