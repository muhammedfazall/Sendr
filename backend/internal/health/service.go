package health

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// dbPinger defines the database ping method needed by the health service.
type dbPinger interface {
	Ping(ctx context.Context) error
}

// redisPinger defines the Redis ping method needed by the health service.
type redisPinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// HealthStatus holds the status of each dependency.
type HealthStatus struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Redis  string `json:"redis"`
}

// Service contains the business logic for health checks.
type Service struct {
	db  dbPinger
	rdb redisPinger
}

// NewService creates a new health Service.
func NewService(db dbPinger, rdb redisPinger) *Service {
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

	overall := "ok"
	if dbStatus != "ok" || redisStatus != "ok" {
		overall = "degraded"
	}

	return HealthStatus{
		Status: overall,
		DB:     dbStatus,
		Redis:  redisStatus,
	}
}
