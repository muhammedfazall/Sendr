package unsubrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository stores unsubscriptions.
type PostgresRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Unsubscribe(ctx context.Context, email string) error {
	const q = `INSERT INTO unsubscriptions (email) VALUES ($1) ON CONFLICT (email) DO NOTHING`
	_, err := r.db.Exec(ctx, q, email)
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}
	return nil
}

func (r *PostgresRepository) IsUnsubscribed(ctx context.Context, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM unsubscriptions WHERE email = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, q, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check unsubscribed: %w", err)
	}
	return exists, nil
}
