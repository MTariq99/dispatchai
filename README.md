# DispatchAI

**DispatchAI** is a domain-agnostic, production-oriented AI assistant and orchestration engine written in Go.

It allows existing applications to add natural-language AI capabilities without giving the AI system ownership of the application's business logic, database, authorization, or operational services.

DispatchAI is designed around one fundamental boundary:

> **DispatchAI understands, reasons, plans, and orchestrates.
> The host project owns the business, data, authorization, and execution.**

This makes DispatchAI reusable across different domains such as:

* E-commerce
* Fleet management
* Field service
* Healthcare
* Logistics
* IoT
* Internal enterprise systems
* Customer support
* Operations platforms

The host application exposes the capabilities it wants the AI to use. DispatchAI discovers and uses those capabilities through a generic tool contract.

---

# What Is DispatchAI?

A traditional application usually works like this:

```text
User
  │
  ▼
API
  │
  ▼
Business Logic
  │
  ▼
Database / External Services
```

DispatchAI adds an AI reasoning layer:

```text
User
  │
  ▼
Host Project
  │
  │ authenticated request
  ▼
DispatchAI
  │
  ▼
LLM
  │
  │ reasoning + tool calls
  ▼
DispatchAI
  │
  │ tool request
  ▼
Host Project
  │
  ▼
Business Logic
  │
  ▼
Database / External Services
  │
  │ tool result
  ▼
DispatchAI
  │
  ▼
LLM
  │
  ▼
Final Response
  │
  ▼
Host Project
  │
  ▼
User
```

The important distinction is that **the LLM never becomes the source of business truth**.

For example, if a user asks:

> "Show me the customers who haven't placed an order in the last 30 days."

The LLM should not directly query the application's PostgreSQL database.

Instead, DispatchAI may produce a structured tool call such as:

```json
{
  "name": "get_inactive_customers",
  "arguments": {
    "days": 30
  }
}
```

The host project receives that request and decides:

1. Is the user authenticated?
2. Does the user have permission?
3. Is this tool available?
4. Are the arguments valid?
5. What business rules apply?
6. Which service should execute it?
7. Which database/query should be used?

The host project then returns the result to DispatchAI.

---

# Core Responsibility Boundary

DispatchAI owns:

* Natural-language understanding
* LLM interaction
* Reasoning orchestration
* Tool-call orchestration
* Conversation context
* Planning
* Optional knowledge retrieval
* Optional memory
* Policy abstractions
* Approval abstractions
* Idempotency abstractions
* Audit abstractions
* Observability
* Notification orchestration
* LLM provider abstraction
* Knowledge provider abstraction
* Storage abstractions

The host project owns:

* Authentication
* User identity
* Tenant identity
* Authorization
* Business rules
* Business services
* Business databases
* Domain models
* External business APIs
* Actual business-side tool execution
* Business-side notification policies
* Business data

### The fundamental rule

```text
                    DISPATCHAI
        ┌─────────────────────────────┐
        │ Understand                  │
        │ Reason                      │
        │ Plan                        │
        │ Orchestrate                 │
        │ Call tools                  │
        │ Process tool results        │
        │ Generate final response     │
        └──────────────┬──────────────┘
                       │
                       │ generic contracts
                       ▼
                ┌───────────────┐
                │ HOST PROJECT  │
                ├───────────────┤
                │ Authenticate  │
                │ Authorize     │
                │ Validate      │
                │ Execute       │
                │ Own data      │
                │ Own business  │
                └───────────────┘
```

DispatchAI should **never bypass this boundary**.

---

# Why DispatchAI?

Without a reusable AI engine, every project tends to implement its own:

```text
LLM client
prompt handling
conversation handling
tool calling
tool loop
memory
RAG
approval
audit
observability
retry handling
provider integration
```

This creates duplicated AI infrastructure across projects.

DispatchAI centralizes these concerns behind reusable interfaces.

A project can therefore integrate AI without rebuilding the entire AI orchestration layer.

---

# Architecture

