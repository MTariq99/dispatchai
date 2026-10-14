package integration

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

const (
	maxOnboardingBodyBytes = 1 << 20 // 1 MiB
	// The real payload is under 200 bytes; this endpoint is unauthenticated.
	maxEnrollmentBodyBytes = 64 << 10 // 64 KiB
)

type IntegrationOnboardingHandler struct {
	service    *OnboardingService
	adminToken string
}

func NewIntegrationOnboardingHandler(service *OnboardingService, adminToken string) *IntegrationOnboardingHandler {
	return &IntegrationOnboardingHandler{
		service:    service,
		adminToken: strings.TrimSpace(adminToken),
	}
}

func (h *IntegrationOnboardingHandler) Create(c *gin.Context) {
	if h == nil || h.service == nil || h.adminToken == "" {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "integration onboarding is not configured",
		})
		return
	}

	if !h.authorized(c.GetHeader("Authorization")) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "valid onboarding credentials are required",
		})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxOnboardingBodyBytes)

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

func (h *IntegrationOnboardingHandler) authorized(header string) bool {
	parts := strings.Fields(header)
	if len(parts) != 2 ||
		!strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return false
	}

	provided := parts[1]
	if len(provided) != len(h.adminToken) {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(h.adminToken)) == 1
}

func (h *IntegrationOnboardingHandler) Enroll(c *gin.Context) {
	if h == nil || h.service == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"error": "integration enrollment is not configured",
		})
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxEnrollmentBodyBytes)
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
