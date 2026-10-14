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

const maxOnboardingBodyBytes = 1 << 20 // 1 MiB

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
