package planrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

type PostgresPlanRepository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *PostgresPlanRepository {
	return &PostgresPlanRepository{db: db}
}

func (r *PostgresPlanRepository) FindByName(ctx context.Context, name string) (*domain.Plan, error) {
	var p domain.Plan
	err := r.db.QueryRow(ctx,
		`SELECT id, name, daily_limit, max_api_keys, rate_wait_secs, price_paise, created_at
		 FROM plans WHERE name = $1`, name,
	).Scan(&p.ID, &p.Name, &p.DailyLimit, &p.MaxAPIKeys, &p.RateWaitSecs, &p.PricePaise, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("find plan: %w", err)
	}
	return &p, nil
}

func (r *PostgresPlanRepository) ListAll(ctx context.Context) ([]domain.Plan, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, daily_limit, max_api_keys, rate_wait_secs, price_paise, created_at
		 FROM plans ORDER BY price_paise ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.DailyLimit, &p.MaxAPIKeys,
			&p.RateWaitSecs, &p.PricePaise, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, nil
}