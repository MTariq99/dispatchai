package run

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RunService struct {
	store RunStore
}

func NewRunService(store RunStore) *RunService {
	return &RunService{
		store: store,
	}
}

func (s *RunService) Create(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, conversationID uuid.UUID, requestID uuid.UUID, now time.Time) (*Run, error) {
	r, err := NewRun(tenantID, userID, conversationID, requestID, now)
	if err != nil {
		return nil, err
	}
	if err := s.store.Create(ctx, r); err != nil {
		return nil, err
	}

	return r, nil
}

func (s *RunService) Get(ctx context.Context, runID uuid.UUID) (*Run, error) {
	return s.store.Get(ctx, runID)
}

func (s *RunService) GetByRequestID(ctx context.Context, requestID uuid.UUID) (*Run, error) {
	return s.store.GetByRequestID(ctx, requestID)
}

func (s *RunService) TransitionTo(ctx context.Context, r *Run, next State, now time.Time) error {
	return s.transition(ctx, r, next, now)
}

func (s *RunService) Start(ctx context.Context, r *Run, now time.Time) error {
	return s.transition(ctx, r, StateRunning, now)
}

func (s *RunService) WaitForTools(ctx context.Context, r *Run, now time.Time) error {
	return s.transition(ctx, r, StateWaitingForTools, now)
}

func (s *RunService) WaitForApproval(ctx context.Context, r *Run, now time.Time) error {
	return s.transition(ctx, r, StateWaitingForApproval, now)
}

func (s *RunService) Complete(ctx context.Context, r *Run, now time.Time) error {
	return s.transition(ctx, r, StateCompleted, now)
}

func (s *RunService) Fail(ctx context.Context, r *Run, now time.Time) error {
	return s.transition(ctx, r, StateFailed, now)
}

func (s *RunService) Cancel(ctx context.Context, r *Run, now time.Time) error {
	return s.transition(ctx, r, StateCancelled, now)
}

func (s *RunService) transition(ctx context.Context, r *Run, next State, now time.Time) error {
	candidate := *r

	if err := candidate.TransitionTo(next, now); err != nil {
		return err
	}

	if err := s.store.Update(ctx, &candidate); err != nil {
		return err
	}
	*r = candidate
	return nil
}