```text
                         ┌───────────────────────┐
                         │       USER / ADMIN    │
                         └───────────┬───────────┘
                                     │
                                     ▼
                         ┌───────────────────────┐
                         │     HOST PROJECT      │
                         │                       │
                         │ Auth                  │
                         │ Tenant Context        │
                         │ Authorization         │
                         │ Business Logic        │
                         └───────────┬───────────┘
                                     │
                                     │ Request
                                     ▼
                  ┌──────────────────────────────────────┐
                  │              DISPATCHAI               │
                  │                                      │
                  │  Assistant                           │
                  │    │                                 │
                  │    ├── Conversation                  │
                  │    ├── Context                       │
                  │    ├── Planner                       │
                  │    └── Tool Loop                     │
                  │            │                         │
                  │            ▼                         │
                  │           LLM                        │
                  │            │                         │
                  │            ▼                         │
                  │        Tool Calls                    │
                  │            │                         │
                  │            ▼                         │
                  │     Project Tool Gateway             │
                  └────────────┬─────────────────────────┘
                               │
                               │ ToolRequest
                               ▼
                  ┌───────────────────────────────┐
                  │         HOST PROJECT         │
                  │                               │
                  │       Tool Gateway            │
                  │              │                │
                  │       ┌──────┼──────┐         │
                  │       ▼      ▼      ▼         │
                  │    Customer Order  Fleet      │
                  │    Service   Service Service   │
                  │       │      │      │          │
                  │       ▼      ▼      ▼          │
                  │      DB     DB   External API  │
                  └──────────────┬────────────────┘
                                 │
                                 │ ToolResponse
                                 ▼
                           DispatchAI
                                 │
                                 ▼
                                LLM
                                 │
                                 ▼
                           Final Answer
```

---

# Repository Structure

```text
dispatch-ai/
│
├── adapters/
│   ├── knowledge/
│   │   └── pgvector/
│   │
│   ├── llm/
│   │   ├── anthropic/
│   │   ├── gemini/
│   │   └── openai/
│   │
│   ├── notification/
│   │   ├── email/
│   │   ├── push/
│   │   ├── sms/
│   │   └── webhook/
│   │
│   └── storage/
│       ├── postgres/
│       └── redis/
│
├── cmd/
│   ├── assistant/
│   └── example-project/
│
├── contracts/
│   ├── dispatch.proto
│   ├── notification.proto
│   └── tool.proto
│
├── deployments/
│   ├── docker/
│   └── kubernetes/
│
├── examples/
│   ├── ecommerce/
│   ├── field-service/
│   └── fleet/
│
├── internal/
│   ├── api/
│   ├── approval/
│   ├── assistant/
│   ├── audit/
│   ├── config/
│   ├── enums/
│   ├── idempotency/
│   ├── knowledge/
│   ├── llm/
│   ├── memory/
│   ├── notification/
│   ├── observability/
│   ├── policy/
│   ├── project/
│   └── tools/
│
├── models/
│
├── migrations/
│
├── tests/
│   ├── evaluation/
│   ├── failure/
│   ├── integration/
│   ├── load/
│   └── unit/
│
├── Makefile
├── README.md
├── go.mod
└── go.sum
```

---

# Major Components

## `internal/assistant`

The core AI orchestration layer.

It coordinates:

* Conversations
* Context
* Planning
* LLM requests
* Tool calls
* Tool results
* Final responses

The assistant does not contain domain-specific business logic.

For example, it should not contain:

```go
if toolName == "get_customers" {
    // ecommerce logic
}
```

Instead, it treats tools generically.

---

# `internal/llm`

Defines the generic LLM interface.

The assistant depends on this abstraction rather than a specific provider.

Conceptually:

```go
type Client interface {
    Generate(
        ctx context.Context,
        req Request,
    ) (Response, error)
}
```

This allows the same assistant to work with:

* Gemini
* OpenAI
* Anthropic
* Mock LLMs
* Future providers

---

# `adapters/llm`

Contains concrete LLM provider implementations.

```text
adapters/llm/
├── anthropic/
├── gemini/
└── openai/
```

The dependency direction is:

```text
Assistant
    │
    ▼
llm.Client
    │
    ├── Gemini Adapter
    ├── OpenAI Adapter
    └── Anthropic Adapter
```

Provider-specific code stays outside the core assistant.

