package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/internal/integration"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
	"github.com/mtariq99/dispatchai/internal/run"
	"github.com/mtariq99/dispatchai/internal/utils"
	"github.com/mtariq99/dispatchai/models"
)

type RequestAssistant interface {
	HandleRequest(ctx context.Context, userMessage string, execCtx models.ExecutionContext) (string, error)
}
type ConversationHandler struct {
	conversation           *ConversationService
	assistant              RequestAssistant
	runService             *run.RunService
	requestIdempotencyServ *idempotency.IdempotencyService
	AdminService           *integration.AdminService
}

func NewConversationHandler(cfg *models.Config, conversationSrv *ConversationService, assistant RequestAssistant, runService *run.RunService, requestIdempotencyServ *idempotency.IdempotencyService, AdminServ *integration.AdminService) (*ConversationHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}

	if conversationSrv == nil {
		return nil, fmt.Errorf("conversation service  is nil")
	}

	if assistant == nil {
		return nil, fmt.Errorf("assistant is nil")
	}

	if runService == nil {
		return nil, fmt.Errorf("run service is nil")
	}

	if requestIdempotencyServ == nil {
		return nil, fmt.Errorf("request idempotency service is nil")
	}

	return &ConversationHandler{
		conversation:           conversationSrv,
		assistant:              assistant,
		runService:             runService,
		requestIdempotencyServ: requestIdempotencyServ,
		AdminService:           AdminServ,
	}, nil
}

