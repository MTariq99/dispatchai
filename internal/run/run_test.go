package run

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewRun(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	conversationID := uuid.New()
	requestID := uuid.New()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		tenant  uuid.UUID
		user    uuid.UUID
		convo   uuid.UUID
		request uuid.UUID
		wantErr error
	}{
		{
			name:    "valid",
			tenant:  tenantID,
			user:    userID,
			convo:   conversationID,
			request: requestID,
		},
		{
			name:    "missing tenant",
			tenant:  uuid.Nil,
			user:    userID,
			convo:   conversationID,
			request: requestID,
			wantErr: ErrInvalidTenantID,
		},
		{
			name:    "missing user",
			tenant:  tenantID,
			user:    uuid.Nil,
			convo:   conversationID,
			request: requestID,
			wantErr: ErrInvalidUserID,
		},
		{
			name:    "missing conversation",
			tenant:  tenantID,
			user:    userID,
			convo:   uuid.Nil,
			request: requestID,
			wantErr: ErrInvalidConversation,
		},
		{
			name:    "missing request",
			tenant:  tenantID,
			user:    userID,
			convo:   conversationID,
			request: uuid.Nil,
			wantErr: ErrInvalidRequestID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewRun(
				tt.tenant,
				tt.user,
				tt.convo,
				tt.request,
				now,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}

			if tt.wantErr != nil {
				if r != nil {
					t.Fatal("expected nil run on error")
				}
				return
			}

			if r == nil {
				t.Fatal("expected run, got nil")
			}

			if r.ID == uuid.Nil {
				t.Fatal("expected generated run ID")
			}

			if r.TenantID != tt.tenant {
				t.Fatalf("tenant ID = %v, want %v", r.TenantID, tt.tenant)
			}

			if r.UserID != tt.user {
				t.Fatalf("user ID = %v, want %v", r.UserID, tt.user)
			}

			if r.ConversationID != tt.convo {
				t.Fatalf("conversation ID = %v, want %v", r.ConversationID, tt.convo)
			}

			if r.RequestID != tt.request {
				t.Fatalf("request ID = %v, want %v", r.RequestID, tt.request)
			}

			if r.Status != StateCreated {
				t.Fatalf("status = %v, want %v", r.Status, StateCreated)
			}

			if !r.CreatedAt.Equal(now) {
				t.Fatalf("created at = %v, want %v", r.CreatedAt, now)
			}

			if !r.UpdatedAt.Equal(now) {
				t.Fatalf("updated at = %v, want %v", r.UpdatedAt, now)
			}

			if !r.StartedAt.IsZero() {
				t.Fatal("started at should initially be zero")
			}

			if r.CompletedAt != nil {
				t.Fatal("completed at should initially be nil")
			}
		})
	}
}

func TestNewRun_ZeroTimeUsesCurrentTime(t *testing.T) {
	before := time.Now().UTC()

	r, err := NewRun(
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Time{},
	)
	if err != nil {
		t.Fatalf("NewRun() error = %v", err)
	}

	after := time.Now().UTC()

	if r.CreatedAt.Before(before) || r.CreatedAt.After(after) {
		t.Fatalf(
			"created at = %v, expected between %v and %v",
			r.CreatedAt,
			before,
			after,
		)
	}

	if !r.UpdatedAt.Equal(r.CreatedAt) {
		t.Fatalf("updated at = %v, want %v", r.UpdatedAt, r.CreatedAt)
	}
}

