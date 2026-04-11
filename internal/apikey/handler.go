package apikey

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/muhammedfazall/Sendr/internal/middleware"
	"github.com/muhammedfazall/Sendr/pkg/hash"
)

// POST /apikeys — create a new API key
func Create(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user_id from JWT claims
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims["user_id"].(string)

		// Parse request body — just needs a name
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}

		// Generate the key
		key, err := hash.GenerateAPIKey()
		if err != nil {
			http.Error(w, "failed to generate key", http.StatusInternalServerError)
			return
		}

		// Store prefix + hash in DB (never the full key)
		var id string
		err = db.QueryRow(context.Background(),
			`INSERT INTO api_keys (user_id, name, prefix, hashed_key)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			userID, body.Name, key.Prefix, key.Hashed,
		).Scan(&id)
		if err != nil {
			log.Printf("failed to save key: %v", err)
			http.Error(w, "failed to save key", http.StatusInternalServerError)
			return
		}

		// Return the FULL key once — never shown again
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":      id,
			"name":    body.Name,
			"prefix":  key.Prefix,
			"api_key": key.Full,
		})
	}
}

// GET /apikeys — list all keys for the user
func List(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims["user_id"].(string)

		rows, err := db.Query(context.Background(),
			`SELECT id, name, prefix, created_at
			 FROM api_keys
			 WHERE user_id = $1 AND revoked = false
			 ORDER BY created_at DESC`,
			userID,
		)
		if err != nil {
			http.Error(w, "failed to fetch keys", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type keyRow struct {
			ID        string    `json:"id"`
			Name      string    `json:"name"`
			Prefix    string    `json:"prefix"`
			CreatedAt time.Time `json:"created_at"`
		}

		var keys []keyRow
		for rows.Next() {
			var k keyRow
			if err := rows.Scan(&k.ID, &k.Name, &k.Prefix, &k.CreatedAt); err != nil {
				continue
			}
			keys = append(keys, k)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
	}
}

// DELETE /apikeys/{id} — revoke a key
func Revoke(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := middleware.GetClaims(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		userID := claims["user_id"].(string)

		// Get key ID from URL
		keyID := r.PathValue("id")

		var revokedID string
		err := db.QueryRow(context.Background(),
			`UPDATE api_keys SET revoked = true
			 WHERE id = $1 AND user_id = $2 AND revoked = false
			 RETURNING id`,
			keyID, userID,
		).Scan(&revokedID)
		if err != nil {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