func (ch *ConversationHandler) CreateConversation(c *gin.Context) {
	integrationIDStr, ok := utils.AuthenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}
	ctx := c.Request.Context()
	if integrationIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration id is required"})
		return
	}
	integrationID, err := utils.ParseUUID(integrationIDStr, "integration id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	var req models.CreateConversationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation name is required"})
		return
	}

	resp, err := ch.conversation.CreateConversation(ctx, integrationID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (ch *ConversationHandler) GetConversation(c *gin.Context) {
	integrationIDStr, ok := utils.AuthenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}
	integrationID, err := utils.ParseUUID(integrationIDStr, "integration id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	ctx := c.Request.Context()

	conversationIDStr := c.Param("conversation_id")
	if conversationIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation ID is required"})
		return
	}
	conversationID, err := utils.ParseUUID(conversationIDStr, "conversation id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	converstion, err := ch.conversation.GetConversation(ctx, integrationID, conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, converstion)
}

func (ch *ConversationHandler) Chat(c *gin.Context) {
	integrationIDStr, ok := integrationctx.Get(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "integration authentication required"})
		return
	}
	ctx := c.Request.Context()
	integrationID, err := utils.ParseUUID(integrationIDStr, "integration id")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var req models.ChatRequest
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	clientKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if clientKey == "" || len(clientKey) > 200 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "valid Idempotency-Key header is required"})
		return
	}
	idempotencyKey := integrationIDStr + ":" + clientKey // isolates integrations

	fingerprint, err := utils.RequestFingerprint(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to fingerprint request"})
		return
	}
	if req.ConversationID == uuid.Nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "failed to fingerprint request"})
		return
	}
	if _, err := ch.conversation.GetConversation(ctx, integrationID, req.ConversationID); err != nil {
		if errors.Is(err, errs.ErrConversationNotFound) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
			return
		} else {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
			return
		}
	}

	requestID := uuid.New()

	state, found, err := ch.requestIdempotencyServ.GetState(ctx, integrationID, idempotencyKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to check request idempotency " + err.Error(),
		})
		return
	}

	if found {
		if state.Fingerprint == "" || state.Fingerprint != fingerprint {
			c.JSON(http.StatusConflict, models.ErrorResponse{
				Error: "Idempotency-Key was already used for a different or unverifiable request",
			})
			return
		}
		switch state.Status {
		case models.RequestStatusCompleted:
			if state.Response == nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error: "completed idempotency record has no response",
				})
				return
			}

			c.JSON(http.StatusOK, state.Response)
			return

		case models.RequestStatusFailed:
			if state.Response == nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error: "failed idempotency record has no response",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, state.Response)
			return

		case models.RequestStatusPending:
			recoveredRun, lookupErr := ch.runService.GetByRequestID(ctx, state.RequestID)
			if lookupErr == nil && recoveredRun.IntegrationID == integrationID && recoveredRun.ConversationID == state.ConversationID && recoveredRun.RequestID == state.RequestID && recoveredRun.Status == run.StateCompleted && recoveredRun.Answer != "" && (state.RunID == nil || *state.RunID == recoveredRun.ID) {

				response := &models.ChatResponse{
					RequestID:      recoveredRun.RequestID.String(),
					ConversationID: recoveredRun.ConversationID,
					RunID:          recoveredRun.ID,
					Status:         string(recoveredRun.Status),
					Answer:         recoveredRun.Answer,
				}

				if err := ch.requestIdempotencyServ.CompleteReq(ctx, integrationID, idempotencyKey, state.RequestID, recoveredRun.ID, response); err != nil {
					c.JSON(http.StatusInternalServerError, models.ErrorResponse{
						Error: "completed run found but request idempotency reconciliation failed",
					})
					return
				}

				c.JSON(http.StatusOK, response)
				return
			}

			if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Error: "failed to inspect pending request run",
				})
				return
			}

			response := &models.ChatResponse{
				RequestID:      state.RequestID.String(),
				ConversationID: state.ConversationID,
				Status:         string(models.RequestStatusPending),
			}

			if state.RunID != nil {
				response.RunID = *state.RunID
			}

			c.JSON(http.StatusConflict, response)
			return

		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "unknown request idempotency state",
			})
			return
		}
	}

	reserved, err := ch.requestIdempotencyServ.ReserveReq(ctx, integrationID, idempotencyKey, req.ConversationID, requestID, fingerprint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}
	if !reserved {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "request with this Idempotency-Key is already being processed",
		})
		return
	}

	runRecord, err := ch.runService.Create(ctx, integrationID, req.ConversationID, requestID, time.Now().UTC())
	if err != nil {
		failedResponse := &models.ChatResponse{
			RequestID:      requestID.String(),
			ConversationID: req.ConversationID,
			Status:         string(run.StateFailed),
		}

		if idemErr := ch.requestIdempotencyServ.FailBeforeRun(ctx, integrationID, idempotencyKey, requestID, failedResponse); idemErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "failed to create run and failed to persist request failure",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, failedResponse)
		return
	}
	if err := ch.runService.Start(ctx, runRecord, time.Now().UTC()); err != nil {
		runRecord.ErrorCode = errs.RunErrorCodeStartFailed
		runRecord.ErrorMessage = err.Error()

		if failErr := ch.runService.Fail(ctx, runRecord, time.Now().UTC()); failErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "run start failed and failed to persist run failure",
			})
			return
		}

		failedResponse := &models.ChatResponse{
			RequestID:      requestID.String(),
			ConversationID: req.ConversationID,
			RunID:          runRecord.ID,
			Status:         string(runRecord.Status),
		}

		if idemErr := ch.requestIdempotencyServ.Fail(ctx, integrationID, idempotencyKey, requestID, runRecord.ID, failedResponse); idemErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "run failed but failed to persist request result",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, failedResponse)
		return
	}
	perms, err := ch.AdminService.GrantedPermissions(ctx, integrationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to load integration permissions"})
		return
	}
	if perms == nil {
		perms = []string{}
	}

	execCtx := models.ExecutionContext{
		RunID:          runRecord.ID,
		RequestID:      requestID.String(),
		ConversationID: req.ConversationID,
		IntegrationID:  integrationID,
		Authorization:  models.AuthorizationContext{Roles: []string{}, Permissions: perms},
	}

	answer, err := ch.assistant.HandleRequest(ctx, req.Message, execCtx)
	if err != nil {
		runRecord.ErrorCode = errs.RunErrorCodeAssistant
		runRecord.ErrorMessage = err.Error()
		failErr := ch.runService.Fail(ctx, runRecord, time.Now().UTC())
		failedResponse := &models.ChatResponse{
			RequestID:      requestID.String(),
			ConversationID: req.ConversationID,
			RunID:          runRecord.ID,
			Status:         string(runRecord.Status),
		}

		idempotencyErr := ch.requestIdempotencyServ.Fail(ctx, integrationID, idempotencyKey, requestID, runRecord.ID, failedResponse)
		if failErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "assistant execution failed and failed to persist run failure",
			})
			return
		}

		if idempotencyErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "assistant execution failed and failed to persist request result",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, failedResponse)
		return
	}

	runRecord.Answer = answer
	if err := ch.runService.Complete(ctx, runRecord, time.Now().UTC()); err != nil {
		runRecord.ErrorCode = errs.RunErrorCodeComplete
		runRecord.ErrorMessage = err.Error()
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to complete run",
		})
		return
	}

	response := &models.ChatResponse{
		RequestID:      requestID.String(),
		ConversationID: req.ConversationID,
		RunID:          runRecord.ID,
		Status:         string(runRecord.Status),
		Answer:         answer,
	}

	if err := ch.requestIdempotencyServ.CompleteReq(ctx, integrationID, idempotencyKey, requestID, runRecord.ID, response); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to complete request idempotency",
		})
		return
	}
	c.JSON(http.StatusOK, response)
}
