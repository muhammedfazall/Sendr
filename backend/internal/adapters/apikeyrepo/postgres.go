package apikeyrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// PostgresAPIKeyRepository implements ports.APIKeyRepository.
type PostgresAPIKeyRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresAPIKeyRepository {
	return &PostgresAPIKeyRepository{db: db}
}

func (r *PostgresAPIKeyRepository) Create(ctx context.Context, userID, name, prefix, hashedKey string) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.db.QueryRow(ctx,
		`INSERT INTO api_keys (user_id, name, prefix, hashed_key)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, name, prefix, hashed_key, revoked, created_at`,
		userID, name, prefix, hashedKey,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.Hashed, &k.Revoked, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &k, nil
}

func (r *PostgresAPIKeyRepository) ListByUser(ctx context.Context, userID string) ([]domain.APIKey, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, name, prefix, hashed_key, revoked, created_at
		 FROM api_keys
		 WHERE user_id = $1 AND revoked = false
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.Hashed, &k.Revoked, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *PostgresAPIKeyRepository) FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, name, prefix, hashed_key, revoked, created_at
		 FROM api_keys WHERE prefix = $1`,
		prefix,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.Hashed, &k.Revoked, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("find by prefix: %w", err)
	}
	return &k, nil
}

func (r *PostgresAPIKeyRepository) Revoke(ctx context.Context, keyID, userID string) error {
	var id string
	err := r.db.QueryRow(ctx,
		`UPDATE api_keys SET revoked = true
		 WHERE id = $1 AND user_id = $2 AND revoked = false
		 RETURNING id`,
		keyID, userID,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	return nil
}