func TestRunTransitionTo(t *testing.T) {
	baseTime := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initial       State
		next          State
		wantErr       error
		wantStatus    State
		wantStarted   bool
		wantCompleted bool
	}{
		{
			name:          "created to running",
			initial:       StateCreated,
			next:          StateRunning,
			wantStatus:    StateRunning,
			wantStarted:   true,
			wantCompleted: false,
		},
		{
			name:          "created to cancelled",
			initial:       StateCreated,
			next:          StateCancelled,
			wantStatus:    StateCancelled,
			wantStarted:   false,
			wantCompleted: true,
		},
		{
			name:          "running to waiting for tools",
			initial:       StateRunning,
			next:          StateWaitingForTools,
			wantStatus:    StateWaitingForTools,
			wantStarted:   false,
			wantCompleted: false,
		},
		{
			name:          "running to waiting for approval",
			initial:       StateRunning,
			next:          StateWaitingForApproval,
			wantStatus:    StateWaitingForApproval,
			wantStarted:   false,
			wantCompleted: false,
		},
		{
			name:          "running to completed",
			initial:       StateRunning,
			next:          StateCompleted,
			wantStatus:    StateCompleted,
			wantStarted:   false,
			wantCompleted: true,
		},
		{
			name:          "running to failed",
			initial:       StateRunning,
			next:          StateFailed,
			wantStatus:    StateFailed,
			wantStarted:   false,
			wantCompleted: true,
		},
		{
			name:          "running to cancelled",
			initial:       StateRunning,
			next:          StateCancelled,
			wantStatus:    StateCancelled,
			wantStarted:   false,
			wantCompleted: true,
		},
		{
			name:          "waiting for tools to running",
			initial:       StateWaitingForTools,
			next:          StateRunning,
			wantStatus:    StateRunning,
			wantStarted:   false,
			wantCompleted: false,
		},
		{
			name:          "waiting for tools to failed",
			initial:       StateWaitingForTools,
			next:          StateFailed,
			wantStatus:    StateFailed,
			wantStarted:   false,
			wantCompleted: true,
		},
		{
			name:          "waiting for approval to running",
			initial:       StateWaitingForApproval,
			next:          StateRunning,
			wantStatus:    StateRunning,
			wantStarted:   false,
			wantCompleted: false,
		},
		{
			name:          "waiting for approval to cancelled",
			initial:       StateWaitingForApproval,
			next:          StateCancelled,
			wantStatus:    StateCancelled,
			wantStarted:   false,
			wantCompleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Run{
				ID:        uuid.New(),
				Status:    tt.initial,
				CreatedAt: baseTime,
				UpdatedAt: baseTime,
			}

			err := r.TransitionTo(tt.next, baseTime.Add(time.Minute))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("TransitionTo() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			if r.Status != tt.wantStatus {
				t.Fatalf("status = %v, want %v", r.Status, tt.wantStatus)
			}

			if !r.UpdatedAt.Equal(baseTime.Add(time.Minute)) {
				t.Fatalf(
					"updated at = %v, want %v",
					r.UpdatedAt,
					baseTime.Add(time.Minute),
				)
			}

			if tt.wantStarted && r.StartedAt.IsZero() {
				t.Fatal("expected StartedAt to be set")
			}

			if !tt.wantStarted && tt.initial != StateRunning && tt.initial != StateWaitingForTools && tt.initial != StateWaitingForApproval {
				if !r.StartedAt.IsZero() {
					t.Fatal("StartedAt should remain zero")
				}
			}

			if tt.wantCompleted && r.CompletedAt == nil {
				t.Fatal("expected CompletedAt to be set")
			}

			if !tt.wantCompleted && r.CompletedAt != nil {
				t.Fatal("CompletedAt should remain nil")
			}
		})
	}
}

func TestRunTransitionTo_InvalidTransitions(t *testing.T) {
	tests := []struct {
		name    string
		initial State
		next    State
	}{
		{
			name:    "created to completed",
			initial: StateCreated,
			next:    StateCompleted,
		},
		{
			name:    "created to failed",
			initial: StateCreated,
			next:    StateFailed,
		},
		{
			name:    "created to waiting for tools",
			initial: StateCreated,
			next:    StateWaitingForTools,
		},
		{
			name:    "completed to running",
			initial: StateCompleted,
			next:    StateRunning,
		},
		{
			name:    "failed to running",
			initial: StateFailed,
			next:    StateRunning,
		},
		{
			name:    "cancelled to running",
			initial: StateCancelled,
			next:    StateRunning,
		},
		{
			name:    "waiting for tools to completed",
			initial: StateWaitingForTools,
			next:    StateCompleted,
		},
		{
			name:    "waiting for approval to completed",
			initial: StateWaitingForApproval,
			next:    StateCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

			r := &Run{
				ID:        uuid.New(),
				Status:    tt.initial,
				CreatedAt: now,
				UpdatedAt: now,
			}

			err := r.TransitionTo(tt.next, now.Add(time.Minute))
			if !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf(
					"TransitionTo() error = %v, want %v",
					err,
					ErrInvalidTransition,
				)
			}

			if r.Status != tt.initial {
				t.Fatalf("status changed to %v after invalid transition", r.Status)
			}

			if !r.UpdatedAt.Equal(now) {
				t.Fatalf("UpdatedAt changed after invalid transition")
			}
		})
	}
}

