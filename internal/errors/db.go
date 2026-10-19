package errs

import "errors"

// Integration enrollment errors
var (
	ErrInvalidEnrollmentRequest = errors.New("invalid enrollment request")
	ErrEnrollmentTokenInvalid   = errors.New("enrollment token is invalid or expired")
	ErrIntegrationNotEnrollable = errors.New("integration is not enrollable")
	ErrKeyIDConflict            = errors.New("integration key id conflict")

	// Integration onboarding errors
	ErrInvalidOnboardingRequest = errors.New("invalid onboarding request")
	ErrIntegrationNotFound      = errors.New("integration not found")
	ErrInvalidStatusTransition  = errors.New("invalid integration status transition")
	ErrNoActiveKey              = errors.New("integration has no active signing key")
	ErrUnknownPermission        = errors.New("unknown permission")
	ErrClientIDRequired         = errors.New("client id is required")
	ErrGrantedByRequired        = errors.New("granted by is required")
	ErrPermissionRequired       = errors.New("at least one permission is required")
	ErrPermissionEmpty          = errors.New("permission cannot be empty")

	// Integration enrollment database operations
	ErrBeginEnrollmentTransaction  = errors.New("begin integration enrollment transaction")
	ErrConsumeEnrollmentToken      = errors.New("consume enrollment token")
	ErrInsertIntegrationKey        = errors.New("insert integration key")
	ErrCommitEnrollmentTransaction = errors.New("commit integration enrollment transaction")

	// Integration onboarding database operations
	ErrBeginOnboardingTransaction  = errors.New("begin integration onboarding transaction")
	ErrCreatePendingIntegration    = errors.New("create pending integration")
	ErrPersistEnrollmentToken      = errors.New("persist integration enrollment token")
	ErrCommitOnboardingTransaction = errors.New("commit integration onboarding transaction")
	ErrScanHostIntegration         = errors.New("scan host integration")
	ErrLockIntegration             = errors.New("lock integration")
	ErrCheckActiveSigningKeys      = errors.New("check active signing keys")
	ErrUpdateIntegrationStatus     = errors.New("update integration status")
	ErrListHostPermissions         = errors.New("list host permissions")
	ErrScanHostPermission          = errors.New("scan host permission")
	ErrBeginGrantPermissions       = errors.New("begin grant permissions transaction")
	ErrGrantPermission             = errors.New("grant permission")
	ErrCommitPermissionGrants      = errors.New("commit permission grants")
	ErrCheckIntegration            = errors.New("check integration")
	ErrListGrantedPermissions      = errors.New("list granted permissions")
	ErrScanGrantedPermission       = errors.New("scan granted permission")
	ErrMissingPermissionFields     = errors.New("integration id and permission are required")
	ErrCheckIntegrationPermission  = errors.New("check integration permission")
)

var (
	ErrToolNotFound        = errors.New("tool not found")
	ErrVersionConflict     = errors.New("tool definition version conflict")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
	ErrIntegrationInactive = errors.New("integration is not active")
	ErrToolUnavailable     = errors.New("tool is disabled or deleted")
	ErrFieldsRequired      = errors.New("missing required fields")
)

var (
	// Configuration errors
	ErrDatabaseNotConfigured = errors.New("database is not configured")
	ErrDatabaseCannotBeNil   = errors.New("database cannot be nil")

	// Validation errors
	ErrConversationIDRequired     = errors.New("conversation id is required")
	ErrIntegrationIDRequired      = errors.New("integration id is required")
	ErrIdempotencyKeyRequired     = errors.New("idempotency key is required")
	ErrRequestIDRequired          = errors.New("request id is required")
	ErrRunIDRequired              = errors.New("run id is required")
	ErrRequestFingerprintRequired = errors.New("request fingerprint is required")
	ErrResponseRequired           = errors.New("response is required")
	ErrConversationMessageNil     = errors.New("conversation message is nil")

	ErrConversationNotFound = errors.New("conversation not found or does not belong to integration")

	// Idempotency errors
	ErrIdempotencyRecordNotPending = errors.New(
		"request idempotency record is not pending",
	)

	// Database operation errors
	ErrAcquireConnection             = errors.New("acquire database connection")
	ErrLockConversation              = errors.New("lock conversation")
	ErrUnlockConversation            = errors.New("unlock conversation")
	ErrReleaseConnection             = errors.New("release database connection")
	ErrCheckConversationOwnership    = errors.New("check conversation ownership")
	ErrLoadConversationMessages      = errors.New("load conversation messages")
	ErrScanConversationMessage       = errors.New("scan conversation message")
	ErrDecodeToolCalls               = errors.New("decode conversation tool calls")
	ErrIterateConversationMessages   = errors.New("iterate conversation messages")
	ErrBeginConversationTransaction  = errors.New("begin save conversation transaction")
	ErrDeleteConversationMessages    = errors.New("delete previous conversation messages")
	ErrMarshalToolCalls              = errors.New("marshal tool calls")
	ErrInsertConversationMessage     = errors.New("insert conversation message")
	ErrUpdateConversationTimestamp   = errors.New("update conversation timestamp")
	ErrCommitConversationTransaction = errors.New("commit conversation messages transaction")
	ErrCreateConversation            = errors.New("create conversation")
	ErrGetConversation               = errors.New("get conversation")

	// Idempotency persistence errors
	ErrMarshalChatResponse         = errors.New("marshal chat response")
	ErrMarshalFailedChatResponse   = errors.New("marshal failed chat response")
	ErrCompleteRequestIdempotency  = errors.New("complete request idempotency")
	ErrFailRequestIdempotency      = errors.New("fail request idempotency")
	ErrFailRequestBeforeRun        = errors.New("fail request before run")
	ErrReserveRequestIdempotency   = errors.New("reserve request idempotency")
	ErrGetRequestIdempotency       = errors.New("get request idempotency")
	ErrGetRequestIdempotencyState  = errors.New("get request idempotency state")
	ErrUnmarshalStoredChatResponse = errors.New("unmarshal stored chat response")
	ErrGetAffectedRows             = errors.New("get affected rows")
	ErrMarshal                     = errors.New("Error marshalling")
)

