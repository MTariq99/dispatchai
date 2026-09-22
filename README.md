// This project is a domain-agnostic AI Dispatch and Notification
// Assistant.
//
// The Assistant understands natural-language requests, reasons about
// what information or capabilities it needs, requests tools through
// the Host Project integration contract, interprets returned results,
// and continues reasoning until the task is complete.
//
// The Assistant does NOT own business data or business logic.
//
// The Host Project owns:
//
//     - Authentication
//     - Authorization
//     - Business rules
//     - Business services
//     - Databases
//     - Tool execution
//     - Notification execution
//
// The Assistant owns:
//
//     - LLM interaction
//     - Conversation context
//     - Planning
//     - Tool selection
//     - Tool-call orchestration
//     - Result interpretation
//     - Final response generation
//
// The fundamental execution flow is:
//
//     User
//       ↓
//     Host Project
//       ↓
//     AI Assistant
//       ↓
//     LLM
//       ↓
//     Tool Call
//       ↓
//     Host Project
//       ↓
//     Business Services / Database
//       ↓
//     Tool Result
//       ↓
//     AI Assistant
//       ↓
//     LLM
//       ↓
//     Final Response
//       ↓
//     Host Project
//       ↓
//     User


# DispatchAI

DispatchAI is a **domain-agnostic AI Dispatch and Notification Assistant** that can be integrated into different existing backend applications.

The goal is not to build an ecommerce chatbot, fleet chatbot, healthcare chatbot, or another domain-specific AI application.

The goal is to build a reusable AI Assistant that can connect to **any Host Project** through a well-defined capability/tool contract.

The Host Project owns the business.

DispatchAI owns the AI reasoning and orchestration.

---

# 1. Core Idea

The most important rule in this project is:

> **The AI Assistant does not own the business. It understands and orchestrates. The Host Project owns the business, data, authorization, and execution.**

For example, a user might say:

> "Find all customers whose orders are delayed by more than 30 minutes and notify them."

DispatchAI does not directly query an orders database.

Instead:

```text
User
 |
 | "Find delayed customers and notify them"
 v
Host Project
 |
 | authenticated request
 | available capabilities
 v
DispatchAI
 |
 v
LLM
 |
 | "I need find_delayed_orders"
 v
DispatchAI
 |
 | tool request
 v
Host Project
 |
 | authorization
 | validation
 | business logic
 | database
 v
Tool Result
 |
 v
DispatchAI
 |
 v
LLM
 |
 | "I need customer preferences"
 v
DispatchAI
 |
 v
Host Project
 |
 v
Tool Result
 |
 v
LLM
 |
 | "send_notification"
 v
DispatchAI
 |
 v
Host Project
 |
 | business validation
 | notification service
 v
Tool Result
 |
 v
LLM
 |
 | final response
 v
DispatchAI
 |
 v
Host Project
 |
 v
User
```

The same architecture works for:

```text
E-commerce
Fleet Management
Healthcare
Field Service
IoT
Logistics
CRM
Finance
SaaS Applications
```

The Assistant does not need to know what the business domain is.

It only needs to know what capabilities the Host Project exposes.

---

# 2. Responsibility Boundary

There are two systems.

## Host Project

The existing application that wants to integrate AI.

The Host Project owns:

```text
Authentication
Authorization
Tenant Context
Business Rules
Business Services
Database
External Business APIs
Tool Implementations
Notification Delivery
Business State
```

For example:

```text
Host Project
 |
 +-- Order Service
 +-- Customer Service
 +-- PostgreSQL
 +-- Payment Service
 +-- Notification Service
```

---

# 3. DispatchAI

DispatchAI owns:

```text
Natural-language understanding
LLM interaction
Conversation context
Planning
Tool selection
Tool-call orchestration
Tool-result interpretation
Final response generation
Assistant memory
AI observability
AI-specific audit information
```

DispatchAI does **not** own:

```text
Orders
Customers
Vehicles
Drivers
Patients
Technicians
Payments
Business authorization
Business truth
```

---

# 4. The Most Important Boundary

Never implement this:

```text
LLM
 |
 +----> Database
```

Never implement this:

```text
LLM
 |
 +----> Send SMS
```

Never implement this:

```text
LLM
 |
 +----> Update Order
```

Instead:

```text
LLM
 |
 | structured tool call
 v
DispatchAI
 |
 v
Host Project
 |
 +--> Authorization
 +--> Business Rules
 +--> Business Service
 +--> Database
 +--> External API
 |
 v
Tool Result
 |
 v
DispatchAI
 |
 v
LLM
```

