package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/mtariq99/dispatchai/models"
)

var (
	errInvalidIdempotencyRequest = errors.New("invalid idempotency request")
	errIdempotencyKeyConflict    = errors.New("idempotency key reused with different request")
)

type idempotencyEntry struct {
	fingerprint string
	done        chan struct{}
	response    *models.ToolResponse
	err         error
}

type idempotencyStore struct {
	mu      sync.Mutex
	entries map[string]*idempotencyEntry
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{
		entries: make(map[string]*idempotencyEntry),
	}
}

func (s *idempotencyStore) execute(ctx context.Context, req *models.ToolRequest, fn func() (*models.ToolResponse, error)) (*models.ToolResponse, error) {
	if req == nil ||
		req.RunID == uuid.Nil ||
		req.CallID == "" ||
		req.ToolName == "" ||
		len(req.Arguments) == 0 {
		return nil, errInvalidIdempotencyRequest
	}

	fingerprint, err := requestFingerprint(req)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid arguments: %v", errInvalidIdempotencyRequest, err)
	}

	key := req.RunID.String() + ":" + req.CallID

	for {
		s.mu.Lock()

		if existing, ok := s.entries[key]; ok {
			if existing.fingerprint != fingerprint {
				s.mu.Unlock()
				return nil, errIdempotencyKeyConflict
			}

			done := existing.done
			s.mu.Unlock()

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-done:
			}

			// If the original execution failed, retry by checking the map
			// again. Otherwise, return its cached response.
			if existing.err != nil {
				continue
			}

			return cloneToolResponse(existing.response)
		}

		entry := &idempotencyEntry{
			fingerprint: fingerprint,
			done:        make(chan struct{}),
		}
		s.entries[key] = entry
		s.mu.Unlock()

		response, executionErr := fn()

		s.mu.Lock()

		if executionErr != nil || response == nil {
			if executionErr == nil {
				executionErr = errors.New("tool returned a nil response")
			}

			entry.err = executionErr
			delete(s.entries, key)
			close(entry.done)
			s.mu.Unlock()

			return nil, executionErr
		}

		cachedResponse, cloneErr := cloneToolResponse(response)
		if cloneErr != nil {
			entry.err = cloneErr
			delete(s.entries, key)
			close(entry.done)
			s.mu.Unlock()

			return nil, cloneErr
		}

		entry.response = cachedResponse
		close(entry.done)
		s.mu.Unlock()

		return cloneToolResponse(cachedResponse)
	}
}

func requestFingerprint(req *models.ToolRequest) (string, error) {
	var arguments any

	if err := json.Unmarshal(req.Arguments, &arguments); err != nil {
		return "", err
	}

	canonicalArguments, err := json.Marshal(arguments)
	if err != nil {
		return "", err
	}

	fingerprint, err := json.Marshal(struct {
		ToolName      string                      `json:"tool_name"`
		Arguments     json.RawMessage             `json:"arguments"`
		Identity      models.Identity             `json:"identity"`
		Tenant        models.TenantContext        `json:"tenant"`
		Authorization models.AuthorizationContext `json:"authorization"`
	}{
		ToolName:      req.ToolName,
		Arguments:     canonicalArguments,
		Identity:      req.Identity,
		Tenant:        req.Tenant,
		Authorization: req.Authorization,
	})
	if err != nil {
		return "", err
	}

	return string(fingerprint), nil
}

func cloneToolResponse(response *models.ToolResponse) (*models.ToolResponse, error) {
	if response == nil {
		return nil, errors.New("cannot clone a nil tool response")
	}
	data, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	var cloned models.ToolResponse
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}

	return &cloned, nil
}
