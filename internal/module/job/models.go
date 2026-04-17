package job

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusSent       Status = "sent"
	StatusFailed     Status = "failed"
)

type Job struct {
	ID uuid.UUID
	UserID string
	APIKeyID string
	Payload []byte
	Status Status
	Retries int
	MaxRetries int
	CreatedAt time.Time
	UpdatedAt time.Time
}