The LLM is allowed to **request** an operation.

The Host Project decides whether and how that operation happens.

---

# 5. Repository Structure

```text
dispatch-ai/
│
├── cmd/
│   ├── assistant/
│   │   └── main.go
│   │
│   └── example-project/
│       └── main.go
│
├── internal/
│   │
│   ├── assistant/
│   │   ├── assistant.go
│   │   ├── orchestrator.go
│   │   ├── conversation.go
│   │   ├── planner.go
│   │   ├── tool_loop.go
│   │   └── context.go
│   │
│   ├── llm/
│   │   ├── llm.go
│   │   ├── request.go
│   │   ├── response.go
│   │   ├── tool_call.go
│   │   └── mock.go
│   │
│   ├── tools/
│   │   ├── tool.go
│   │   ├── registry.go
│   │   ├── definition.go
│   │   ├── call.go
│   │   ├── result.go
│   │   └── executor.go
│   │
│   ├── project/
│   │   ├── client.go
│   │   ├── tool_gateway.go
│   │   ├── request.go
│   │   └── response.go
│   │
│   ├── notification/
│   │   ├── notification.go
│   │   ├── dispatcher.go
│   │   ├── channel.go
│   │   ├── delivery.go
│   │   └── preference.go
│   │
│   ├── policy/
│   │   ├── policy.go
│   │   ├── engine.go
│   │   ├── decision.go
│   │   └── rules.go
│   │
│   ├── approval/
│   │   ├── approval.go
│   │   ├── request.go
│   │   └── status.go
│   │
│   ├── audit/
│   │   ├── audit.go
│   │   ├── event.go
│   │   └── store.go
│   │
│   ├── idempotency/
│   │   ├── idempotency.go
│   │   └── store.go
│   │
│   ├── memory/
│   │   ├── conversation.go
│   │   └── store.go
│   │
│   ├── knowledge/
│   │   ├── knowledge.go
│   │   └── provider.go
│   │
│   ├── observability/
│   │   ├── logging.go
│   │   ├── metrics.go
│   │   └── tracing.go
│   │
│   ├── api/
│   │   ├── handler.go
│   │   ├── request.go
│   │   ├── response.go
│   │   └── routes.go
│   │
│   └── config/
│       └── config.go
│
├── adapters/
│   │
│   ├── llm/
│   │   ├── openai/
│   │   │   └── client.go
│   │   ├── gemini/
│   │   │   └── client.go
│   │   └── anthropic/
│   │       └── client.go
│   │
│   ├── notification/
│   │   ├── email/
│   │   ├── sms/
│   │   ├── push/
│   │   └── webhook/
│   │
│   ├── storage/
│   │   ├── postgres/
│   │   └── redis/
│   │
│   └── knowledge/
│       └── pgvector/
│
├── contracts/
│   ├── dispatch.proto
│   ├── tool.proto
│   └── notification.proto
│
├── examples/
│   ├── ecommerce/
│   ├── fleet/
│   └── field-service/
│
├── tests/
│   ├── unit/
│   ├── integration/
│   ├── evaluation/
│   ├── failure/
│   └── load/
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile
│   │   └── docker-compose.yml
│   │
│   └── kubernetes/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── configmap.yaml
│
├── migrations/
├── .env.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

# 6. The Dependency Direction

The project should follow this general direction:

```text
cmd
 |
 v
api
 |
 v
assistant
 |
 +----> llm
 |
 +----> tools
 |        |
 |        v
 |     project
 |
 +----> memory
 |
 +----> policy
 |
 +----> audit
 |
 +----> observability
 |
 v
adapters
```

Provider-specific implementations stay behind interfaces.

For example:

```text
Assistant
    |
    v
llm.Client
    |
    +------> Gemini Adapter
    |
    +------> OpenAI Adapter
    |
    +------> Anthropic Adapter
```

The Assistant does not know which provider is being used.

---

# 7. LLM Architecture

`internal/llm` contains the application's generic LLM abstraction.

```text
internal/llm/

llm.go
    |
    +-- Client interface

request.go
    |
    +-- Internal LLM request

response.go
    |
    +-- Internal LLM response

tool_call.go
    |
    +-- Normalized tool call

mock.go
    |
    +-- Testing implementation
```

Provider implementations live here:

```text
adapters/llm/

openai/client.go
gemini/client.go
anthropic/client.go
```

The flow is:

```text
Assistant
   |
   v
llm.Client
   |
   v
Gemini Adapter
   |
   v
Gemini API
```

or:

```text
Assistant
   |
   v
llm.Client
   |
   v
OpenAI Adapter
   |
   v
