# DispatchAI

**DispatchAI** is a domain-agnostic AI assistant and orchestration
engine written in Go. It helps existing applications add
natural-language interfaces to their capabilities without moving
business logic, authorization, or transactional data into the AI layer.

> **DispatchAI understands, reasons, and orchestrates. The host
> application owns business truth, authorization, and execution.**

DispatchAI is designed to integrate with systems such as e-commerce,
fleet management, logistics, field service, customer support, and
internal enterprise applications.

## Table of contents

- [DispatchAI](#dispatchai)
  - [Table of contents](#table-of-contents)
  - [Core design](#core-design)
  - [What DispatchAI does](#what-dispatchai-does)
  - [Architecture](#architecture)
  - [Request and tool-execution flow](#request-and-tool-execution-flow)
  - [Security and responsibility boundaries](#security-and-responsibility-boundaries)
    - [DispatchAI is responsible for](#dispatchai-is-responsible-for)
    - [The host application is responsible for](#the-host-application-is-responsible-for)
  - [Repository layout](#repository-layout)
  - [Technology and adapters](#technology-and-adapters)
    - [LLM providers](#llm-providers)
    - [Storage](#storage)
    - [Knowledge](#knowledge)
    - [Notifications](#notifications)
  - [Persistence and reliability](#persistence-and-reliability)
  - [API integration model](#api-integration-model)
  - [Configuration and running](#configuration-and-running)
    - [Prerequisites](#prerequisites)
    - [Download dependencies and run checks](#download-dependencies-and-run-checks)
  - [Testing and quality checks](#testing-and-quality-checks)
  - [Operational considerations](#operational-considerations)
  - [Current scope and limitations](#current-scope-and-limitations)
  - [Guiding principle](#guiding-principle)

## Core design

A conventional application handles requests through its own business
services:

``` text
User → Host API → Business services → Database / external systems
```

DispatchAI adds an AI orchestration layer to that flow:

``` text
User
  │
  ▼
Host application
  │ authenticated request + execution context
  ▼
DispatchAI API
  │
  ▼
Assistant / orchestration loop
  │
  ▼
LLM provider
  │ structured tool call, if needed
  ▼
DispatchAI tool executor
  │ policy checks + idempotency handling
  ▼
Host application's tool gateway
  │
  ▼
Host business services → Database / external systems
  │
  ▼
Structured tool result → DispatchAI → LLM → Final response
```

The LLM is not a source of business truth. It can propose a tool call,
but the host application remains responsible for validating the request,
enforcing authorization, applying business rules, and executing the
operation.

## What DispatchAI does

The repository provides the building blocks for:

-   **LLM integration** through provider adapters for Gemini, OpenAI,
    and Anthropic.
-   **Tool orchestration** using generic tool definitions, a registry,
    policy checks, an execution gateway, and structured results.
-   **Conversation handling** with persisted message history and
    per-conversation locking to serialize concurrent requests.
-   **Request idempotency** to track request state and return persisted
    results for completed requests.
-   **Run lifecycle tracking** for created, running, waiting, completed,
    failed, and cancelled execution states, subject to supported
    transitions.
-   **Knowledge integration** through a provider abstraction and a
    pgvector adapter.
-   **Notification adapters** for email, push, SMS, and webhooks.
-   **Operational building blocks** for policy decisions, approvals,
    audit events, and observability.

These are reusable backend components, not a claim that every
deployment-specific workflow or operational policy is fully configured
out of the box. The host application must still implement and enforce
its own domain-specific controls.

## Architecture

``` text
┌──────────────────────── HOST APPLICATION ────────────────────────┐
│                                                                  │
│  Authentication · Identity · Tenant context · Authorization      │
│  Domain services · Business rules · Transactional data           │
│  Host tool gateway · External business integrations              │
│                                                                  │
└─────────────────────────────┬────────────────────────────────────┘
                              │ request / execution context
                              ▼
┌────────────────────────────── DISPATCHAI ─────────────────────────┐
│                                                                   │
│  API and request lifecycle                                        │
│       │                                                           │
│       ├── Request idempotency                                      │
│       ├── Run lifecycle                                            │
│       └── Conversation persistence and locking                     │
│                                                                   │
│  Assistant / orchestration                                         │
│       │                                                           │
│       ├── LLM abstraction ───────────► Provider adapters            │
│       ├── Tool registry and policy                                  │
│       ├── Tool executor ─────────────► Host tool gateway            │
│       ├── Knowledge abstraction ────► Knowledge adapter             │
│       └── Notification abstractions ► Notification adapters         │
│                                                                   │
│  Storage adapters · Audit abstractions · Observability              │
└───────────────────────────────────────────────────────────────────┘
```

The core is intended to depend on interfaces rather than vendor-specific
implementations. Adapters connect those interfaces to concrete providers
and infrastructure.

## Request and tool-execution flow

1.  The host application authenticates the user and supplies trusted
    identity and tenant context.
2.  DispatchAI validates the request and handles its idempotency key.
3.  DispatchAI loads the conversation under a conversation-level lock.
4.  The assistant sends the current context and available tool
    definitions to the configured LLM provider.
5.  If the model requests a tool, DispatchAI validates and routes the
    call through its tool infrastructure.
6.  Policy and idempotency checks run before the tool gateway is
    invoked.
7.  The host application's gateway performs authoritative authorization,
    argument validation, business-rule checks, and execution.
8.  DispatchAI passes the structured result back into the orchestration
    loop.
9.  The final answer, conversation state, run state, and idempotency
    response are persisted according to the request path.

Tool calls are untrusted proposals from the model. A model response must
never be treated as proof that a user is authorized to perform an
operation.

## Security and responsibility boundaries

### DispatchAI is responsible for

-   LLM request/response orchestration
-   Generic tool-call routing and execution coordination
-   Conversation and request lifecycle management
-   Idempotency coordination
-   Policy abstractions and approval-related building blocks
-   Optional knowledge retrieval integration
-   Provider and storage abstractions
-   Structured results and operational instrumentation

### The host application is responsible for

-   Authentication and trusted identity propagation
-   Tenant isolation and authoritative authorization
-   Domain-specific permissions and business rules
-   Validating tool arguments against business requirements
-   Business databases and transactional consistency
-   Execution of business operations and external business APIs
-   Business-specific notification policies
-   Deciding which capabilities are exposed to the AI

**Never rely on the LLM to authorize an action.** The host tool endpoint
must independently authenticate or verify the caller/service identity,
validate tenant context, enforce permissions, and reject unauthorized
requests.

Avoid placing secrets, credentials, sensitive personal data, or
unrestricted business records in prompts and logs. Tool schemas should
expose only the minimum capabilities and fields required.

## Repository layout

The repository is organized around core interfaces, infrastructure
adapters, contracts, migrations, and examples. The exact tree may evolve
as implementation changes.

``` text
.
├── adapters/
│   ├── knowledge/
│   │   └── pgvector/
│   ├── llm/
│   │   ├── anthropic/
│   │   ├── gemini/
│   │   └── openai/
│   ├── notification/
│   │   ├── email/
│   │   ├── push/
│   │   ├── sms/
│   │   └── webhook/
│   └── storage/
│       ├── postgres/
│       └── redis/
├── cmd/
│   └── assistant/
├── contracts/
├── deployments/
├── examples/
│   └── ecommerce/
├── internal/
│   ├── api/
│   ├── approval/
│   ├── audit/
│   ├── config/
│   ├── conversation/
│   ├── idempotency/
│   ├── knowledge/
│   ├── llm/
│   ├── notification/
│   ├── observability/
│   ├── orchestration/
│   ├── policy/
│   ├── project/
│   ├── run/
│   └── tools/
├── migrations/
├── models/
├── Makefile
├── go.mod
└── README.md
```

Use the actual checked-out tree as the source of truth if package names
or directories differ from this overview.

## Technology and adapters

### LLM providers

Provider implementations live under `adapters/llm/` and integrate with
the core LLM abstraction.

-   Gemini
-   OpenAI
-   Anthropic

Configure credentials and provider selection through the application's
configuration. Never commit API keys or place them in example
configuration files.

### Storage

Storage adapters include PostgreSQL and Redis. PostgreSQL is used for
durable application records such as conversations, runs, and idempotency
state; Redis is available for supported ephemeral or fast-access storage
use cases.

The exact set of records persisted by each adapter is defined by its
implementation and database migrations.

### Knowledge

The knowledge abstraction supports pluggable retrieval providers,
including a pgvector adapter. Knowledge retrieval is appropriate for
relatively stable content such as policies, SOPs, manuals, and internal
documentation.

For live transactional data---such as current order status, inventory,
account balances, or vehicle location---prefer an authorized
host-project tool rather than treating retrieved documents as current
truth.

### Notifications

Notification adapters are organized by delivery channel: email, push,
SMS, and webhook. Their availability does not replace host-owned
recipient authorization, consent, rate limits, or business notification
policies.

## Persistence and reliability

DispatchAI tracks runs and request idempotency separately. These records
support lifecycle visibility and safe handling of repeated requests, but
they do not automatically make arbitrary external side effects
exactly-once.

Important design considerations:

-   Use a stable idempotency key for retries of the same logical
    request.
-   Bind an idempotency key to the original request fingerprint; reject
    reuse for a different request.
-   Persist run and request outcomes before reporting durable
    completion.
-   Do not automatically rerun a potentially completed business
    operation merely because response persistence failed.
-   Ensure host tools implement their own idempotency for side effects
    where duplicate execution would be harmful.
-   Treat ambiguous database/network failures as ambiguous until durable
    state can be inspected.
-   Set request, provider, tool, and database timeouts appropriate to
    the deployment.
-   Plan database connection capacity carefully when conversation locks
    are held across LLM and tool operations.

Conversation locking serializes requests for the same conversation. It
does not replace tenant authorization or host-side transactional
controls.

## API integration model

A host application can expose a generic tool endpoint, for example:

``` http
POST /api/v1/ai/tools/execute
```

The precise request and response fields must match the contracts
implemented by the checked-out version of DispatchAI and the host
integration.

A conceptual tool request might look like:

``` json
{
  "request_id": "request-uuid",
  "conversation_id": "conversation-uuid",
  "tool_name": "get_customer_orders",
  "call_id": "tool-call-id",
  "arguments": {
    "customer_id": "customer-123"
  }
}
```

The host gateway should map registered tool names to existing domain
services rather than embedding business rules in DispatchAI:

``` text
Host tool endpoint
    ↓
Authenticate service request and establish trusted context
    ↓
Resolve registered tool
    ↓
Validate arguments and authorize the user/tenant
    ↓
Apply business rules
    ↓
Call existing service/repository
    ↓
Return a structured tool result
```

Do not trust user IDs, tenant IDs, roles, or permissions merely because
they appear in a JSON request body. Establish or verify this context
through a trusted authentication mechanism between the host application
and DispatchAI.

## Configuration and running

This repository is a Go backend. Use the configuration and migration
mechanisms provided by the checked-out version; required environment
variables, service ports, and provider settings should be confirmed from
the configuration package, deployment files, and example environment
file rather than guessed from this overview.

### Prerequisites

-   A Go version compatible with `go.mod`
-   PostgreSQL configured for the project's migrations
-   Redis if the configured deployment uses it
-   Credentials for the selected LLM provider
-   A reachable host-project tool gateway when using external business
    tools

### Download dependencies and run checks

``` bash
go mod download
go test ./... -count=1
go vet ./...
git diff --check
```

Apply database migrations using the repository's existing migration
runner before starting a deployment that depends on schema changes.

Do not commit `.env` files, provider keys, database credentials, or
production secrets. Use your deployment platform's secret manager or
protected environment configuration.

## Testing and quality checks

The repository uses Go's test tooling. Tests should cover both ordinary
results and failure/retry behavior.

``` bash
go test ./... -count=1
go vet ./...
git diff --check
```

Prioritize tests for:

-   Tool registry validation and unknown tools
-   Executor configuration and malformed tool calls
-   Policy denial and non-retryable failures
-   Idempotency key reuse with a different request fingerprint
-   Duplicate requests and recovery after persistence errors
-   Conversation ownership and concurrent requests
-   Run state transitions and persistence
-   Provider timeouts, rate limits, and malformed responses
-   Host gateway errors and duplicate side-effect prevention

Unit tests alone do not prove database migrations, multi-instance
locking, or end-to-end recovery behavior. Run integration tests against
the actual supported infrastructure for those guarantees.

## Operational considerations

Before deploying to production, configure and verify:

-   Authentication and service-to-service trust
-   TLS and network restrictions
-   Tenant isolation
-   LLM provider quotas, rate limits, and timeouts
-   Database migrations, backups, and connection-pool sizing
-   Redis availability and capacity, if used
-   Structured logs, metrics, and distributed tracing
-   Redaction of prompts, tool arguments, and provider errors where they
    may contain sensitive data
-   Limits on tool calls, request size, tokens, and execution time
-   Host-side idempotency for irreversible operations
-   Recovery procedures for ambiguous or partially persisted outcomes
-   Load and failure tests under expected concurrency

Do not describe a deployment as production-ready solely because
`go test ./...` passes. Production readiness also depends on database
integration tests, operational configuration, security review, and
failure recovery.

## Current scope and limitations

DispatchAI is an evolving orchestration backend. The presence of a
package or abstraction does not mean every end-to-end capability is
automatically enabled or production-complete. In particular:

-   Human approval workflows require complete lifecycle wiring, durable
    state, and authorization checks before being relied on for sensitive
    actions.
-   Audit and observability abstractions require deployment-specific
    exporters, retention, alerting, and redaction policies.
-   Knowledge retrieval quality depends on ingestion, chunking, access
    control, retrieval configuration, and evaluation.
-   Request idempotency inside DispatchAI cannot alone guarantee
    exactly-once behavior across arbitrary external systems.
-   Durable workflow orchestration is a separate capability and should
    not be assumed unless it is explicitly integrated and configured.

Check the implementation, migrations, tests, and deployment
configuration for the authoritative status of any feature before
depending on it.

## Guiding principle

``` text
DISPATCHAI
  Understand intent
  Reason and plan
  Orchestrate LLM and tool interactions
  Manage request and conversation lifecycle
  Return a grounded response

HOST APPLICATION
  Authenticate and authorize
  Enforce tenant boundaries
  Apply business rules
  Execute operations
  Own transactional data and business truth
```

**DispatchAI does not own the business. It orchestrates the capabilities
that the host application deliberately exposes.**