package assistant

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/models"
)

type lockTestStore struct {
	conversation.Store

	messages []*models.Message

	lockCalls   int
	unlockCalls int
	loadCalls   int
	saveCalls   int

	lockErr   error
	unlockErr error
	loadErr   error
	saveErr   error
}

func (s *lockTestStore) Lock(_ context.Context, _ uuid.UUID) (func() error, error) {
	s.lockCalls++
	if s.lockErr != nil {
		return nil, s.lockErr
	}

	return func() error {
		s.unlockCalls++
		return s.unlockErr
	}, nil
}

func (s *lockTestStore) Load(conversationID uuid.UUID, integrationID uuid.UUID) ([]*models.Message, error) {
	s.loadCalls++
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	return s.messages, nil
}

func (s *lockTestStore) Save(conversationID uuid.UUID, integrationID uuid.UUID, messages []*models.Message) error {
	s.saveCalls++
	if s.saveErr != nil {
		return s.saveErr
	}
	s.messages = messages
	return nil
}

var _ conversation.Store = (*lockTestStore)(nil)

func newLockTestOrchestrator(
	t *testing.T,
	llmClient *testLLM,
	store *lockTestStore,
) *Orchestrator {
	t.Helper()
	conversationService, err := conversation.NewConversationService(store)
	if err != nil {
		t.Fatalf("NewConversationService() error = %v", err)
	}

	return NewOrchestrator(
		llmClient,
		nil,
		nil,
		"test-model",
		1000,
		0,
		conversationService,
	)
}

func newLockTestExecutionContext() *models.ExecutionContext {
	return &models.ExecutionContext{
		RequestID:      "lock-test-request",
		RunID:          uuid.New(),
		ConversationID: uuid.New(),
		IntegrationID:  uuid.New(),
	}
}

func TestOrchestratorReleasesConversationLockOnSuccess(t *testing.T) {
	store := &lockTestStore{}
	llmClient := &testLLM{
		responses: []*models.Response{
			{Content: "Hello."},
		},
	}
	orchestrator := newLockTestOrchestrator(t, llmClient, store)

	answer, err := orchestrator.Run(
		context.Background(),
		newLockTestExecutionContext(),
		"Hi",
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if answer != "Hello." {
		t.Fatalf("Run() answer = %q, want %q", answer, "Hello.")
	}
	if store.lockCalls != 1 {
		t.Errorf("lock calls = %d, want 1", store.lockCalls)
	}
	if store.unlockCalls != 1 {
		t.Errorf("unlock calls = %d, want 1", store.unlockCalls)
	}
	if store.loadCalls != 1 {
		t.Errorf("load calls = %d, want 1", store.loadCalls)
	}
	if store.saveCalls != 1 {
		t.Errorf("save calls = %d, want 1", store.saveCalls)
	}
}

func TestOrchestratorReleasesConversationLockWhenLoadFails(t *testing.T) {
	store := &lockTestStore{
		loadErr: errors.New("load failed"),
	}
	orchestrator := newLockTestOrchestrator(t, &testLLM{}, store)

	_, err := orchestrator.Run(
		context.Background(),
		newLockTestExecutionContext(),
		"Hi",
	)
	if err == nil {
		t.Fatal("Run() error = nil, want load error")
	}
	if store.unlockCalls != 1 {
		t.Errorf("unlock calls = %d, want 1", store.unlockCalls)
	}
	if store.saveCalls != 0 {
		t.Errorf("save calls = %d, want 0", store.saveCalls)
	}
}

func TestOrchestratorReleasesConversationLockWhenLLMFails(t *testing.T) {
	store := &lockTestStore{}
	orchestrator := newLockTestOrchestrator(t, &testLLM{}, store)

	_, err := orchestrator.Run(
		context.Background(),
		newLockTestExecutionContext(),
		"Hi",
	)
	if err == nil {
		t.Fatal("Run() error = nil, want LLM error")
	}
	if store.unlockCalls != 1 {
		t.Errorf("unlock calls = %d, want 1", store.unlockCalls)
	}
	if store.saveCalls != 0 {
		t.Errorf("save calls = %d, want 0", store.saveCalls)
	}
}

func TestOrchestratorReturnsUnlockErrorAfterSuccessfulRun(t *testing.T) {
	store := &lockTestStore{
		unlockErr: errors.New("unlock failed"),
	}
	llmClient := &testLLM{
		responses: []*models.Response{
			{Content: "Hello."},
		},
	}
	orchestrator := newLockTestOrchestrator(t, llmClient, store)

	answer, err := orchestrator.Run(
		context.Background(),
		newLockTestExecutionContext(),
		"Hi",
	)
	if err == nil {
		t.Fatal("Run() error = nil, want unlock error")
	}
	if answer != "" {
		t.Errorf("Run() answer = %q, want empty answer on unlock failure", answer)
	}
	if store.unlockCalls != 1 {
		t.Errorf("unlock calls = %d, want 1", store.unlockCalls)
	}
}
