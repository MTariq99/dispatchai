package run

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type runStoreStub struct {
	updateErr error
	updateRun *Run
}

func (s *runStoreStub) Create(context.Context, *Run) error {
	return nil
}

func (s *runStoreStub) Get(context.Context, uuid.UUID) (*Run, error) {
	return nil, nil
}

func (s *runStoreStub) GetByRequestID(context.Context, uuid.UUID) (*Run, error) {
	return nil, nil
}

func (s *runStoreStub) Update(_ context.Context, r *Run) error {
	candidate := *r
	s.updateRun = &candidate

	return s.updateErr
}

func TestRunServiceTransitionDoesNotMutateRunWhenUpdateFails(t *testing.T) {
	persistErr := errors.New("database unavailable")
	store := &runStoreStub{
		updateErr: persistErr,
	}
	service := NewRunService(store)

	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	r, err := NewRun(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		now,
	)
	if err != nil {
		t.Fatalf("NewRun() error = %v", err)
	}

	original := *r

	err = service.Start(context.Background(), r, now.Add(time.Minute))
	if !errors.Is(err, persistErr) {
		t.Fatalf("Start() error = %v, want %v", err, persistErr)
	}

	if r.Status != original.Status {
		t.Errorf(
			"Status changed to %q after persistence failure; want %q",
			r.Status,
			original.Status,
		)
	}

	if !r.StartedAt.Equal(original.StartedAt) {
		t.Errorf(
			"StartedAt changed after persistence failure: got %v, want %v",
			r.StartedAt,
			original.StartedAt,
		)
	}

	if !r.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf(
			"UpdatedAt changed after persistence failure: got %v, want %v",
			r.UpdatedAt,
			original.UpdatedAt,
		)
	}

	if store.updateRun == nil {
		t.Fatal("expected store.Update to be called")
	}

	if store.updateRun.Status != StateRunning {
		t.Errorf(
			"persisted candidate status = %q, want %q",
			store.updateRun.Status,
			StateRunning,
		)
	}
}

func TestRunServiceTransitionToDoesNotMutateRunWhenUpdateFails(t *testing.T) {
	persistErr := errors.New("database unavailable")
	store := &runStoreStub{
		updateErr: persistErr,
	}
	service := NewRunService(store)

	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

	r, err := NewRun(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		now,
	)
	if err != nil {
		t.Fatalf("NewRun() error = %v", err)
	}

	original := *r

	err = service.TransitionTo(context.Background(), r, StateRunning, now.Add(time.Minute))
	if !errors.Is(err, persistErr) {
		t.Fatalf("TransitionTo() error = %v, want %v", err, persistErr)
	}

	if r.Status != original.Status {
		t.Errorf("Status changed to %q after persistence failure; want %q", r.Status, original.Status)
	}

	if !r.StartedAt.Equal(original.StartedAt) {
		t.Errorf("StartedAt changed after persistence failure: got %v, want %v", r.StartedAt, original.StartedAt)
	}

	if !r.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("UpdatedAt changed after persistence failure: got %v, want %v", r.UpdatedAt, original.UpdatedAt)
	}

	if store.updateRun == nil {
		t.Fatal("expected store.Update to be called")
	}

	if store.updateRun.Status != StateRunning {
		t.Errorf("persisted candidate status = %q, want %q", store.updateRun.Status, StateRunning)
	}
}