var (
	// Signed authentication persistence
	ErrLookupSigningKey   = errors.New("lookup signing key")
	ErrRecordNonce        = errors.New("record request nonce")
	ErrPurgeExpiredNonces = errors.New("purge expired nonces")

	// Run persistence
	ErrCreateRun         = errors.New("create run")
	ErrUpdateRun         = errors.New("update run")
	ErrGetRun            = errors.New("get run")
	ErrGetUpdatedRunRows = errors.New("get updated run rows")
	ErrScanRun           = errors.New("scan run")
	ErrRunNotFound       = errors.New("run not found")

	// Tool registry validation
	ErrToolNameRequired              = errors.New("tool name cannot be empty")
	ErrToolDescriptionRequired       = errors.New("tool description cannot be empty")
	ErrToolNameTooLong               = errors.New("tool name exceeds maximum length")
	ErrToolSchemaRequired            = errors.New("tool parameters schema cannot be nil")
	ErrToolSchemaMustBeObject        = errors.New("tool parameters schema must have type object")
	ErrEncodeToolSchema              = errors.New("encode tool schema")
	ErrDecodeToolSchema              = errors.New("decode tool input schema")
	ErrToolNameConflict              = errors.New("tool name already exists in this integration")
	ErrMissingToolRegistrationFields = errors.New("missing required tool registration fields")

	// Tool registry persistence
	ErrBeginToolRegistration          = errors.New("begin tool registration")
	ErrLoadRegisteredTool             = errors.New("load registered tool")
	ErrCommitRegistrationReplay       = errors.New("commit registration replay")
	ErrCheckRegistrationIdempotency   = errors.New("check registration idempotency")
	ErrLoadToolIntegration            = errors.New("load integration for tool registration")
	ErrInsertToolDefinition           = errors.New("insert tool definition")
	ErrPersistRegistrationIdempotency = errors.New("persist registration idempotency")
	ErrCommitToolRegistration         = errors.New("commit tool registration")
	ErrListEnabledTools               = errors.New("list enabled tool definitions")
	ErrScanToolDefinition             = errors.New("scan tool definition")
	ErrIterateToolDefinitions         = errors.New("iterate tool definitions")
	ErrValidateToolForExecution       = errors.New("validate tool for execution")
	ErrBeginToolMutation              = errors.New("begin tool mutation")
	ErrLoadToolForMutation            = errors.New("load tool for mutation")
	ErrMutateToolDefinition           = errors.New("mutate tool definition")
	ErrCommitToolMutation             = errors.New("commit tool mutation")
	ErrEncodeVersionedToolSchema      = errors.New("encode versioned tool schema")
	ErrPersistToolDefinitionVersion   = errors.New("persist tool definition version")

	// Database initialization and migrations
	ErrDatabaseDSNRequired      = errors.New("database DSN or URL is required")
	ErrEnsureDatabaseExists     = errors.New("ensure database exists")
	ErrOpenPostgresConnection   = errors.New("open postgres connection")
	ErrPingPostgres             = errors.New("ping postgres")
	ErrRunMigrations            = errors.New("run database migrations")
	ErrCreateMigrationDriver    = errors.New("create migration driver")
	ErrCreateMigrationInstance  = errors.New("create migration instance")
	ErrParseDatabaseName        = errors.New("parse database name from DSN")
	ErrDatabaseNameEmpty        = errors.New("database name is empty in DSN")
	ErrCreateDefaultDatabaseDSN = errors.New("create default database DSN")
	ErrConnectDefaultPostgres   = errors.New("connect to default postgres database")
	ErrPingDefaultPostgres      = errors.New("ping default postgres database")
	ErrCheckDatabaseExists      = errors.New("check if database exists")
	ErrCreateDatabase           = errors.New("create database")
)
