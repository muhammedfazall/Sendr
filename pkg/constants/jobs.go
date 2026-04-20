package constants

import "time"

const (
	JobLockDuration = 5 * time.Minute
	JobBatchSize    = 10
	JobMaxRetries   = 3
)
