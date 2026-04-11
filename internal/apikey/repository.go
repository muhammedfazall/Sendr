package apikey

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles all database operations for API keys.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new apikey Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create inserts a new API key record and returns its generated ID.
func (r *Repository) Create(ctx context.Context, userID, name, prefix, hashedKey string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, name, prefix, hashed_key)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		userID, name, prefix, hashedKey,
	).Scan(&id)
	return id, err
}

// ListByUser returns all non-revoked API keys for the given user.
func (r *Repository) ListByUser(ctx context.Context, userID string) ([]APIKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, prefix, created_at
		 FROM api_keys
		 WHERE user_id = $1 AND revoked = false
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.CreatedAt); err != nil {
			continue
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// Revoke soft-deletes an API key by setting revoked = true.
// Returns an error if the key doesn't exist or doesn't belong to the user.
func (r *Repository) Revoke(ctx context.Context, keyID, userID string) error {
	var revokedID string
	return r.db.QueryRow(ctx,
		`UPDATE api_keys SET revoked = true
		 WHERE id = $1 AND user_id = $2 AND revoked = false
		 RETURNING id`,
		keyID, userID,
	).Scan(&revokedID)
}
