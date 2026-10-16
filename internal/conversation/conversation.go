package conversation

import (
	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/models"
)

type Conversation struct {
	Messages []*models.Message
}

func NewConversation() *Conversation {
	return &Conversation{
		Messages: make([]*models.Message, 0),
	}
}

func (c *Conversation) AddSystemMessage(content string) {
	c.Messages = append(c.Messages, &models.Message{
		Role:    enums.RoleSystem,
		Content: content,
	})
}

func (c *Conversation) AddUserMessage(content string) {
	c.Messages = append(c.Messages, &models.Message{
		Role:    enums.RoleUser,
		Content: content,
	})
}

func (c *Conversation) AddAssistantMessage(msg models.Message) {
	msg.Role = enums.RoleAssistant
	c.Messages = append(c.Messages, &msg)
}

func (c *Conversation) AddToolResult(toolCallID, content string) {
	c.Messages = append(c.Messages, &models.Message{
		Role:       enums.RoleTool,
		ToolCallID: toolCallID,
		Content:    content,
	})
}