OpenAI API
```

This prevents provider-specific code from leaking into the Assistant.

---

# 8. Complete Runtime Flow

Suppose the user sends:

```text
"Notify customers whose orders have been delayed by more than 30 minutes."
```

## Step 1 — Host Project receives the request

The Host Project authenticates the user.

It knows:

```text
user_id
tenant_id
roles
permissions
```

It also knows what capabilities it wants to expose.

For example:

```text
find_delayed_orders
get_customer
get_customer_preferences
send_notification
```

The Host Project sends DispatchAI:

```text
User Request
+
Authenticated Context
+
Available Tool Definitions
```

---

# 9. `cmd/assistant/main.go`

The Assistant process starts here.

Execution begins:

```text
cmd/assistant/main.go
```

`main()` initializes:

```text
Config
LLM
Project Client
Tool Registry
Memory
Policy
Audit
Observability
Assistant
HTTP Server
```

Conceptually:

```text
main()
 |
 +--> load config
 |
 +--> create LLM client
 |
 +--> create project client
 |
 +--> create memory store
 |
 +--> create policy engine
 |
 +--> create audit store
 |
 +--> create observability
 |
 +--> create Assistant
 |
 +--> create API handler
 |
 +--> register routes
 |
 +--> start HTTP server
```

`main.go` contains no AI reasoning.

It only wires objects together.

---

# 10. `internal/api/routes.go`

The routes are registered.

For example:

```text
POST /v1/assistant/chat
POST /v1/assistant/dispatch
POST /v1/assistant/events
GET  /v1/assistant/conversations/:id
```

A request enters:

```text
HTTP
 |
 v
api/routes.go
 |
 v
api/handler.go
```

---

# 11. `internal/api/handler.go`

The handler receives the HTTP request.

It should:

```text
Decode request
Validate transport-level input
Extract metadata
Call Assistant
Return response
```

It should NOT:

```text
Call LLM directly
Execute tools
Query business database
Send notification
```

The handler calls:

```text
assistant.HandleRequest(...)
```

---

# 12. `internal/assistant/assistant.go`

This is the public entry point to the Assistant core.

It receives something conceptually like:

```text
Request
User Context
Tenant Context
Conversation ID
Available Capabilities
```

Then it passes the request to:

```text
orchestrator.Run(...)
```

This keeps the API layer separated from the internal reasoning engine.

---

# 13. `internal/assistant/context.go`

The Assistant builds the context that will be provided to the LLM.

It combines:

```text
User Request
+
Authenticated Identity
+
Tenant Context
+
Conversation History
+
Available Tools
+
Optional Knowledge
```

The resulting context is given to the LLM.

Important:

The LLM does not get to decide:

```text
"I am admin."

"I can access tenant X."

"I am allowed to send this notification."
```

Those facts come from trusted application context.

---

# 14. `internal/tools/registry.go`

The Host Project provides available capabilities.

For example:

```text
find_delayed_orders
get_customer
get_customer_preferences
send_notification
```

The registry stores the definitions available for this request.

It is NOT a registry of business implementations.

This distinction is extremely important.

DispatchAI stores:

```text
Tool Name
Description
Input Schema
Metadata
```

The Host Project owns:

```text
Actual implementation
Database
Business service
Business rules
Authorization
```

---

# 15. `internal/tools/definition.go`

A tool definition tells the LLM what capability exists.

Example:

```text
Name:
    find_delayed_orders

Description:
    Find orders delayed beyond a specified number of minutes.

Arguments:
    delay_minutes: integer
```

The definition is converted into the format expected by the selected LLM provider.

---

# 16. `internal/llm/request.go`

The Assistant creates an internal LLM request.

Conceptually:

```text
System Instructions
+
Conversation
+
User Request
+
Available Tool Definitions
+
Relevant Context
```

This is the application's internal representation.

It is NOT an OpenAI request.

It is NOT a Gemini request.

---

# 17. `internal/llm/llm.go`

The Assistant calls:

```text
llm.Client.Generate(...)
```

The Assistant doesn't care whether the implementation is:

```text
Gemini
OpenAI
Anthropic
Mock
```

It only knows:

```text
Request -> Response
```

---

# 18. `adapters/llm/gemini/client.go`

If Gemini is configured, the Gemini adapter receives the generic request.

It converts:

```text
DispatchAI Request
       ↓
Gemini API Request
```

Calls Gemini.

Then converts:

```text
Gemini Response
       ↓
DispatchAI Response
```

The rest of the application never needs to know Gemini's API structures.

---

# 19. `internal/llm/response.go`

The response is normalized.

The LLM might return:

```text
Tool Call:

name:
    find_delayed_orders

arguments:
    {
        "delay_minutes": 30
    }
```

DispatchAI converts that into its internal `ToolCall`.

At this point:

**Nothing has been executed.**

The LLM only requested a capability.

---

# 20. `internal/assistant/tool_loop.go`

The tool loop sees:

```text
LLM returned Tool Call
```

It asks:

```text
Is this a valid tool?
Are the arguments valid?
Are execution limits respected?
```

Then it passes the call to:

```text
internal/tools/executor.go
```

---

# 21. `internal/tools/executor.go`

The executor does NOT call a business service.

Instead:

```text
Tool Call
   |
   v
Project Client
```

For example:

```text
find_delayed_orders({
    delay_minutes: 30
})
```

becomes a Host Project execution request.

---

# 22. `internal/project/request.go`

The request contains the tool operation plus trusted metadata.

Conceptually:

```text
Tool:
    find_delayed_orders

Arguments:
    delay_minutes = 30

Request ID:
    req-123

Conversation ID:
    conv-456

User:
    user-789

Tenant:
    tenant-abc

Authorization Context:
    ...
```

---

# 23. `internal/project/client.go`

The Project Client sends the request to the Host Project.

The transport may be:

```text
HTTP
```

or:

```text
gRPC
```

The flow is:

```text
DispatchAI
    |
    v
Project Client
    |
    v
Host Project
```

---

# 24. Host Project Executes the Tool

This is where actual business logic happens.

The Host Project receives:

```text
find_delayed_orders
```

It performs:

```text
Authentication / context validation
        ↓
Authorization
        ↓
Business validation
        ↓
Order Service
        ↓
PostgreSQL
        ↓
Result
```

For example:

```text
[
    ORD-1001,
    ORD-1007,
    ORD-1014
]
```

DispatchAI never directly queried the orders database.

---

# 25. `internal/project/response.go`

The Host Project sends a structured result back.

Conceptually:

```text
success: true

data:
    delayed_orders:
        - ORD-1001
        - ORD-1007
        - ORD-1014
```

DispatchAI converts this into its internal `ToolResult`.

---

# 26. `internal/tools/result.go`

The result is represented internally.

The Assistant adds it to the conversation.

Now the LLM sees:

```text
User:
    Notify customers whose orders are delayed >30 minutes.

Assistant:
    Called find_delayed_orders.

Tool:
    3 delayed orders found.
```

The LLM can now reason about what it needs next.

---

# 27. The Loop Continues

The LLM may request:

```text
get_customer
```

Then:

```text
get_customer_preferences
```

Then:

```text
send_notification
```

Each time:

```text
LLM
 ↓
DispatchAI
 ↓
Host Project
 ↓
Business Service
 ↓
Tool Result
 ↓
DispatchAI
 ↓
LLM
```

The loop continues until the LLM returns a final response.

---

# 28. Final Response

Eventually the LLM may return:

```text
"I notified 3 customers about their delayed orders."
```

The Assistant returns this to the API.

Then:

```text
Assistant
   ↓
API Handler
   ↓
Host Project
   ↓
User
```

---

# 29. Complete File-to-File Execution Flow

The most important sequence to understand is this:

```text
cmd/assistant/main.go
        |
        v
internal/api/routes.go
        |
        v
internal/api/handler.go
        |
        v
internal/assistant/assistant.go
        |
        v
internal/assistant/orchestrator.go
        |
        v
internal/assistant/context.go
        |
        v
internal/assistant/conversation.go
        |
        v
internal/tools/registry.go
        |
        v
internal/llm/request.go
        |
        v
internal/llm/llm.go
        |
        v
adapters/llm/gemini/client.go
        |
        v
Gemini API
        |
        v
internal/llm/response.go
        |
        v
internal/assistant/tool_loop.go
        |
        v
internal/tools/call.go
        |
        v
internal/tools/executor.go
        |
        v
internal/project/request.go
        |
        v
internal/project/client.go
        |
        v
HOST PROJECT
        |
        v
Business Service
        |
        v
Database
        |
        v
internal/project/response.go
        |
        v
internal/tools/result.go
        |
        v
internal/assistant/conversation.go
        |
        v
internal/llm/request.go
        |
        v
LLM
        |
        +----> another tool call
        |          |
        |          +----> repeat
        |
        +----> final response
                    |
                    v
              orchestrator.go
                    |
                    v
              assistant.go
                    |
                    v
              handler.go
                    |
                    v
                  USER
