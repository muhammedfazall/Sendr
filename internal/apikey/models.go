package apikey

import "time"

// APIKey represents an API key record from the database.
type APIKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Prefix    string    `json:"prefix"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateKeyRequest is the expected JSON body for creating a new API key.
type CreateKeyRequest struct {
	Name string `json:"name"`
}

// CreateKeyResponse is returned once after key creation — the full key is never shown again.
type CreateKeyResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Prefix string `json:"prefix"`
	APIKey string `json:"api_key"`
}
