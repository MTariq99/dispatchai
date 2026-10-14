package tools

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

type Tool interface {
	Name() string
	Definition() models.Definition
	Execute(ctx context.Context, call models.Call) models.Result
}
