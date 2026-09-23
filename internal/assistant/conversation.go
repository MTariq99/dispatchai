package assistant

import (
	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/models"
)

// This file manages the conversation state used during an Assistant
// interaction.
//
// It records messages exchanged between the user, Assistant, LLM,
// tool calls, and tool results.
//
// A conversation may look like:
//
//   User Message
//      ↓
//   Assistant -> Tool Call
//      ↓
//   Tool Result
//      ↓
//   Assistant -> Another Tool Call
//      ↓
//   Tool Result
//      ↓
//   Final Assistant Message
//
// The conversation exists so the LLM can understand what has already
// happened and what information has already been retrieved.
//
// It does NOT store business truth.
// Live business state must come from Host Project tools.

type Conversation struct {
	Messages []models.Message
}

func NewConversation() *Conversation {
	return &Conversation{
		Messages: make([]models.Message, 0),
	}
}

func (c *Conversation) AddSystemMessage(content string) {
	c.Messages = append(c.Messages, models.Message{
		Role:    enums.RoleAssistant,
		Content: content,
	})
}

func (c *Conversation) AddUserMessage(content string) {
	c.Messages = append(c.Messages, models.Message{
		Role:    enums.RoleUser,
		Content: content,
	})
}

func (c *Conversation) AddAssistantMessage(msg models.Message) {
	msg.Role = enums.RoleAssistant
	c.Messages = append(c.Messages, msg)
}

func (c *Conversation) AddToolResult(toolCallID, content string) {
	c.Messages = append(c.Messages, models.Message{
		Role:       enums.RoleTool,
		ToolCallID: toolCallID,
		Content:    content,
	})
}
