package job

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Insert(ctx context.Context, j *Job) error {
	query := `
	INSERT INTO jobs (user_id, api_key_id, payload, status, max_retries)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at
	`

	return r.db.QueryRow(ctx, query,
		j.UserID,
		j.APIKeyID,
		j.Payload,
		j.Status,
		j.MaxRetries,
	).Scan(&j.ID, &j.CreatedAt)
}