```

This is the flow you should be able to explain from memory.

---

# 30. Where Policy Fits

Policy should sit around execution.

Conceptually:

```text
LLM
 |
 | tool call
 v
Tool Executor
 |
 v
Policy Engine
 |
 +---- DENY
 |
 +---- REQUIRE_APPROVAL
 |
 +---- ALLOW
          |
          v
     Host Project
```

The LLM cannot override policy.

---

# 31. Where Audit Fits

Important operations should generate audit events.

For example:

```text
Request received
      ↓
LLM call
      ↓
Tool requested
      ↓
Policy evaluated
      ↓
Tool executed
      ↓
Tool result
      ↓
Final response
```

Audit provides a historical record of what happened.

---

# 32. Where Idempotency Fits

Idempotency protects operations that must not happen twice.

For example:

```text
LLM
 |
 | send_notification
 v
Policy
 |
 v
Idempotency Check
 |
 +---- already completed -> don't execute again
 |
 +---- new operation
          |
          v
      Host Project
```

This becomes especially important when retries or asynchronous execution are introduced.

---

# 33. Where Memory Fits

Conversation memory stores Assistant conversation state.

It does NOT replace Host Project business data.

For example:

```text
Memory:
    "The user asked about delayed orders."

Host Project:
    "Order ORD-123 is currently delayed by 47 minutes."
```

The second piece must come from the Host Project.

---

# 34. Where RAG Fits

RAG is optional.

If enabled:

```text
User Request
     |
     v
Assistant
     |
     +----> Knowledge Provider
     |           |
     |           v
     |       pgvector
     |
     v
    LLM
```

RAG should provide:

```text
Policies
SOPs
Documentation
Manuals
Internal Knowledge
```

RAG should NOT be used as the authoritative source for:

```text
Current order status
Current vehicle location
Current account balance
Current appointment status
Current inventory
```

Those should come from Host Project tools.

---

# 35. Where Temporal Fits

Temporal should NOT be introduced into every request.

A simple request can remain:

```text
HTTP
 ↓
Assistant
 ↓
LLM
 ↓
Tool
 ↓
Result
 ↓
Final Response
```

Temporal becomes useful when the operation becomes durable or long-running.

For example:

```text
"Notify 500,000 customers about this incident,
retry failures, respect provider limits,
and ask me for approval before sending."
```

Then the architecture can become:

```text
Host Project
      |
      v
DispatchAI
      |
      v
Plan
      |
      v
Temporal Workflow
      |
      +--> Resolve recipients
      |
      +--> Human approval
      |
      +--> Batch notifications
      |
      +--> Retry failures
      |
      +--> Track completion
      |
      v
Completion
```

Temporal is an execution durability mechanism.

It does not replace the LLM.

---

# 36. EXACT IMPLEMENTATION ORDER

Do NOT start by implementing every file in the repository.

That will create dependency problems.

Build the system vertically.

The first goal is:

> **One complete request must successfully travel from the API → LLM → tool call → Host Project → tool result → LLM → final response.**

Only after that should you add production capabilities.

---

# Phase 1 — Create the project

Start here:

```text
go.mod
.env.example
README.md
```

Then create:

```text
cmd/assistant/main.go
```

Initially, `main.go` can simply prove that the application starts.

Do not build everything yet.

---

# Phase 2 — Build the LLM abstraction

Implement these files:

```text
internal/llm/llm.go
internal/llm/request.go
internal/llm/response.go
internal/llm/tool_call.go
internal/llm/mock.go
```

Goal:

```text
Assistant can say:

Generate(request)

and receive:

Response
```

At this stage, use the mock LLM first.

Do NOT start with Gemini/OpenAI.

First prove the abstraction.

---

# Phase 3 — Build the first real LLM adapter

Choose ONE provider.

For example:

```text
adapters/llm/gemini/client.go
```

Implement:

```text
internal Request
       ↓
Gemini Request
       ↓
Gemini API
       ↓
Gemini Response
       ↓
internal Response
```

Do not implement OpenAI and Anthropic yet.

You only need one working provider to prove the architecture.

---

# Phase 4 — Build tool representations

Now implement:

```text
internal/tools/tool.go
internal/tools/definition.go
internal/tools/call.go
internal/tools/result.go
internal/tools/registry.go
```

At this point the Assistant should understand:

```text
What tools exist?
What arguments do they accept?
What tool did the LLM request?
What result came back?
```

Still do not implement business logic.

---

# Phase 5 — Build Host Project communication

Now implement:

```text
internal/project/request.go
internal/project/response.go
internal/project/client.go
internal/project/tool_gateway.go
```

For the first implementation, keep the transport simple.

You can use HTTP first.

The goal is:

```text
DispatchAI
   |
   | execute tool
   v