---

# `internal/tools`

Contains the generic tool infrastructure.

DispatchAI understands a tool as a capability with:

* Name
* Description
* Parameters
* Call
* Result
* Execution mechanism

It does not need to know what the tool actually does.

For example:

```text
get_customers
get_orders
get_vehicle_location
create_work_order
send_notification
```

are all simply capabilities from DispatchAI's perspective.

---

# `internal/project`

This package is the integration boundary between DispatchAI and the host application.

It contains:

```text
client.go
tool_gateway.go
```

The project client communicates with the host project.

The project tool gateway translates generic AI tool calls into host-project tool requests.

For example:

```text
LLM Tool Call
     │
     ▼
ProjectToolGateway
     │
     ▼
ToolRequest
     │
     ▼
ProjectClient
     │
     ▼
Host Project
```

DispatchAI does not know which Go service ultimately executes the tool.

---

# `internal/knowledge`

Provides a generic knowledge abstraction.

Knowledge can be used for:

* Company policies
* SOPs
* Documentation
* Manuals
* Internal knowledge
* Product documentation

RAG is optional.

DispatchAI should not use RAG for live transactional state when that state belongs to the host project.

For example:

```text
"What's our refund policy?"
        │
        ▼
Knowledge / RAG
```

Whereas:

```text
"Which orders are currently delayed?"
        │
        ▼
Host Project Tool
```

Live business state should come from the host project.

---

# `adapters/knowledge/pgvector`

Provides a pgvector implementation of the knowledge abstraction.

Other knowledge providers can be added without changing the assistant.

---

# `internal/memory`

Provides conversation-memory abstractions.

Memory can contain things such as:

* Conversation history
* Previous interactions
* Relevant persistent context

Memory should not be confused with the host project's source of truth.

---

# `internal/policy`

Provides policy and decision abstractions.

This can be used for AI-related policies such as:

* Allowed capabilities
* Tool restrictions
* Approval requirements
* Risk-sensitive actions
* Request constraints

The host project remains responsible for authoritative business authorization.

---

# `internal/approval`

Provides human-in-the-loop abstractions.

For example:

```text
User
 │
 ▼
AI requests high-impact action
 │
 ▼
Approval required
 │
 ▼
Human approval
 │
 ▼
Tool execution
```

Approval becomes particularly useful for actions such as:

* Sending large-scale notifications
* Cancelling orders
* Issuing refunds
* Modifying operational state
* Executing sensitive actions

---

# `internal/idempotency`

Provides idempotency abstractions for operations that may be retried.

This is important for actions such as:

```text
send notification
create order
issue refund
create work order
```

The goal is to prevent duplicate side effects when the same operation is executed more than once.

---

# `internal/audit`

Provides audit-event abstractions.

AI-driven operations should be observable after execution.

For example:

```text
User requested action
        ↓
AI generated tool call
        ↓
Host project authorized action
        ↓
Tool executed
        ↓
Result returned
```

These events can be recorded for operational and compliance purposes.

---

# `internal/notification`

Provides generic notification orchestration.

Supported adapter categories include:

```text
Email
SMS
Push
Webhook
```

The actual ownership of business notification policy remains with the host application.

---

# `internal/observability`

Contains:

* Logging
* Metrics
* Tracing

Production AI systems need visibility into more than HTTP latency.

Important signals include:

* LLM latency
* LLM errors
* Token usage
* Tool-call count
* Tool latency
* Tool failures
* Conversation failures
* Retry counts
* Provider failures
* End-to-end request latency

---

# Integrating a Project With DispatchAI

A host project does **not** need to move its business logic into DispatchAI.

Instead, the project exposes an integration endpoint and implements the tool capabilities it wants DispatchAI to use.

For example, suppose an ecommerce application already has:

```text
customers
orders
products
payments
notifications
```

The application can expose capabilities such as:

```text
get_customers
get_customer_orders
get_order
get_product
send_customer_notification
```

DispatchAI can then orchestrate these capabilities.

---

# Integration Flow

The integration has two sides.

## DispatchAI side

```text
User Request
     │
     ▼
Assistant
     │
     ▼
LLM
     │
     ▼
Tool Call
     │
     ▼
ProjectToolGateway
     │
     ▼
ProjectClient
```

