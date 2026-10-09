package api

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

func fingerprintTestRequest() models.ChatRequest {
	return models.ChatRequest{
		Message:        "Where is my order?",
		UserID:         uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		TenantID:       uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		ConversationID: uuid.Nil,
		Roles:          []string{"member", "support"},
		Permissions:    []string{"orders:read", "orders:write"},
	}
}

func TestRequestFingerprintIsStable(t *testing.T) {
	req := fingerprintTestRequest()

	first, err := requestFingerprint(req)
	if err != nil {
		t.Fatal(err)
	}

	req.Roles = []string{"support", "member"}
	req.Permissions = []string{"orders:write", "orders:read"}

	second, err := requestFingerprint(req)
	if err != nil {
		t.Fatal(err)
	}

	if first != second {
		t.Fatal("same request with reordered authorization lists produced different fingerprints")
	}
}

func TestRequestFingerprintChangesWhenRequestChanges(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*models.ChatRequest)
	}{
		{
			name: "message",
			mutate: func(req *models.ChatRequest) {
				req.Message = "Cancel my order."
			},
		},
		{
			name: "user",
			mutate: func(req *models.ChatRequest) {
				req.UserID = uuid.New()
			},
		},
		{
			name: "tenant",
			mutate: func(req *models.ChatRequest) {
				req.TenantID = uuid.New()
			},
		},
		{
			name: "conversation",
			mutate: func(req *models.ChatRequest) {
				req.ConversationID = uuid.New()
			},
		},
		{
			name: "roles",
			mutate: func(req *models.ChatRequest) {
				req.Roles = []string{"admin"}
			},
		},
		{
			name: "permissions",
			mutate: func(req *models.ChatRequest) {
				req.Permissions = []string{"orders:delete"}
			},
		},
	}

	original, err := requestFingerprint(fingerprintTestRequest())
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fingerprintTestRequest()
			tt.mutate(&req)

			got, err := requestFingerprint(req)
			if err != nil {
				t.Fatal(err)
			}
			if got == original {
				t.Fatal("changed request produced the same fingerprint")
			}
		})
	}
}