func TestRunTransitionTo_SameStateIsIdempotent(t *testing.T) {
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	r := &Run{
		ID:        uuid.New(),
		Status:    StateRunning,
		CreatedAt: now,
		UpdatedAt: now,
		StartedAt: now,
	}

	err := r.TransitionTo(StateRunning, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	if !r.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt changed on same-state transition")
	}

	if !r.StartedAt.Equal(now) {
		t.Fatalf("StartedAt changed on same-state transition")
	}
}

func TestRunTransitionTo_StartedAtIsSetOnlyOnce(t *testing.T) {
	createdAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	startedAt := createdAt.Add(time.Minute)
	later := createdAt.Add(5 * time.Minute)

	r := &Run{
		ID:        uuid.New(),
		Status:    StateCreated,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	if err := r.TransitionTo(StateRunning, startedAt); err != nil {
		t.Fatalf("first transition error = %v", err)
	}

	if !r.StartedAt.Equal(startedAt) {
		t.Fatalf("StartedAt = %v, want %v", r.StartedAt, startedAt)
	}

	if err := r.TransitionTo(StateWaitingForTools, later); err != nil {
		t.Fatalf("second transition error = %v", err)
	}

	if err := r.TransitionTo(StateRunning, later.Add(time.Minute)); err != nil {
		t.Fatalf("third transition error = %v", err)
	}

	if !r.StartedAt.Equal(startedAt) {
		t.Fatalf(
			"StartedAt was overwritten: got %v, want %v",
			r.StartedAt,
			startedAt,
		)
	}
}

func TestRunTransitionTo_CompletedAtIsSetOnlyOnce(t *testing.T) {
	createdAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	completedAt := createdAt.Add(5 * time.Minute)

	r := &Run{
		ID:        uuid.New(),
		Status:    StateRunning,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		StartedAt: createdAt.Add(time.Minute),
	}

	if err := r.TransitionTo(StateCompleted, completedAt); err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	if r.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}

	if !r.CompletedAt.Equal(completedAt) {
		t.Fatalf(
			"CompletedAt = %v, want %v",
			*r.CompletedAt,
			completedAt,
		)
	}

	originalCompletedAt := *r.CompletedAt

	// Terminal states cannot transition, so the original timestamp must remain.
	err := r.TransitionTo(StateFailed, completedAt.Add(time.Minute))
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}

	if !r.CompletedAt.Equal(originalCompletedAt) {
		t.Fatal("CompletedAt changed after invalid transition")
	}
}

func TestRunTransitionTo_ZeroTimeUsesCurrentTime(t *testing.T) {
	before := time.Now().UTC()

	r := &Run{
		ID:        uuid.New(),
		Status:    StateCreated,
		CreatedAt: before,
		UpdatedAt: before,
	}

	err := r.TransitionTo(StateRunning, time.Time{})
	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	after := time.Now().UTC()

	if r.StartedAt.Before(before) || r.StartedAt.After(after) {
		t.Fatalf(
			"StartedAt = %v, expected between %v and %v",
			r.StartedAt,
			before,
			after,
		)
	}

	if r.UpdatedAt.Before(before) || r.UpdatedAt.After(after) {
		t.Fatalf(
			"UpdatedAt = %v, expected between %v and %v",
			r.UpdatedAt,
			before,
			after,
		)
	}
}
