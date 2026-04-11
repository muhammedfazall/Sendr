package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthStatus holds the status of each dependency.
type HealthStatus struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Redis  string `json:"redis"`
}

// Service contains the business logic for health checks.
type Service struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

// NewService creates a new health Service.
func NewService(db *pgxpool.Pool, rdb *redis.Client) *Service {
	return &Service{db: db, rdb: rdb}
}

// Check pings the database and Redis, returning the status of each.
func (s *Service) Check(ctx context.Context) HealthStatus {
	dbStatus := "ok"
	if err := s.db.Ping(ctx); err != nil {
		dbStatus = "error"
	}

	redisStatus := "ok"
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		redisStatus = "error"
	}

	return HealthStatus{
		Status: "ok",
		DB:     dbStatus,
		Redis:  redisStatus,
	}
}
