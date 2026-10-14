package integration

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

type IntegrationOnboardingHandler struct {
	service *OnboardingService
}

func NewIntegrationOnboardingHandler(service *OnboardingService) *IntegrationOnboardingHandler {
	return &IntegrationOnboardingHandler{service: service}
}

func (h *IntegrationOnboardingHandler) Create(c *gin.Context) {
	if h == nil || h.service == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "integration onboarding is not configured",
		})
		return
	}

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()

	var request models.CreateIntegrationRequest
	if err := decoder.Decode(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid onboarding request",
		})
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "request must contain exactly one JSON object",
		})
		return
	}

	response, err := h.service.CreatePending(c.Request.Context(), request)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidOnboardingRequest) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid onboarding request",
			})
			return
		}

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "could not create integration",
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *IntegrationOnboardingHandler) Enroll(c *gin.Context) {
	if h == nil || h.service == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "integration enrollment is not configured",
		})
		return
	}
	c.Header("Cache-Control", "no-store")

	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var request models.EnrollIntegrationRequest
	if err := decoder.Decode(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment request"})
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment request"})
		return
	}

	response, err := h.service.Enroll(c.Request.Context(), &request)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidEnrollmentRequest),
			errors.Is(err, errs.ErrInvalidPublicKey):
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment request"})

		case errors.Is(err, errs.ErrEnrollmentTokenInvalid),
			errors.Is(err, errs.ErrIntegrationNotEnrollable):
			// Deliberately identical bodies: no oracle for token or integration state.
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "enrollment token is invalid, expired, or already used",
			})

		case errors.Is(err, errs.ErrKeyIDConflict):
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "key_id already exists"})

		default:
			// Never echo err.Error(): it may contain database details.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not complete enrollment"})
		}
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *IntegrationOnboardingHandler) ToggleIntegration(c *gin.Context) {
	if h == nil || h.service == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "integration onboarding is not configured",
		})
		return
	}
	clientID := c.Param("client_id")
	if clientID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "client ID is required",
		})
		return
	}
	var req *models.ToggleIntegrationReq
	if err := c.ShouldBindJSON(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	resp, err := h.service.ToggleIntegration(c.Request.Context(), clientID, req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": err,
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
