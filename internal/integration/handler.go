package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

type IntegrationOnboardingHandler struct {
	service *OnboardingService // onboarding + enrollment
	admin   *AdminService      // lifecycle + permissions
}

func NewIntegrationOnboardingHandler(service *OnboardingService, admin *AdminService) *IntegrationOnboardingHandler {
	return &IntegrationOnboardingHandler{service: service, admin: admin}
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
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid onboarding request"})
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "request must contain exactly one JSON object"})
		return
	}

	response, err := h.service.CreatePending(c.Request.Context(), request)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidOnboardingRequest) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid onboarding request"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not create integration"})
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

func decodeStrict(c *gin.Context, dst any) bool {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return false
	}
	var extra any
	return errors.Is(dec.Decode(&extra), io.EOF)
}

func writeAdminError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrIntegrationNotFound):
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "integration not found"})
	case errors.Is(err, errs.ErrInvalidStatusTransition):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "invalid status transition"})
	case errors.Is(err, errs.ErrNoActiveKey):
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "integration has no active signing key"})
	case errors.Is(err, errs.ErrUnknownPermission), errors.Is(err, errs.ErrInvalidOnboardingRequest):
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	default:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
	}
}

func (h *IntegrationOnboardingHandler) adminReady(c *gin.Context) bool {
	if h == nil || h.admin == nil {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "integration admin is not configured"})
		return false
	}
	return true
}

func (h *IntegrationOnboardingHandler) ToggleIntegration(c *gin.Context) {
	if !h.adminReady(c) {
		return
	}
	var req models.ToggleIntegrationReq
	if !decodeStrict(c, &req) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.admin.SetStatus(c.Request.Context(), c.Param("client_id"), &req); err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": req.Status})
}

// GetPermissions lists the permission catalog.
func (h *IntegrationOnboardingHandler) GetPermissions(c *gin.Context) {
	if !h.adminReady(c) {
		return
	}
	perms, err := h.admin.PermissionCatalog(c.Request.Context())
	if err != nil {
		fmt.Println("err>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>", err)
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

// GetGrantedPermissions lists what one integration has been granted.
func (h *IntegrationOnboardingHandler) GetGrantedPermissions(c *gin.Context) {
	if !h.adminReady(c) {
		return
	}
	perms, err := h.admin.GrantedPermissions(c.Request.Context(), c.Param("client_id"))
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permissions": perms})
}

func (h *IntegrationOnboardingHandler) UpsertGrantedPermission(c *gin.Context) {
	if !h.adminReady(c) {
		return
	}
	actor := c.GetString(enums.AdminActorKey)
	if actor == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin identity is missing"})
		return
	}
	var req models.GrantPermissionsReq
	if !decodeStrict(c, &req) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	resp, err := h.admin.GrantPermissions(c.Request.Context(), c.Param("client_id"), actor, &req)
	if err != nil {
		writeAdminError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
