package health

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/pkg/response"
	"github.com/redis/go-redis/v9"

	"net/http"
)

func Check(db *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := db.Ping(context.Background()); err != nil {
			dbStatus = "error"
		}

		redisStatus := "ok"
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			redisStatus = "error"
		}

		status := http.StatusOK
		if dbStatus != "ok" || redisStatus != "ok" {
			status = http.StatusServiceUnavailable
		}

		response.JSON(w, status, map[string]string{
			"status": "ok",
			"db":     dbStatus,
			"redis":  redisStatus,
		})
	}
}
