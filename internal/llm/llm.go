package llm

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

type Client interface {
	Generate(ctx context.Context, req models.Request) (models.Response, error)
}