Example Host Project
   |
   | result
   v
DispatchAI
```

Do not add gRPC merely for the sake of using gRPC.

First make the boundary work.

---

# Phase 6 — Build the Example Host Project

Now implement:

```text
cmd/example-project/main.go
```

Give it ONE tool first.

For example:

```text
get_customer
```

Then add:

```text
get_customer_preferences
```

Then:

```text
send_notification
```

Do not create a real ecommerce system.

The example project exists only to demonstrate the integration boundary.

---

# Phase 7 — Build the Tool Executor

Now implement:

```text
internal/tools/executor.go
```

The executor should perform:

```text
LLM ToolCall
      ↓
Validate
      ↓
Project Client
      ↓
Host Project
      ↓
Tool Result
```

At this point you have the critical architectural boundary working.

---

# Phase 8 — Build the Assistant

Now implement:

```text
internal/assistant/context.go
internal/assistant/conversation.go
internal/assistant/tool_loop.go
internal/assistant/orchestrator.go
internal/assistant/assistant.go
```

Implement the smallest complete loop:

```text
User
 ↓
Assistant
 ↓
LLM
 ↓
Tool Call
 ↓
Host Project
 ↓
Tool Result
 ↓
LLM
 ↓
Final Answer
```

DO NOT add planning, RAG, Temporal, approvals, complex memory, etc. yet.

Get this loop working first.

---

# Phase 9 — Build the API

Now implement:

```text
internal/api/request.go
internal/api/response.go
internal/api/handler.go
internal/api/routes.go
```

Expose:

```text
POST /v1/assistant/chat
```

Now you can test:

```text
curl
   ↓
API
   ↓
Assistant
   ↓
LLM
   ↓
Tool
   ↓
Host Project
   ↓
Tool Result
   ↓
LLM
   ↓
Response
```

At this point you have your **first vertical slice**.

---

# Phase 10 — STOP AND VERIFY

Before continuing, verify this exact scenario:

```text
User:

"Get information about customer 123."
```

Expected:

```text
API
 ↓
Assistant
 ↓
LLM
 ↓
get_customer(123)
 ↓
Host Project
 ↓
Customer Service
 ↓
Result
 ↓
LLM
 ↓
Final Answer
```

If this does not work reliably, **do not continue**.

Fix it here.

---

# Phase 11 — Add Conversation Memory

Implement:

```text
internal/memory/conversation.go
internal/memory/store.go
```

Start with an in-memory implementation.

Do not immediately introduce Postgres.

First prove:

```text
Conversation
 ↓
Message
 ↓
Tool Call
 ↓
Tool Result
 ↓
Next Message
```

Then add persistent storage.

---

# Phase 12 — Add Planning

Now implement:

```text
internal/assistant/planner.go
```

Test a multi-step request:

```text
"Find delayed customers and notify them."
```

The Assistant should now be able to perform multiple tool calls.

---

# Phase 13 — Add Policy

Implement:

```text
internal/policy/policy.go
internal/policy/decision.go
internal/policy/rules.go
internal/policy/engine.go
```

Then enforce:

```text
Tool Call
 ↓
Policy
 ↓
ALLOW / DENY / REQUIRE_APPROVAL
```

Do not let the LLM decide policy.

---

# Phase 14 — Add Idempotency

Implement:

```text
internal/idempotency/idempotency.go
internal/idempotency/store.go
```

Test duplicate execution.

Especially test:

```text
send_notification
```

because duplicate notifications are a realistic failure scenario.

---

# Phase 15 — Add Audit

Implement:

```text
internal/audit/audit.go
internal/audit/event.go
internal/audit/store.go
```

Record:

```text
Request
LLM call
Tool call
Policy decision
Tool result
Final response
```

---

# Phase 16 — Add Observability

Implement:

```text
internal/observability/logging.go
internal/observability/metrics.go
internal/observability/tracing.go
```

Then you should be able to trace:

```text
Request
 ↓
LLM
 ↓
Tool
 ↓
Host Project
 ↓
Database
 ↓
LLM
 ↓
Response
```

---

# Phase 17 — Add Approval

Implement:

```text
internal/approval/status.go
internal/approval/request.go
internal/approval/approval.go
```

Then introduce:

```text
Policy
 ↓
REQUIRE_APPROVAL
 ↓
Human
 ↓
APPROVED
 ↓
