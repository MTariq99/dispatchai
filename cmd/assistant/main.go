package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/mtariq99/dispatchai/adapters/llm/gemini"
	storage "github.com/mtariq99/dispatchai/adapters/storage/postgres"
	"github.com/mtariq99/dispatchai/internal/api"
	"github.com/mtariq99/dispatchai/internal/config"
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/internal/idempotency"
	assistant "github.com/mtariq99/dispatchai/internal/orchestration"
	"github.com/mtariq99/dispatchai/internal/policy"
	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/internal/run"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	logger, err := storage.BuildLogger(cfg.App.Env)
	if err != nil {
		log.Fatalf("logger: %v", err)
	}
	defer logger.Sync()

	db, err := storage.InitDB(cfg, logger)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	geminiClient, err := gemini.NewGeminiClient(cfg)
	if err != nil {
		log.Fatalf("initialize Gemini client: %v", err)
	}

	// -------------------------------------------------------------------------
	// Run
	// -------------------------------------------------------------------------

	runStore := storage.NewRunStore(db)
	runService := run.NewRunService(runStore)

	// -------------------------------------------------------------------------
	// Conversation
	// -------------------------------------------------------------------------

	conversationStore := conversation.NewMemoryStore(db)

	// -------------------------------------------------------------------------
	// Project / Host integration
	// -------------------------------------------------------------------------

	pc, err := project.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	toolGateway := project.NewProjectToolGateway(pc)

	// -------------------------------------------------------------------------
	// Tool registry
	// -------------------------------------------------------------------------

	registry := tools.NewRegistry()

	if err := registry.Register(tools.NewStaticTool(models.Definition{
		Name:        "get_customers",
		Description: "List customers for the current tenant, optionally paginated.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"limit": map[string]any{
					"type":        "integer",
					"description": "Max number of customers to return",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Pagination offset",
				},
			},
		},
	})); err != nil {
		log.Fatal(err)
	}

	if err := registry.Register(tools.NewStaticTool(models.Definition{
		Name:        "find_delayed_customers",
		Description: "Find customers whose orders have been delayed beyond a specified number of minutes.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"delay_minutes": map[string]any{
					"type":        "integer",
					"description": "Minimum delay in minutes to filter by",
				},
			},
		},
	})); err != nil {
		log.Fatal(err)
	}

	if err := registry.Register(tools.NewStaticTool(models.Definition{
		Name:        "send_notification",
		Description: "Send a notification message to a specific customer.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"customer_id": map[string]any{
					"type":        "string",
					"description": "The ID of the customer to notify",
				},
				"message": map[string]any{
					"type":        "string",
					"description": "The notification message content",
				},
			},
			"required": []string{
				"customer_id",
				"message",
			},
		},
	})); err != nil {
		log.Fatal(err)
	}

	// -------------------------------------------------------------------------
	// Policy
	// -------------------------------------------------------------------------

	policyEngine := policy.NewEngine(
		policy.MaxNotificationsPerRun(1),
	)

	// -------------------------------------------------------------------------
	// Tool idempotency
	//
	// Protects business tool execution inside a Run.
	// Key: (run_id, call_id)
	// -------------------------------------------------------------------------

	toolIdempotency := idempotency.NewToolIdempotency(db)

	executor := tools.NewExecutor(
		registry,
		toolGateway,
		policyEngine,
		toolIdempotency,
	)

	// -------------------------------------------------------------------------
	// Orchestration
	// -------------------------------------------------------------------------

	orchestrator := assistant.NewOrchestrator(
		geminiClient,
		registry,
		executor,
		cfg.LLM.Model,
		cfg.LLM.MaxTokens,
		cfg.LLM.Temperature,
		conversationStore,
	)

	assistantService := assistant.NewAssistant(orchestrator)

	// -------------------------------------------------------------------------
	// Request idempotency
	//
	// Protects the API request before a Run exists.
	// Key: (tenant_id, idempotency_key)
	// -------------------------------------------------------------------------

	requestIdempotency := idempotency.NewIdempotencyStore(db)

	// -------------------------------------------------------------------------
	// API
	//
	// Handler only knows about:
	//   - Assistant
	//   - RunService
	//   - RequestIdempotency
	//
	// It does not know about Gemini, tools, registry, or tool execution.
	// -------------------------------------------------------------------------

	handler, err := api.NewHandler(
		cfg,
		assistantService,
		runService,
		requestIdempotency,
	)
	if err != nil {
		log.Fatalf("handler: %v", err)
	}

	assistantHandler, err := api.NewAssistantHandler(cfg, handler)
	if err != nil {
		log.Fatalf("assistant handler: %v", err)
	}

	// -------------------------------------------------------------------------
	// HTTP server
	// -------------------------------------------------------------------------

	router := gin.Default()

	assistantHandler.RegisterRoutes(router)

	fmt.Println("DispatchAI server is listening on :8989")

	if err := router.Run(":8989"); err != nil {
		log.Fatal(err)
	}
}