## Host Project side

```text
HTTP Endpoint
     │
     ▼
Tool Gateway
     │
     ▼
Domain Handler
     │
     ▼
Service
     │
     ▼
Repository / External API
```

Together:

```text
┌──────────────────── DispatchAI ────────────────────┐
│                                                    │
│ User Request                                       │
│      ↓                                             │
│ Assistant                                           │
│      ↓                                             │
│ LLM                                                 │
│      ↓                                             │
│ Tool Call                                           │
│      ↓                                             │
│ ProjectToolGateway                                  │
│      ↓                                             │
│ ProjectClient                                       │
│                                                    │
└───────────────────────┬────────────────────────────┘
                        │
                        │ HTTP
                        ▼
┌──────────────────── Host Project ──────────────────┐
│                                                    │
│ /api/v1/ai/tools/execute                            │
│      ↓                                             │
│ Tool Gateway                                       │
│      ↓                                             │
│ Domain Handler                                     │
│      ↓                                             │
│ Business Service                                   │
│      ↓                                             │
│ Database / External Service                        │
│                                                    │
└────────────────────────────────────────────────────┘
```

---

# Step 1 — Create the Host Project Tool Endpoint

Your application exposes:

```http
POST /api/v1/ai/tools/execute
```

The endpoint accepts a generic `ToolRequest`.

Example:

```json
{
  "request_id": "req_123",
  "conversation_id": "conv_456",
  "tool_name": "get_customers",
  "call_id": "call_789",
  "arguments": {
    "limit": 20,
    "offset": 0
  },
  "identity": {
    "user_id": "user_123"
  },
  "tenant": {
    "tenant_id": "tenant_123"
  },
  "authorization": {
    "roles": [
      "admin"
    ],
    "permissions": [
      "customer.read"
    ]
  }
}
```

---

# Step 2 — Implement a Host Project Tool Gateway

The host project maps tool names to its own domain services.

For example:

```go
func (g *ToolGateway) Execute(
    ctx context.Context,
    req *models.ToolRequest,
) (*models.ToolResponse, error) {

    switch req.ToolName {

    case "get_customers":
        return g.customers.GetCustomers(ctx, req)

    case "get_orders":
        return g.orders.GetOrders(ctx, req)

    default:
        return unknownTool(req), nil
    }
}
```

This mapping belongs to the **host project**.

DispatchAI does not contain this mapping.

---

# Step 3 — Implement the Domain Handler

For example:

```go
func (c *Customers) GetCustomers(
    ctx context.Context,
    req *models.ToolRequest,
) (*models.ToolResponse, error) {

    // Validate arguments.

    // Validate authorization.

    // Apply tenant isolation.

    // Call the existing customer service.

    // Return structured data.
}
```

The host project can therefore reuse its existing business architecture:

```text
Tool Gateway
      ↓
Customers
      ↓
Customer Service
      ↓
Customer Repository
      ↓
PostgreSQL
```

There is no need to move the customer implementation into DispatchAI.

---

# Step 4 — Register the Tool With DispatchAI

The host project must provide DispatchAI with the capabilities available to the assistant.

A tool definition contains:

```text
Name
Description
Parameters
```

Example:

```json
{
  "name": "get_customers",
  "description": "Retrieve customers accessible to the current user.",
  "parameters": {
    "type": "object",
    "properties": {
      "limit": {
        "type": "integer",
        "minimum": 1,
        "maximum": 100
      },
      "offset": {
        "type": "integer",
        "minimum": 0
      }
    }
  }
}
```

The description and schema are provided to the LLM.

The LLM can then decide when the capability is appropriate.

---

# Step 5 — Pass User and Tenant Context

The host project remains responsible for identity.

DispatchAI should receive enough context to forward the request safely:

```text
User ID
Tenant ID
Roles
Permissions
Request ID
Conversation ID
Trace ID
```

For example:

```go
models.ExecutionContext{
    RequestID:      "req_123",
    ConversationID: "conv_456",

    Identity: models.Identity{
        UserID: "user_123",
    },

    Tenant: models.TenantContext{
        TenantID: "tenant_123",
    },

    Authorization: models.AuthorizationContext{
        Roles: []string{
            "admin",
        },
        Permissions: []string{
            "customer.read",
        },
    },
}
```

The host project must remain the final authority for authorization.

---

# Complete Example: Ecommerce Integration

Suppose an ecommerce project wants an AI assistant.

Its existing application might look like:

```text
ecommerce/
├── customers/
├── orders/
├── products/
├── payments/
├── notifications/
└── ...
```

It can expose these AI capabilities:

```text
get_customers
get_customer
get_customer_orders
get_order
get_product
send_customer_notification
```

The user asks:

> "Find customers who haven't ordered anything in the last 30 days."

The flow becomes:

```text
User
 │
 │ "Find customers who haven't ordered..."
 ▼
Ecommerce Application
 │
 ▼
DispatchAI
 │
 ▼
LLM
 │
 │ get_inactive_customers
 ▼
DispatchAI
 │
 │ ToolRequest
 ▼
Ecommerce Tool Gateway
 │
 ▼
Customer / Order Services
 │
 ▼
PostgreSQL
 │
 │ ToolResponse
 ▼
DispatchAI
 │
 ▼
LLM
 │
 ▼
Final response
```

DispatchAI never needs to know:

```text
customer table
order table
SQL queries
database schema
customer repository
ecommerce business rules
```

Those remain inside the ecommerce application.

---

# Example: Fleet Integration

A fleet-management application might expose:

```text
get_vehicle_location
get_vehicle_status
get_driver
get_active_trips
send_driver_notification
```

A user could ask:

> "Where is vehicle V-102?"

DispatchAI may generate:

```json
{
  "name": "get_vehicle_location",
  "arguments": {
    "vehicle_id": "V-102"
  }
}
```

The fleet application performs the actual lookup.

DispatchAI does not need to know whether the data comes from:

```text
PostgreSQL
TimescaleDB
Traccar
GPSGate
Redis
External fleet API
```

That is the host project's responsibility.

---

# Example: Field Service Integration

A field-service application could expose:

```text
get_work_order
get_available_technicians
create_work_order
assign_technician
send_customer_notification
```

The AI can orchestrate these capabilities without DispatchAI knowing anything about field-service business logic.

---

# Tool Execution Security

A tool call generated by an LLM must never be treated as authorization.

The correct flow is:

```text
LLM says:
"Call delete_order"

        ↓

DispatchAI forwards request

        ↓

Host Project

        ↓

Authenticate

        ↓

Authorize

        ↓

Validate arguments

        ↓

Apply business rules

        ↓

Execute
```

The LLM saying:

```text
"delete_order"
```

does not mean the user is allowed to delete an order.

The host project must make that decision.

---

# Live Data vs Knowledge

DispatchAI separates two fundamentally different types of information.

## Knowledge

Examples:

```text
Company policies
Product documentation
SOPs
Manuals
Internal documentation
```

These can use:

```text
Knowledge Provider
      ↓
RAG
      ↓
Vector Database
```

## Live Business State

Examples:

```text
Current order status
Current vehicle location
Current account balance
Current inventory
Current technician availability
```

These should normally come from host-project tools.

```text
LLM
 ↓
Tool
 ↓
Host Project
 ↓
Live Business Data
```

This distinction prevents stale knowledge from being treated as authoritative transactional state.

---

# LLM Provider Architecture

DispatchAI does not lock the application to a single LLM provider.

Current adapters:

```text
adapters/llm/
├── anthropic/
├── gemini/
└── openai/
```

The assistant depends on:

```go
llm.Client
```

rather than:

```go
gemini.Client
```

This gives the system provider independence.

For example:

```text
                    llm.Client
                        │
          ┌─────────────┼─────────────┐
          ▼             ▼             ▼
       Gemini         OpenAI       Anthropic
```

---

# Knowledge Provider Architecture

Knowledge follows the same adapter pattern.

```text
internal/knowledge
        │
        ▼
knowledge.Provider
        │
        ▼
adapters/knowledge/pgvector
```

Additional providers can be added later without changing the core assistant.

---

# Storage Architecture