Tool Execution
```

---

# Phase 18 — Add Persistent Storage

Now implement:

```text
adapters/storage/postgres/
adapters/storage/redis/
migrations/
```

At this point you already have a working system.

You are replacing temporary implementations with production infrastructure.

This is much safer than introducing the database at the beginning.

---

# Phase 19 — Add RAG

Now implement:

```text
internal/knowledge/knowledge.go
internal/knowledge/provider.go
adapters/knowledge/pgvector/
```

Test:

```text
User
 ↓
Assistant
 ├──> Knowledge
 │      ↓
 │   Documentation
 │
 └──> Host Project Tool
        ↓
     Live Business State
```

Keep these two sources separate.

---

# Phase 20 — Add Additional LLM Providers

Only after one provider works:

```text
adapters/llm/openai/
adapters/llm/anthropic/
```

Now you can test:

```text
Same Assistant
     |
     +---- Gemini
     |
     +---- OpenAI
     |
     +---- Anthropic
```

No Assistant code should need to change.

---

# Phase 21 — Add More Example Integrations

Now implement:

```text
examples/ecommerce/
examples/fleet/
examples/field-service/
```

The important test is:

> Can I change the Host Project capabilities without changing the Assistant core?

For example:

```text
E-commerce:

find_delayed_orders
get_customer
send_notification
```

versus:

```text
Fleet:

find_stationary_vehicles
get_driver
send_notification
```

The Assistant code should remain unchanged.

---

# Phase 22 — Failure Testing

Implement:

```text
tests/failure/
```

Test:

```text
LLM timeout
LLM rate limit
LLM malformed response
Unknown tool
Invalid arguments
Host Project unavailable
Host Project timeout
Tool failure
Policy denial
Approval rejection
Duplicate request
Infinite tool loop
Notification failure
```

This is where the project starts becoming operationally serious.

---

# Phase 23 — Evaluation

Implement:

```text
tests/evaluation/
```

Test AI behavior:

```text
Did the model select the correct tool?

Did it generate valid arguments?

Did it avoid unnecessary tools?

Did it correctly interpret tool results?

Did it produce an accurate final response?
```

This is different from normal unit testing.

---

# Phase 24 — Load Testing

Implement:

```text
tests/load/
```

Measure:

```text
Requests/sec
Concurrent users
LLM latency
Tool latency
Host Project latency
Database load
Token usage
Cost
Failure rate
```

---

# Phase 25 — Docker

Only after the application works locally:

```text
deployments/docker/Dockerfile
deployments/docker/docker-compose.yml
```

Run:

```text
Assistant
+
Example Project
+
Postgres
+
Redis
+
Observability
```

---

# Phase 26 — Kubernetes

Only after Docker works reliably:

```text
deployments/kubernetes/
```

Then introduce:

```text
Deployment
Service
ConfigMap
Secrets
Health Checks
Resource Limits
Horizontal Scaling
```

---

# 37. DO NOT IMPLEMENT THE PROJECT TOP-TO-BOTTOM

Do NOT do this:

```text
main.go
↓
every config file
↓
every notification file
↓
every policy file
↓
every storage file
↓
every adapter
↓
Temporal
↓
RAG
↓
finally try to make the Assistant work
```

You will get stuck because too many incomplete abstractions exist simultaneously.

Instead build **vertical slices**.

---

# 38. The Correct Learning/Implementation Path

Your development path should be:

```text
                    PHASE 1

                 API Request
                      |
                      v
                  Assistant
                      |
                      v
                     LLM
                      |
                      v
                 Final Answer
```

Then:

```text
                    PHASE 2

                 Assistant
                     |
                     v
                    LLM
                     |
                     v
                 Tool Call
                     |
                     v
               Host Project
                     |
                     v
                 Tool Result
                     |
                     v
                    LLM
                     |
                     v
               Final Answer
```

Then:

```text
                    PHASE 3

              Multi-step Tool Loop
                      |
                      v
               Policy + Limits
                      |
                      v
                 Idempotency
                      |
                      v
                    Audit
                      |
                      v
               Observability
```

Then:

```text
                    PHASE 4

             Persistent Memory
                     +
                   RAG
                     +
                 Approval
```

Then:

```text
                    PHASE 5

              Failure Testing
                     +
               Load Testing
                     +
               Multi-provider
                     +
                Deployment
```

Then finally:

```text
                    PHASE 6

                  Temporal

        Only for operations that actually
        require durable long-running execution.
```

---

# 39. Your First Milestone

Do not think about the entire repository initially.

Your first milestone is only this:

```text
POST /v1/assistant/chat

        |
        v
   Assistant
        |
        v
       LLM
        |
        v
   Tool Call
        |
        v
