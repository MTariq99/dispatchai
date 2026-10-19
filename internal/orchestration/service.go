package assistant

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

type Assistant struct {
	orchestrator *Orchestrator
}

func NewAssistant(orchestrator *Orchestrator) *Assistant {
	return &Assistant{orchestrator: orchestrator}
}

func (a *Assistant) HandleRequest(ctx context.Context, userMessage string, execCtx models.ExecutionContext) (string, error) {
	return a.orchestrator.Run(ctx, &execCtx, userMessage)
}
