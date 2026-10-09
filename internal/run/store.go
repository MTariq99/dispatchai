package run

import (
	"context"

	"github.com/google/uuid"
)

type RunStore interface {
	Create(ctx context.Context, r *Run) error
	Get(ctx context.Context, runID uuid.UUID) (*Run, error)
	GetByRequestID(ctx context.Context, requestID uuid.UUID) (*Run, error)
	Update(ctx context.Context, r *Run) error
}