Host Project
        |
        v
   Tool Result
        |
        v
       LLM
        |
        v
 Final Response
```

Once this works, you have proven the **core architecture**.

Everything else is an additional production capability.

---

# 40. First Files You Should Actually Write

Start with exactly these:

```text
1. go.mod

2. cmd/assistant/main.go

3. internal/llm/llm.go
4. internal/llm/request.go
5. internal/llm/response.go
6. internal/llm/tool_call.go
7. internal/llm/mock.go

8. adapters/llm/gemini/client.go

9. internal/tools/tool.go
10. internal/tools/definition.go
11. internal/tools/call.go
12. internal/tools/result.go
13. internal/tools/registry.go

14. internal/project/request.go
15. internal/project/response.go
16. internal/project/client.go
17. internal/project/tool_gateway.go

18. cmd/example-project/main.go

19. internal/tools/executor.go

20. internal/assistant/context.go
21. internal/assistant/conversation.go
22. internal/assistant/tool_loop.go
23. internal/assistant/orchestrator.go
24. internal/assistant/assistant.go

25. internal/api/request.go
26. internal/api/response.go
27. internal/api/handler.go
28. internal/api/routes.go
```

**Stop here and make the complete flow work.**

Only after that should you add:

```text
29. memory
30. policy
31. idempotency
32. audit
33. observability
34. approval
35. Postgres
36. Redis
37. RAG
38. additional LLM providers
39. examples
40. failure tests
41. evaluation
42. load tests
43. Docker
44. Kubernetes
45. Temporal
```

---

# 41. The Rule That Prevents You From Getting Stuck

At every phase ask:

> **"What is the smallest end-to-end behavior I can make work before adding another abstraction?"**

For example, before building policy:

```text
Can I successfully execute:

User
→ LLM
→ Tool Call
→ Host Project
→ Tool Result
→ LLM
→ Final Response?
```

Before building RAG:

```text
Can I already execute tools reliably?
```

Before building Temporal:

```text
Do I actually have a long-running/durable operation that requires it?
```

Before Kubernetes:

```text
Does the application work reliably as a Dockerized service?
```

This keeps the project **incremental instead of becoming a giant unfinished architecture**.

---

# 42. Final Architecture

Once everything is implemented, the system looks like:

```text
                         USER / ADMIN
                              |
                              v
                    ┌───────────────────┐
                    │   HOST PROJECT    │
                    │                   │
                    │ Auth              │
                    │ Tenant Context    │
                    │ Business Rules    │
                    │ Business Services │
                    │ Database           │
                    │ Tool Implementations
                    └─────────┬─────────┘
                              |
                    Request + Capabilities
                              |
                              v
                    ┌───────────────────┐
                    │    DISPATCH AI    │
                    │                   │
                    │ API               │
                    │ Assistant         │
                    │ Context           │
                    │ Planner           │
                    │ Tool Loop         │
                    │ Policy            │
                    │ Memory            │
                    │ Audit             │
                    │ Observability     │
                    └─────────┬─────────┘
                              |
                              v
                    ┌───────────────────┐
                    │       LLM         │
                    │                   │
                    │ Understand        │
                    │ Reason            │
                    │ Plan              │
                    │ Select Tool       │
                    │ Generate Args     │
                    │ Interpret Result  │
                    └─────────┬─────────┘
                              |
                         Tool Call
                              |
                              v
                    ┌───────────────────┐
                    │    DISPATCH AI    │
                    │  Tool Executor    │
                    └─────────┬─────────┘
                              |
                              v
                    ┌───────────────────┐
                    │   HOST PROJECT    │
                    │                   │
                    │ Authorization     │
                    │ Validation        │
                    │ Business Logic    │
                    │ Database          │
                    │ External APIs     │
                    │ Notifications     │
                    └─────────┬─────────┘
                              |
                         Tool Result
                              |
                              v
                    ┌───────────────────┐
                    │    DISPATCH AI    │
                    └─────────┬─────────┘
                              |
                              v
                           LLM
                         /     \
                        /       \
                More Tool Calls  Final Answer
                     |                |
                     └───────┐        |
                             |        v
                             |   Host Project
                             |        |
                             |        v
                             |       USER
                             |
                             └──> Repeat
```

The architecture should always preserve this fundamental relationship:

```text
                 AI ASSISTANT
                      |
        "What should happen?"
                      |
                      v
                HOST PROJECT
                      |
        "Is it allowed and how
         should it actually happen?"
                      |
                      v
              BUSINESS SYSTEM
                      |
        "What is the actual state?"
```

That is the foundation of DispatchAI.
