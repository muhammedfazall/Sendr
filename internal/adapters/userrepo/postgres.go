package userrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

// PostgresUserRepository implements ports.UserRepository.
type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Upsert inserts a new user or updates their email on google_id conflict.
// Assigns the 'free' plan on first login.
func (r *PostgresUserRepository) Upsert(ctx context.Context, googleID, email, name string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (email, name, google_id, plan_id)
		 VALUES ($1, $2, $3, (SELECT id FROM plans WHERE name = 'free'))
		 ON CONFLICT (google_id) DO UPDATE SET email = EXCLUDED.email
		 RETURNING id, email, name, google_id, plan_id, created_at`,
		email, name, googleID,
	).Scan(&u.ID, &u.Email, &u.Name, &u.GoogleID, &u.PlanID, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}
	return &u, nil
}

// FindByID returns a user by their UUID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, google_id, plan_id, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.GoogleID, &u.PlanID, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return &u, nil
}

// FindWithPlan returns the user and their associated plan in one query.
func (r *PostgresUserRepository) FindWithPlan(ctx context.Context, id string) (*domain.User, *domain.Plan, error) {
	var u domain.User
	var p domain.Plan
	err := r.db.QueryRow(ctx,
		`SELECT u.id, u.email, u.name, u.google_id, u.plan_id, u.created_at,
		        p.id, p.name, p.daily_limit, p.created_at
		 FROM users u
		 JOIN plans p ON p.id = u.plan_id
		 WHERE u.id = $1`,
		id,
	).Scan(
		&u.ID, &u.Email, &u.Name, &u.GoogleID, &u.PlanID, &u.CreatedAt,
		&p.ID, &p.Name, &p.DailyLimit, &p.CreatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("find user with plan: %w", err)
	}
	return &u, &p, nil
}