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
)

// This file is the entry point for the reusable AI Assistant service.
//
// Its job is ONLY application startup and dependency wiring.
//
// It loads configuration, initializes infrastructure and adapters,
// creates the Assistant and its dependencies, registers the API routes,
// and starts the HTTP server.
//
// It does NOT contain:
//   - LLM reasoning logic
//   - tool execution logic
//   - business logic
//   - notification business rules
//   - database queries
//
// The runtime dependency chain is:
//
//   Config
//      ↓
//   LLM Client
//   Project Client
//   Memory
//   Policy Engine
//   Audit Store
//   Observability
//      ↓
//   Assistant
//      ↓
//   HTTP API
//
// The actual business operations remain inside the Host Project.

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
