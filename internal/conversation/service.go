package conversation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type Store interface {
	Lock(ctx context.Context, conversationID uuid.UUID) (func() error, error)
	Load(conversationID uuid.UUID, integrationID uuid.UUID) ([]*models.Message, error)
	Save(conversationID uuid.UUID, integrationID uuid.UUID, messages []*models.Message) error
	CreateConversation(ctx context.Context, integrationID uuid.UUID, title string) (*models.Conversation, error)
	GetConversation(ctx context.Context, integrationID, conversationID uuid.UUID) (*models.Conversation, error)
}
type ConversationService struct {
	store Store
}

func NewConversationService(store Store) (*ConversationService, error) {
	if store == nil {
		return nil, errors.New("conversation store cannot be nil")
	}

	return &ConversationService{
		store: store,
	}, nil
}

func (cs *ConversationService) CreateConversation(ctx context.Context, integrationID uuid.UUID, req *models.CreateConversationReq) (*models.Conversation, error) {
	if integrationID == uuid.Nil {
		return nil, errors.New("integration id is required")
	}
	if req.Name == "" {
		return nil, errors.New("conversation name is required")
	}

	resp, err := cs.store.CreateConversation(ctx, integrationID, req.Name)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (cs *ConversationService) GetConversation(ctx context.Context, integrationID, conversationID uuid.UUID) (*models.Conversation, error) {
	if integrationID == uuid.Nil || conversationID == uuid.Nil {
		return nil, errors.New("missing required fields")
	}
	return cs.store.GetConversation(ctx, integrationID, conversationID)
}

func (cs *ConversationService) LockConversation(ctx context.Context, conversationID uuid.UUID) (func() error, error) {
	return cs.store.Lock(ctx, conversationID)
}

func (cs *ConversationService) LoadConversation(conversationID uuid.UUID, integrationID uuid.UUID) ([]*models.Message, error) {
	return cs.store.Load(conversationID, integrationID)
}

func (cs *ConversationService) SaveConversation(conversationID uuid.UUID, integrationID uuid.UUID, messages []*models.Message) error {
	return cs.store.Save(conversationID, integrationID, messages)
}
