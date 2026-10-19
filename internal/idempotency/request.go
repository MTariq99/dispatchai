package idempotency

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type RequestIdempotency interface {
	Reserve(ctx context.Context, integrationID uuid.UUID, key string, conversationID uuid.UUID, requestID uuid.UUID, fingerprint string) (bool, error)
	Get(ctx context.Context, integrationID uuid.UUID, key string) (*models.ChatResponse, bool, error)
	Complete(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, response *models.ChatResponse) error
	Fail(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, response *models.ChatResponse) error
	FailBeforeRun(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, response *models.ChatResponse) error
	GetState(ctx context.Context, integrationID uuid.UUID, key string) (*models.RequestState, bool, error)
}

type IdempotencyService struct {
	ReqIdempotencyStore RequestIdempotency
}

func NewIdempotencyService(reqIdempotency RequestIdempotency) (*IdempotencyService, error) {
	if reqIdempotency == nil {
		return nil, errors.New("request idempotency cannot be nil")
	}

	return &IdempotencyService{
		ReqIdempotencyStore: reqIdempotency,
	}, nil
}

func (s *IdempotencyService) CompleteReq(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, response *models.ChatResponse) error {
	if integrationID == uuid.Nil {
		return errors.New("integration id is required")
	}
	if key == "" {
		return errors.New("idempotency key is required")
	}
	if requestID == uuid.Nil {
		return errors.New("request id is required")
	}
	if runID == uuid.Nil {
		return errors.New("run id is required")
	}
	if response == nil {
		return errors.New("response is required")
	}

	return s.ReqIdempotencyStore.Complete(ctx, integrationID, key, requestID, runID, response)
}

func (s *IdempotencyService) ReserveReq(ctx context.Context, integrationID uuid.UUID, key string, conversationID uuid.UUID, requestID uuid.UUID, fingerprint string) (bool, error) {
	if integrationID == uuid.Nil {
		return false, errors.New("integration id is required")
	}
	if key == "" {
		return false, errors.New("idempotency key is required")
	}
	if conversationID == uuid.Nil {
		return false, errors.New("conversation id is required")
	}
	if requestID == uuid.Nil {
		return false, errors.New("request id is required")
	}

	if fingerprint == "" {
		return false, errors.New("request fingerprint is required")
	}
	isReserved, err := s.ReqIdempotencyStore.Reserve(ctx, integrationID, key, conversationID, requestID, fingerprint)
	if err != nil {
		return false, err
	}

	return isReserved, nil
}

func (s *IdempotencyService) Fail(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, response *models.ChatResponse) error {
	if integrationID == uuid.Nil {
		return errors.New("integration id is required")
	}
	if key == "" {
		return errors.New("idempotency key is required")
	}
	if requestID == uuid.Nil {
		return errors.New("request id is required")
	}
	if runID == uuid.Nil {
		return errors.New("run id is required")
	}
	if response == nil {
		return errors.New("response is required")
	}

	return s.ReqIdempotencyStore.Fail(ctx, integrationID, key, requestID, runID, response)
}

func (s *IdempotencyService) FailBeforeRun(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, response *models.ChatResponse) error {
	if integrationID == uuid.Nil {
		return errors.New("integration id is required")
	}

	if key == "" {
		return errors.New("idempotency key is required")
	}

	if requestID == uuid.Nil {
		return errors.New("request id is required")
	}

	if response == nil {
		return errors.New("response is required")
	}
	return s.ReqIdempotencyStore.FailBeforeRun(ctx, integrationID, key, requestID, response)
}

func (s *IdempotencyService) GetState(ctx context.Context, integrationID uuid.UUID, key string) (*models.RequestState, bool, error) {
	if integrationID == uuid.Nil {
		return nil, false, errors.New("integration id is required")
	}

	if key == "" {
		return nil, false, errors.New("idempotency key is required")
	}

	return s.ReqIdempotencyStore.GetState(ctx, integrationID, key)
}
