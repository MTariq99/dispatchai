package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mtariq99/dispatchai/adapters/llm/gemini"
	storage "github.com/mtariq99/dispatchai/adapters/storage/postgres"
	"github.com/mtariq99/dispatchai/internal/api"
	"github.com/mtariq99/dispatchai/internal/authn"
	"github.com/mtariq99/dispatchai/internal/config"
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/internal/integration"
	assistant "github.com/mtariq99/dispatchai/internal/orchestration"
	"github.com/mtariq99/dispatchai/internal/policy"
	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/internal/run"
	"github.com/mtariq99/dispatchai/internal/tools"
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

	runStore := storage.NewRunStore(db)
	runService := run.NewRunService(runStore)

	conversationStore, err := storage.NewConversationStore(db)
	if err != nil {
		log.Fatal(err)
	}

	pc, err := project.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	toolGateway := project.NewProjectToolGateway(pc)
	policyEngine := policy.NewEngine(policy.MaxNotificationsPerRun(1))

	toolIdempotency := idempotency.NewToolIdempotency(db)
	toolRegistryStore := storage.NewToolRegistry(db)
	toolRegistryService, err := tools.NewRegistrationService(toolRegistryStore)
	if err != nil {
		log.Fatalf("tool registry service: %v", err)
	}

	toolRegistryHandler, err := tools.NewToolRegistryHandler(cfg, toolRegistryService)
	if err != nil {
		log.Fatalf("tool registry handler: %v", err)
	}

	executor := tools.NewExecutor(toolRegistryService, toolGateway, policyEngine, toolIdempotency)

	conversationService, err := conversation.NewConversationService(conversationStore)
	if err != nil {
		log.Fatal(err)
	}

	orchestrator := assistant.NewOrchestrator(
		geminiClient,
		toolRegistryService,
		executor,
		cfg.LLM.Model,
		cfg.LLM.MaxTokens,
		cfg.LLM.Temperature,
		conversationService,
	)

	assistantService := assistant.NewAssistant(orchestrator)

	requestIdempotencyStore := storage.NewIdempotencyStore(db)
	idempotencyService, err := idempotency.NewIdempotencyService(requestIdempotencyStore)
	if err != nil {
		log.Fatal(err)
	}

	conversationHandler, err := conversation.NewConversationHandler(
		cfg,
		conversationService,
		assistantService,
		runService,
		idempotencyService,
	)
	if err != nil {
		log.Fatalf("conversation handler: %v", err)
	}

	integrationStore := storage.NewIntegrationOnboardingStore(db)
	enrollmentStore := storage.NewIntegrationEnrollmentStore(db)

	onboardingService, err := integration.NewOnboardingService(
		integrationStore,
		enrollmentStore,
	)
	if err != nil {
		log.Fatalf("integration onboarding service: %v", err)
	}

	adminService, err := integration.NewAdminService(integrationStore)
	if err != nil {
		log.Fatalf("integration admin service: %v", err)
	}

	onboardingHandler := integration.NewIntegrationOnboardingHandler(
		onboardingService,
		adminService,
	)

	assistantHandler, err := api.NewAssistantHandler(cfg, toolRegistryHandler, onboardingHandler, conversationHandler)
	if err != nil {
		log.Fatalf("assistant handler: %v", err)
	}

	router := gin.Default()

	signedAuthStore := storage.NewSignedAuthStore(db)
	requestVerifier, err := authn.NewRequestVerifier(signedAuthStore, signedAuthStore, authn.VerifierConfig{})
	if err != nil {
		log.Fatalf("request verifier: %v", err)
	}
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go signedAuthStore.RunNonceCleanup(cleanupCtx, time.Minute, logger)

	assistantHandler.RegisterRoutes(router, requestVerifier, cfg.App.AdminToken, integrationStore)
	fmt.Println("DispatchAI server is listening on :8989")

	if err := router.Run(":8989"); err != nil {
		log.Fatal(err)
	}
}
