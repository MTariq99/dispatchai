package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/adapters/llm/gemini"
	"github.com/mtariq99/dispatchai/internal/api"
	"github.com/mtariq99/dispatchai/internal/config"
	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	geminiClient := gemini.NewGeminiClient(cfg)
	pc, err := project.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	toolGateway := project.NewProjectToolGateway(pc)
	registry := tools.NewRegistry()
	if err := registry.Register(tools.NewStaticTool(models.Definition{
		Name:        "get_customers",
		Description: "List customers for the current tenant, optionally paginated.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit":  map[string]any{"type": "integer", "description": "Max number of customers to return"},
				"offset": map[string]any{"type": "integer", "description": "Pagination offset"},
			},
		},
	})); err != nil {
		log.Fatal(err)
	}
	executer := tools.NewExecuter(registry, toolGateway)

	handler, err := api.NewHandler(cfg, geminiClient, registry, executer)

	router := gin.Default()
	AssistantHandler, err := api.NewAssistantHandler(cfg, handler)
	AssistantHandler.RegisterRoutes(router)

	fmt.Println("DispatchAI server is listening on : 8989")
	if err := router.Run(":8989"); err != nil {
		log.Fatal(err)
	}
}