Storage is also abstracted from the core application.

Current adapters include:

```text
adapters/storage/
├── postgres/
└── redis/
```

The purpose is to prevent infrastructure-specific code from leaking into the core orchestration layer.

---

# Notification Architecture

DispatchAI provides generic notification abstractions.

Adapters include:

```text
Email
SMS
Push
Webhook
```

However, notification **business ownership remains with the host application**.

For example:

```text
AI:
"Send a notification to these customers."

        ↓

Host Project:
"Is this user allowed to send it?"

        ↓

Policy / Authorization

        ↓

Host Project notification service

        ↓

Email / SMS / Push / Webhook
```

DispatchAI should not bypass the host project's notification policies.

---

# Error Handling

Errors should remain structured.

A tool result can communicate:

```json
{
  "success": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "User does not have permission.",
    "retryable": false
  }
}
```

This allows the assistant to distinguish between:

```text
Invalid arguments
Unauthorized
Forbidden
Not found
Temporary infrastructure failure
Rate limiting
Business rule violation
Unknown tool
```

The distinction between retryable and non-retryable failures is particularly important for production systems.

---

# Observability

Production AI systems require end-to-end visibility.

A typical request should be traceable across:

```text
HTTP Request
     │
     ▼
Assistant
     │
     ▼
LLM
     │
     ▼
Tool Call
     │
     ▼
Host Project
     │
     ▼
Database / External API
```

Useful telemetry includes:

* Request ID
* Conversation ID
* Trace ID
* Tool call ID
* LLM provider
* LLM model
* Input token count
* Output token count
* LLM latency
* Tool latency
* Tool failures
* Retry counts
* Total request latency

Sensitive data and secrets must not be written into logs.

---

# Testing

The repository separates testing concerns:

```text
tests/
├── unit/
├── integration/
├── evaluation/
├── failure/
└── load/
```

## Unit Tests

Test individual components:

```text
Tool registry
Planner
Tool loop
Policies
Idempotency
Conversation handling
```

## Integration Tests

Test real component boundaries:

```text
Assistant
LLM adapter
Project client
Database
Redis
Knowledge provider
```

## Evaluation Tests

Evaluate AI behavior such as:

```text
Tool selection
Argument generation
Response correctness
Instruction following
Failure handling
```

## Failure Tests

Test scenarios such as:

```text
LLM timeout
LLM 429
LLM 5xx
Malformed tool call
Unknown tool
Host project unavailable
Tool timeout
Database failure
Duplicate execution
Invalid authorization
```

## Load Tests

Measure behavior under realistic concurrency and traffic.

---

# Production Design Principles

DispatchAI follows these principles:

### 1. Domain Agnostic

DispatchAI should work with:

```text
Ecommerce
Fleet
Healthcare
Logistics
IoT
Field Service
```

without changing its core orchestration logic.

### 2. Host-Owned Business Logic

Business rules belong to the host project.

### 3. Host-Owned Authorization

The LLM cannot grant permissions.

### 4. Host-Owned Data

The host project remains the source of truth for transactional data.

### 5. Provider Independence

LLM providers are adapters.

### 6. Infrastructure Independence

Storage and knowledge systems are adapters.

### 7. Structured Contracts

Communication between DispatchAI and host projects uses explicit contracts.

### 8. Observable Execution

AI decisions and tool execution should be traceable.

### 9. Safe Side Effects

Actions that modify state should have appropriate authorization, policy, idempotency, and potentially human approval.

### 10. AI Is an Orchestrator, Not the Business System

This is the central architectural principle.

---

# Running the Example

The repository contains an example ecommerce integration:

```text
examples/ecommerce/
├── cmd/
│   └── main.go
├── customers/
│   └── handler.go
└── http/
    └── http.go
```

The example demonstrates how an existing application can expose its business capabilities to DispatchAI.

The example is intentionally kept separate from the core DispatchAI packages so that domain-specific code does not leak into the reusable engine.

---

# Deployment

DispatchAI includes deployment definitions for:

```text
Docker
Kubernetes
```

Docker:

```text
deployments/docker/
├── Dockerfile
└── docker-compose.yml
```

Kubernetes:

