package domain

import "time"

// User is the authenticated principal.
type User struct {
  ID        string
  Email     string
  Name      string
  GoogleID  string
  PlanID    string
  CreatedAt time.Time
}

// Plan represents a billing tier and its limits.
type Plan struct {
  ID         string
  Name       string
  DailyLimit int
  CreatedAt  time.Time
}

// APIKey is the domain representation of an API credential.
// Hashed is never transmitted — only stored and compared.
type APIKey struct {
  ID        string
  UserID    string
  Name      string
  Prefix    string
  Hashed    string  // SHA-256 of the secret half — never exposed
  Revoked   bool
  CreatedAt time.Time
}

// Job represents a queued email send task.
type Job struct {
  ID         string
  UserID     string
  APIKeyID   string
  Payload    EmailPayload
  Status     string
  Retries    int
  MaxRetries int
  RunAt      time.Time
  LockedUntil *time.Time
  CreatedAt  time.Time
  UpdatedAt  time.Time
}

// EmailPayload is the data needed to send a single email.
type EmailPayload struct {
  To      string `json:"to"`
  Subject string `json:"subject"`
  Body    string `json:"body"`
  HTML    bool   `json:"html"`
}