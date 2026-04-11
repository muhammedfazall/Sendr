package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all database operations for authentication.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new auth Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// UpsertUser inserts a new user or updates the existing one based on google_id.
// Returns the user's UUID.
func (r *Repository) UpsertUser(ctx context.Context, googleID, email, name string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (email, name, google_id, plan_id)
		 VALUES ($1, $2, $3, (SELECT id FROM plans WHERE name = 'free'))
		 ON CONFLICT (google_id) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id`,
		email, name, googleID,
	).Scan(&userID)
	return userID, err
}