```text
deployments/kubernetes/
├── configmap.yaml
├── deployment.yaml
└── service.yaml
```

Production deployments should additionally consider:

* Secrets management
* TLS
* Authentication between services
* Network policies
* Rate limiting
* Resource limits
* Horizontal scaling
* LLM provider quotas
* Database connection limits
* Redis capacity
* Observability
* Failure recovery

---

# Future Durable Orchestration

DispatchAI can eventually integrate durable workflow orchestration for operations that should survive:

* Process crashes
* Long waits
* Retries
* Human approvals
* Large fan-out operations
* Long-running workflows

Examples:

```text
Notify 500,000 customers
        ↓
Durable workflow
        ↓
Batch
        ↓
Retry
        ↓
Rate limit
        ↓
Track progress
        ↓
Complete
```

Durable workflow orchestration is **not required for every AI request**.

A simple conversational request should remain a simple request/response operation.

---

# Integration Checklist

To integrate an existing application with DispatchAI:

```text
[ ] Add DispatchAI dependency

[ ] Configure an LLM provider

[ ] Create the host project's tool gateway

[ ] Expose /api/v1/ai/tools/execute

[ ] Define the tools the AI is allowed to use

[ ] Provide JSON schemas for tool arguments

[ ] Implement each tool inside the host project's
    existing business architecture

[ ] Forward authenticated user identity

[ ] Forward tenant context

[ ] Forward authorization context

[ ] Validate authorization inside the host project

[ ] Validate tool arguments

[ ] Apply business rules

[ ] Execute using existing services/repositories

[ ] Return structured ToolResponse objects

[ ] Configure observability

[ ] Add integration tests

[ ] Add failure tests

[ ] Add AI evaluation tests

[ ] Load test the integration
```

---

# What DispatchAI Is Not

DispatchAI is **not**:

* A replacement for your business backend
* A database
* A vector database
* An LLM provider
* A CRM
* An ecommerce system
* A fleet-management system
* A notification provider
* An authorization system
* A source of truth for business data
* A framework that requires every request to become a durable workflow

Instead, it is an **AI orchestration layer that integrates with existing applications**.

---

# The Mental Model

Think of DispatchAI as an AI-powered operations coordinator.

The coordinator can understand:

> "Find the delayed vehicles and notify their assigned drivers."

But the coordinator does not own the fleet system.

It asks the fleet system:

```text
get_delayed_vehicles
```

The fleet system decides what that means and returns the actual data.

Then DispatchAI can request:

```text
get_driver
```

and eventually:

```text
send_driver_notification
```

The host project remains responsible for whether those actions are allowed and how they are executed.

So the relationship is:

```text
                 DISPATCHAI
                     │
             Understands intent
                     │
             Reasons / Plans
                     │
             Selects capabilities
                     │
             Orchestrates calls
                     │
                     ▼
              HOST PROJECT
                     │
              Owns the truth
                     │
              Owns permissions
                     │
              Owns business rules
                     │
              Owns execution
                     │
                     ▼
          Database / APIs / Services
```

---

# Final Architecture Principle

The entire system can be summarized as:

```text
┌────────────────────────────────────────────────────┐
│                    DISPATCHAI                     │
│                                                    │
│  Understand                                         │
│  Reason                                             │
│  Plan                                               │
│  Orchestrate                                        │
│  Use LLMs                                           │
│  Manage conversations                               │
│  Coordinate tools                                   │
│  Retrieve knowledge                                 │
│  Observe execution                                  │
│                                                    │
└────────────────────────┬───────────────────────────┘
                         │
                         │ Generic contracts
                         │
                         ▼
┌────────────────────────────────────────────────────┐
│                  HOST PROJECT                      │
│                                                    │
│  Authenticate                                       │
│  Authorize                                          │
│  Validate                                           │
│  Apply business rules                               │
│  Execute tools                                      │
│  Own databases                                      │
│  Own business services                              │
│  Own external integrations                          │
│  Own business truth                                 │
│                                                    │
└────────────────────────────────────────────────────┘
```

> **DispatchAI does not own the business.
> DispatchAI understands and orchestrates the business capabilities exposed by the host project.**
