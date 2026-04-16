package job

import (
	"context"
)

type Repository interface {
	Insert(ctx context.Context, j *Job) error
	GetByID(ctx context.Context, id string) (*Job, error)
}
