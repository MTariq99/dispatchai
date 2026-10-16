package conversation

import (
	"github.com/gin-gonic/gin"
)

func (c *ConversationHandler) RegisterConversationRoutes(routes *gin.RouterGroup) {
	routes.POST("/conversation", c.CreateConversation)
	routes.GET("/:conversation_id", c.GetConversation)
	routes.POST("/chat", c.Chat)
}
