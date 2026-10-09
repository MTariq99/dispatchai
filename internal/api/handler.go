package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/mtariq99/dispatchai/internal/idempotency"
	assistant "github.com/mtariq99/dispatchai/internal/orchestration"
	"github.com/mtariq99/dispatchai/internal/run"
	"github.com/mtariq99/dispatchai/models"
)

const (
	runErrorCodeCreateFailed = "RUN_CREATION_FAILED"
	runErrorCodeStartFailed  = "RUN_START_FAILED"
	runErrorCodeAssistant    = "ASSISTANT_EXECUTION_FAILED"
	runErrorCodeComplete     = "RUN_COMPLETION_FAILED"
)

type Handler struct {
	assistant          *assistant.Assistant
	runService         *run.RunService
	requestIdempotency idempotency.RequestIdempotency
}

func NewHandler(cfg *models.Config, assistant *assistant.Assistant, runService *run.RunService, requestIdempotency idempotency.RequestIdempotency) (*Handler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}

	if assistant == nil {
		return nil, fmt.Errorf("assistant is nil")
	}

	if runService == nil {
		return nil, fmt.Errorf("run service is nil")
	}

	if requestIdempotency == nil {
		return nil, fmt.Errorf("request idempotency is nil")
	}

	return &Handler{
		assistant:          assistant,
		runService:         runService,
		requestIdempotency: requestIdempotency,
	}, nil
}

func (h *Handler) Chat(c *gin.Context) {
	var req models.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid request body: " + err.Error(),
		})
		return
	}

	if req.UserID == uuid.Nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "user_id is required",
		})
		return
	}

	if req.TenantID == uuid.Nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "tenant_id is required",
		})
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))

	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "Idempotency-Key header is required",
		})
		return
	}
	fingerprint, err := requestFingerprint(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to fingerprint request",
		})
		return
	}
	if req.ConversationID == uuid.Nil {
		req.ConversationID = uuid.New()
	}

	requestID := uuid.New()
	ctx := c.Request.Context()

	state, found, err := h.requestIdempotency.GetState(ctx, req.TenantID, idempotencyKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to check request idempotency",
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
			// The run may have committed successfully while idempotency
			// completion failed. Recover only from a matching completed run.
			recoveredRun, lookupErr := h.runService.GetByRequestID(ctx, state.RequestID)
			if lookupErr == nil &&
				recoveredRun.TenantID == req.TenantID &&
				recoveredRun.UserID == req.UserID &&
				recoveredRun.ConversationID == state.ConversationID &&
				recoveredRun.RequestID == state.RequestID &&
				recoveredRun.Status == run.StateCompleted &&
				recoveredRun.Answer != "" &&
				(state.RunID == nil || *state.RunID == recoveredRun.ID) {

				response := &models.ChatResponse{
					RequestID:      recoveredRun.RequestID.String(),
					ConversationID: recoveredRun.ConversationID,
					RunID:          recoveredRun.ID,
					Status:         string(recoveredRun.Status),
					Answer:         recoveredRun.Answer,
				}

				if err := h.requestIdempotency.Complete(
					ctx,
					req.TenantID,
					idempotencyKey,
					state.RequestID,
					recoveredRun.ID,
					response,
				); err != nil {
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

	reserved, err := h.requestIdempotency.Reserve(ctx, req.TenantID, idempotencyKey, req.ConversationID, requestID, fingerprint)

	if !reserved {
		// Another request owns this idempotency key.
		//
		// For now we return 409. Later we can make this smarter by
		// returning the existing Run status instead.
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Error: "request with this Idempotency-Key is already being processed",
		})
		return
	}

	runRecord, err := h.runService.Create(
		ctx,
		req.TenantID,
		req.UserID,
		req.ConversationID,
		requestID,
		time.Now().UTC(),
	)
	if err != nil {
		failedResponse := &models.ChatResponse{
			RequestID:      requestID.String(),
			ConversationID: req.ConversationID,
			Status:         string(run.StateFailed),
		}

		if idemErr := h.requestIdempotency.FailBeforeRun(
			ctx,
			req.TenantID,
			idempotencyKey,
			requestID,
			failedResponse,
		); idemErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "failed to create run and failed to persist request failure",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, failedResponse)
		return
	}
	if err := h.runService.Start(ctx, runRecord, time.Now().UTC()); err != nil {
		runRecord.ErrorCode = runErrorCodeStartFailed
		runRecord.ErrorMessage = err.Error()

		if failErr := h.runService.Fail(ctx, runRecord, time.Now().UTC()); failErr != nil {
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

		if idemErr := h.requestIdempotency.Fail(
			ctx,
			req.TenantID,
			idempotencyKey,
			requestID,
			runRecord.ID,
			failedResponse,
		); idemErr != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "run failed but failed to persist request result",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, failedResponse)
		return
	}

	execCtx := models.ExecutionContext{
		RunID:          runRecord.ID,
		RequestID:      requestID.String(),
		ConversationID: req.ConversationID,
		Identity: models.Identity{
			UserID: req.UserID,
		},
		Tenant: models.TenantContext{
			TenantID: req.TenantID,
		},
		Authorization: models.AuthorizationContext{
			Roles:       req.Roles,
			Permissions: req.Permissions,
		},
	}

	answer, err := h.assistant.HandleRequest(ctx, req.Message, execCtx)
	if err != nil {
		runRecord.ErrorCode = runErrorCodeAssistant
		runRecord.ErrorMessage = err.Error()
		failErr := h.runService.Fail(ctx, runRecord, time.Now().UTC())
		failedResponse := &models.ChatResponse{
			RequestID:      requestID.String(),
			ConversationID: req.ConversationID,
			RunID:          runRecord.ID,
			Status:         string(runRecord.Status),
		}

		idempotencyErr := h.requestIdempotency.Fail(ctx, req.TenantID, idempotencyKey, requestID, runRecord.ID, failedResponse)
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

	// Persist the final answer atomically with the completed run status.
	runRecord.Answer = answer
	if err := h.runService.Complete(ctx, runRecord, time.Now().UTC()); err != nil {
		runRecord.ErrorCode = runErrorCodeComplete
		runRecord.ErrorMessage = err.Error()

		// Important:
		//
		// Complete() may have failed because the database update failed.
		// We cannot safely assume whether the database actually committed
		// or not.
		//
		// Do NOT blindly execute the assistant again.
		//
		// The business operation may already have happened.
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to complete run",
		})
		return
	}

	// -------------------------------------------------------------------------
	// 8. Build final API response
	// -------------------------------------------------------------------------

	response := &models.ChatResponse{
		RequestID:      requestID.String(),
		ConversationID: req.ConversationID,
		RunID:          runRecord.ID,
		Status:         string(runRecord.Status),
		Answer:         answer,
	}

	if err := h.requestIdempotency.Complete(ctx, req.TenantID, idempotencyKey, requestID, runRecord.ID, response); err != nil {
		// The Run is already completed.
		//
		// Do NOT execute the assistant again.
		// The correct production behavior here is recovery/reconciliation
		// through idempotency persistence.
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "failed to complete request idempotency",
		})
		return
	}

	// -------------------------------------------------------------------------
	// 10. Return final response
	// -------------------------------------------------------------------------

	c.JSON(http.StatusOK, response)
}
