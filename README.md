# DispatchAI

**DispatchAI is a host-agnostic AI dispatch and orchestration engine written in Go.** It lets an existing application expose selected capabilities to an LLM without moving its authentication, authorization, business rules, or transactional data into the AI layer.

> **DispatchAI orchestrates AI and tool interactions. The host application remains the source of business truth.**

A host application might be a logistics platform, fleet-management system, e-commerce service, customer-support application, or internal enterprise product. The host keeps ownership of its users, tenants, domain services, and data. DispatchAI manages the conversation and LLM workflow and coordinates approved tool calls.

## Contents

- [DispatchAI](#dispatchai)
  - [Contents](#contents)
  - [Why DispatchAI](#why-dispatchai)
  - [Core responsibility boundaries](#core-responsibility-boundaries)
  - [Architecture](#architecture)
  - [Integration journey at a glance](#integration-journey-at-a-glance)
  - [Step-by-step integration guide](#step-by-step-integration-guide)
    - [1. Admin creates the integration](#1-admin-creates-the-integration)
    - [2. Host generates its key pair](#2-host-generates-its-key-pair)
    - [3. Host enrolls its public key](#3-host-enrolls-its-public-key)
    - [4. Admin activates the integration](#4-admin-activates-the-integration)
    - [5. Admin grants permissions](#5-admin-grants-permissions)
    - [6. Host registers its tools (signed requests)](#6-host-registers-its-tools-signed-requests)
    - [7. Host implements the tool gateway](#7-host-implements-the-tool-gateway)
    - [8. Host creates a conversation and chats](#8-host-creates-a-conversation-and-chats)
  - [Request signing](#request-signing)
    - [Headers](#headers)
    - [Canonical string (`DISPATCHAI-REQUEST-V1`)](#canonical-string-dispatchai-request-v1)
    - [What the server checks, in order](#what-the-server-checks-in-order)
    - [Go](#go)
    - [Node.js](#nodejs)
    - [Python](#python)
  - [End-to-end chat flow](#end-to-end-chat-flow)
    - [What happens in order](#what-happens-in-order)
  - [API reference](#api-reference)
    - [Chat](#chat)
    - [Tools](#tools)
  - [Host tool gateway contract](#host-tool-gateway-contract)
  - [Idempotency, runs and reliability](#idempotency-runs-and-reliability)
    - [Chat request idempotency](#chat-request-idempotency)
    - [Run states](#run-states)
    - [Tool-call idempotency and policy](#tool-call-idempotency-and-policy)
    - [Reliability principles](#reliability-principles)
    - [Timing expectations](#timing-expectations)
  - [Security model](#security-model)
    - [Required trust-boundary checks](#required-trust-boundary-checks)
  - [How DispatchAI is built](#how-dispatchai-is-built)
    - [Implemented vs scaffolded](#implemented-vs-scaffolded)
  - [Repository layout](#repository-layout)
  - [Configuration and local development](#configuration-and-local-development)
    - [Prerequisites](#prerequisites)
    - [Start PostgreSQL](#start-postgresql)
    - [Configure](#configure)
    - [Run](#run)
    - [Try the whole flow locally](#try-the-whole-flow-locally)
  - [Testing](#testing)
  - [Troubleshooting](#troubleshooting)
  - [Production checklist](#production-checklist)
  - [Known limitations](#known-limitations)
  - [Guiding principle](#guiding-principle)

---

## Why DispatchAI

Without an orchestration layer, an application may need to build LLM calls, conversation management, tool selection, retries, and request tracking directly into its backend.

DispatchAI separates that AI workflow from the host's domain logic:

- **Host application:** authenticates users, establishes trusted identity and tenant context, owns domain services and data, and authorizes and executes business operations.
- **DispatchAI:** coordinates LLM requests, conversation context, registered tools, policy and idempotency checks, run/request state, and the final assistant response.
- **LLM provider:** interprets the conversation and may request a tool. Its output is untrusted input, not authorization and not a source of current business truth.

The host exposes only the capabilities it intends to make available. DispatchAI coordinates those capabilities; it does not replace the host's business backend.

## Core responsibility boundaries

| Concern | Owner | Responsibility |
|---|---|---|
| User authentication | Host application | Authenticate the actual user |
| Tenant and user identity | Host application | Establish and validate trusted context |
| Domain authorization | Host application | Decide whether the user can access or change a resource |
| Business rules | Host application | Validate domain constraints and state transitions |
| Business database | Host application | Own transactional records and authoritative data |
| Tool definitions and routing | DispatchAI, configured with the host | Describe and coordinate exposed capabilities |
| LLM orchestration | DispatchAI | Send context to the model and handle tool-call cycles |
| Conversation and run state | DispatchAI | Track the assistant workflow and persist supported state |
| Request/tool idempotency | DispatchAI and host | Coordinate retries; host must protect external side effects too |
| Final response delivery | DispatchAI → host → frontend | Return the result through the original HTTP request |

**The LLM never becomes the source of business truth.** For a question such as “Where is my shipment?”, the host's shipment service must return the authoritative status. The LLM can then explain that result in natural language.

## Architecture

```text
┌──────────────────────── HOST APPLICATION ────────────────────────┐
│  Frontend · User authentication · Tenant context                 │
│  Authorization · Business rules · Domain services                │
│  Transactional database · External business systems              │
│  Host-owned tool execution endpoint (POST <base>/tools/execute)  │
└─────────────────────────────┬────────────────────────────────────┘
                              │ signed API request (ECDSA P-256)
                              ▼
┌────────────────────────────── DISPATCHAI ─────────────────────────┐
│  Signed-request auth ─► permission check (e.g. assistant:chat)   │
│             │                                                    │
│             ▼                                                    │
│  Request idempotency ── Run lifecycle ── Conversation storage    │
│             │                                                    │
│             ▼                                                    │
│  Orchestrator ──► Tool loop ───────────────────► LLM provider    │
│                      │          final answer ◄───────┤           │
│                      │          or tool calls        │           │
│                      ▼                                           │
│  Tool executor: idempotency → policy → host gateway client       │
└──────────────────────┬────────────────────────────────────────────┘
                       │ POST <PROJECT_BASE_URL>/tools/execute
                       ▼
┌──────────────────────── HOST TOOL GATEWAY ────────────────────────┐
│  Validate request · Authorize user/tenant · Validate arguments    │
│  Apply business rules · Execute domain service · Return result    │
└───────────────────────────────────────────────────────────────────┘
```

There are two directions of traffic, and they are authenticated differently:

| Direction | Purpose | Authentication |
|---|---|---|
| Host → DispatchAI | Chat, conversations, tool registration | Signed requests (ECDSA P-256 + SHA-256) |
| Host → DispatchAI (admin) | Create/activate integrations, grant permissions | `Authorization: Bearer <admin token>` |
| Host → DispatchAI (enroll) | Upload the host's public key | One-time enrollment token in the body |
| DispatchAI → Host | Execute a tool | **Not authenticated by DispatchAI today** (see [Host tool gateway contract](#host-tool-gateway-contract)) |

---

## Integration journey at a glance

Every host follows the same order. Steps 1 and 3–4 are performed by a DispatchAI operator (admin); the rest by the host.

```text
 ADMIN   1. POST /v1/admin/integrations                → client_id + enrollment_token (15 min, single use)
 HOST    2. Generate P-256 key pair locally
 HOST    3. POST /v1/integrations/enroll               → registers key_id + public key (status stays "pending")
 ADMIN   4. POST /v1/admin/integrations/{client_id}/toggle        {"status":"active"}
 ADMIN   5. POST /v1/admin/integrations/{client_id}/permissions   {"permissions":[...]}
 HOST    6. POST /v1/integrations/tools   (signed)     → register each tool the LLM may use
 HOST    7. Run a tool gateway at <PROJECT_BASE_URL>/tools/execute
 HOST    8. POST /v1/conversation/conversation (signed) → conversation_id
 HOST    9. POST /v1/conversation/chat (signed)         → final answer
```

Integration status and permissions are independent. Granting permissions does **not** activate an integration, and an active integration with no grants can call nothing that requires a permission.

```text
        pending ──(enroll key)──► pending ──(admin: active)──► active ◄──► suspended
```

An integration can only be activated if it has at least one usable (active, unrevoked, unexpired) signing key. Only `pending` integrations can enroll a key.

---

## Step-by-step integration guide

The examples use `http://localhost:8989` as the DispatchAI base URL. `$ADMIN_TOKEN` is the onboarding/admin token configured on the DispatchAI server.

### 1. Admin creates the integration

```bash
curl -X POST http://localhost:8989/v1/admin/integrations \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"display_name":"Acme Logistics","metadata":{"env":"dev"}}'
```

`201 Created`:

```json
{
  "client_id": "cli_3f3a5fa0…",
  "status": "pending",
  "enrollment_token": "kB3…(43 chars, base64url)",
  "expires_at": "2026-10-19T12:15:00Z"
}
```

The enrollment token is valid for **15 minutes**, is single use, and is stored only as a SHA-256 hash. Hand `client_id` and `enrollment_token` to the host over a secure channel. `display_name` is required (≤ 200 characters). Unknown JSON fields are rejected.

### 2. Host generates its key pair

The private key **never leaves the host**. DispatchAI stores only the public key.

```bash
# Private key: ECDSA P-256 in PKCS#8 PEM (this is what signing clients load)
openssl ecparam -name prime256v1 -genkey -noout \
  | openssl pkcs8 -topk8 -nocrypt -out private.pem
chmod 600 private.pem

# Public key: standard Base64 (with padding) of the 65-byte uncompressed point 0x04||X||Y.
# The last 65 bytes of the DER SubjectPublicKeyInfo are exactly that point.
openssl ec -in private.pem -pubout -outform DER 2>/dev/null | tail -c 65 | base64 -w0
```

On macOS use `base64 | tr -d '\n'` instead of `base64 -w0`.

### 3. Host enrolls its public key

No signature is required here; the enrollment token is the credential.

```bash
curl -X POST http://localhost:8989/v1/integrations/enroll \
  -H "Content-Type: application/json" \
  -d '{
    "enrollment_token": "<enrollment_token>",
    "key_id": "acme-prod-1",
    "public_key": "<base64 65-byte point>"
  }'
```

`201 Created`:

```json
{ "client_id": "cli_3f3a5fa0…", "key_id": "acme-prod-1", "status": "pending" }
```

`key_id` must match `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`. Store `client_id` and `key_id`; they identify every signed request. The integration is still `pending` and cannot call protected endpoints yet.

### 4. Admin activates the integration

```bash
curl -X POST http://localhost:8989/v1/admin/integrations/$CLIENT_ID/toggle \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"status":"active"}'
```

`200`: `{"status":"active"}`. Only `"active"` and `"suspended"` are accepted. Activation fails with `409` if the integration has no usable signing key or the transition is not allowed (for example activating an already-active integration).

### 5. Admin grants permissions

List the catalog, then grant what the host needs:

```bash
curl http://localhost:8989/v1/admin/permissions -H "Authorization: Bearer $ADMIN_TOKEN"

curl -X POST http://localhost:8989/v1/admin/integrations/$CLIENT_ID/permissions \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"permissions":["assistant:chat","tools:register","tools:read","tools:update","tools:enable","tools:delete"]}'
```

| Permission | Allows |
|---|---|
| `assistant:chat` | Everything under `/v1/conversation/*` |
| `tools:register` | `POST /v1/integrations/tools` |
| `tools:read` | `GET /v1/integrations/tools/:toolID` |
| `tools:update` | `PATCH /v1/integrations/tools/:toolID` |
| `tools:enable` | `PUT /v1/integrations/tools/:toolID/status` |
| `tools:delete` | `DELETE /v1/integrations/tools/:toolID` (soft delete) |

Unknown permission names return `400`. Granting is idempotent (duplicates are ignored).

### 6. Host registers its tools (signed requests)

Each tool is a name, a description the LLM reads, and a JSON-Schema `parameters` object whose top-level `type` must be `"object"`. See [Request signing](#request-signing) for how to sign.

```http
POST /v1/integrations/tools
Idempotency-Key: 7f0c2b4e-…           (required, ≤ 255 chars)
Content-Type: application/json

{
  "name": "get_shipment_status",
  "description": "Look up the current status of a shipment by its tracking number.",
  "parameters": {
    "type": "object",
    "properties": { "tracking_number": { "type": "string" } },
    "required": ["tracking_number"]
  }
}
```

Tool names must match `^[a-zA-Z][a-zA-Z0-9_-]{0,63}$` and be unique among an integration's active tools. Replaying the same key with the same body returns `200` with `"replayed": true`.

### 7. Host implements the tool gateway

Expose `POST <base>/tools/execute` and set `PROJECT_BASE_URL` on the DispatchAI server to `<base>` (no trailing slash needed). The full request/response contract is in [Host tool gateway contract](#host-tool-gateway-contract). A runnable reference host lives in `examples/ecommerce`.

### 8. Host creates a conversation and chats

```http
POST /v1/conversation/conversation
{ "name": "Shipment questions" }
```
→ `201` with the conversation (`id`, `integration_id`, `status`, `name`, timestamps). Then:

```http
POST /v1/conversation/chat
Idempotency-Key: 1c9d…
Content-Type: application/json

{ "conversation_id": "<id>", "message": "Where is shipment ACME-1042?" }
```

→ `200` with the final answer (see [API reference](#api-reference)).

---

## Request signing

Every request to `/v1/integrations/tools/*` and `/v1/conversation/*` must be signed. The admin routes (bearer token) and `/v1/integrations/enroll` are not signed.

### Headers

Each header must appear **exactly once**.

| Header | Value | Validation |
|---|---|---|
| `X-Client-Id` | Your `client_id` | `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$` |
| `X-Key-Id` | The `key_id` you enrolled | same pattern |
| `X-Timestamp` | Current Unix time in **seconds** | 1–12 digits; within **±5 minutes** of server time |
| `X-Nonce` | Random, single-use per integration | `^[A-Za-z0-9_-]{16,64}$` (e.g. 16 random bytes, base64url without padding) |
| `X-Request-Id` | Any unique ID (a UUID works) | same pattern as ids |
| `X-Content-SHA256` | Lowercase hex SHA-256 of the **exact body bytes** | 64 hex chars; empty body → `e3b0c442…b855` |
| `X-Signature-Algorithm` | `ECDSA-P256-SHA256` | exact match |
| `X-Signature` | Base64 (standard, padded) of the **ASN.1 DER** ECDSA signature | max 128 chars |

`Idempotency-Key` (chat and tool registration) and `Content-Type` are **not** part of the signature.

### Canonical string (`DISPATCHAI-REQUEST-V1`)

Eleven fields joined by a single `\n` (LF) — no trailing newline:

```text
DISPATCHAI-REQUEST-V1
<METHOD, uppercase>
<path = URL.EscapedPath()>
<query = URL.RawQuery, or an empty line when there is no query>
<X-Client-Id>
<X-Key-Id>
<X-Timestamp>
<X-Nonce>
<X-Request-Id>
<X-Content-SHA256>
ECDSA-P256-SHA256
```

Then: `signature = ECDSA-P256-Sign(privateKey, SHA256(canonicalBytes))`, DER-encoded, then Base64. Notes:

- The path is the escaped path with no host and no query (`/v1/conversation/chat`). The query string is the raw query without the leading `?` (for tool deletion: `expected_version=3`).
- The body hash must be computed over the exact bytes you send. Re-serializing JSON after hashing breaks verification.
- For requests with no body (GET, DELETE) hash the empty byte string.
- The method, path and query are signed, so a signature cannot be reused for a different endpoint or method.

### What the server checks, in order

1. All headers present, single-valued and well-formed; algorithm equals `ECDSA-P256-SHA256`.
2. Timestamp within ±5 minutes.
3. `(client_id, key_id)` resolves to an **active** key on an **active** integration (not revoked, not expired). Suspended/pending integrations fail here.
4. Body read (max 1 MiB) and SHA-256 compared to `X-Content-SHA256`.
5. Canonical string rebuilt and the signature verified against the stored public key.
6. Nonce recorded (only after a valid signature). A repeated nonce → replay rejection. Nonces are retained for about 11 minutes and purged every minute.
7. The route's required permission is checked against the integration's grants (`403` if missing).

All authentication failures return `401 {"error":"invalid request signature"}`; backend outages return `503`. The specific reason is written to the DispatchAI server log (see [Troubleshooting](#troubleshooting)). **Use a fresh nonce on every attempt, including retries.**

### Go

```go
func sign(key *ecdsa.PrivateKey, clientID, keyID, method string, u *url.URL, body []byte) (map[string]string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil { return nil, err }
	sum := sha256.Sum256(body)

	h := map[string]string{
		"X-Client-Id": clientID, "X-Key-Id": keyID,
		"X-Timestamp":  strconv.FormatInt(time.Now().Unix(), 10),
		"X-Nonce":      base64.RawURLEncoding.EncodeToString(nonceBytes),
		"X-Request-Id": uuid.NewString(),
		"X-Content-SHA256":      hex.EncodeToString(sum[:]),
		"X-Signature-Algorithm": "ECDSA-P256-SHA256",
	}
	path := u.EscapedPath(); if path == "" { path = "/" }
	canonical := strings.Join([]string{
		"DISPATCHAI-REQUEST-V1", strings.ToUpper(method), path, u.RawQuery,
		h["X-Client-Id"], h["X-Key-Id"], h["X-Timestamp"], h["X-Nonce"],
		h["X-Request-Id"], h["X-Content-SHA256"], h["X-Signature-Algorithm"],
	}, "\n")
	digest := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil { return nil, err }
	h["X-Signature"] = base64.StdEncoding.EncodeToString(sig)
	return h, nil
}
```

A complete CLI that signs (and optionally sends) a request is the standalone `dispatchai-client` tool: `-key private.pem -client <client_id> -key-id <key_id> -method POST -url <full url> -body '{…}' [-idem <key>] [-send]`.

### Node.js

```js
const crypto = require('crypto');
const fs = require('fs');

function sign({ privateKeyPem, clientId, keyId, method, url, body = '' }) {
  const u = new URL(url);
  const bodyBuf = Buffer.from(body);
  const h = {
    'X-Client-Id': clientId,
    'X-Key-Id': keyId,
    'X-Timestamp': String(Math.floor(Date.now() / 1000)),
    'X-Nonce': crypto.randomBytes(16).toString('base64url'),
    'X-Request-Id': crypto.randomUUID(),
    'X-Content-SHA256': crypto.createHash('sha256').update(bodyBuf).digest('hex'),
    'X-Signature-Algorithm': 'ECDSA-P256-SHA256',
  };
  const canonical = [
    'DISPATCHAI-REQUEST-V1', method.toUpperCase(), u.pathname, u.search.replace(/^\?/, ''),
    h['X-Client-Id'], h['X-Key-Id'], h['X-Timestamp'], h['X-Nonce'],
    h['X-Request-Id'], h['X-Content-SHA256'], h['X-Signature-Algorithm'],
  ].join('\n');
  // 'der' is Node's default; it is spelled out because the server requires ASN.1 DER.
  h['X-Signature'] = crypto
    .sign('sha256', Buffer.from(canonical), { key: privateKeyPem, dsaEncoding: 'der' })
    .toString('base64');
  return h;
}
```

### Python

```python
import base64, hashlib, os, time, uuid
from urllib.parse import urlsplit
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import ec

def sign(private_pem: bytes, client_id, key_id, method, url, body: bytes = b""):
    key = serialization.load_pem_private_key(private_pem, password=None)
    u = urlsplit(url)
    h = {
        "X-Client-Id": client_id, "X-Key-Id": key_id,
        "X-Timestamp": str(int(time.time())),
        "X-Nonce": base64.urlsafe_b64encode(os.urandom(16)).rstrip(b"=").decode(),
        "X-Request-Id": str(uuid.uuid4()),
        "X-Content-SHA256": hashlib.sha256(body).hexdigest(),
        "X-Signature-Algorithm": "ECDSA-P256-SHA256",
    }
    canonical = "\n".join([
        "DISPATCHAI-REQUEST-V1", method.upper(), u.path or "/", u.query,
        h["X-Client-Id"], h["X-Key-Id"], h["X-Timestamp"], h["X-Nonce"],
        h["X-Request-Id"], h["X-Content-SHA256"], h["X-Signature-Algorithm"],
    ])
    # cryptography returns an ASN.1 DER signature and hashes the input with SHA-256.
    sig = key.sign(canonical.encode(), ec.ECDSA(hashes.SHA256()))
    h["X-Signature"] = base64.b64encode(sig).decode()
    return h
```

---

## End-to-end chat flow

```mermaid
sequenceDiagram
    actor User as Acme User
    participant FE as Acme Frontend
    participant A as Acme Backend
    participant D as DispatchAI
    participant L as LLM

    User->>FE: Where is my shipment?
    FE->>A: Send chat message
    A->>A: Authenticate user, resolve tenant, map conversation → user
    A->>D: POST /v1/conversation/chat (signed, Idempotency-Key)
    D->>D: Verify signature, check assistant:chat, reserve idempotency key, create run
    D->>L: Conversation + enabled tool definitions
    L-->>D: Call get_shipment_status
    D->>D: Tool idempotency + policy checks
    D->>A: POST <PROJECT_BASE_URL>/tools/execute
    A->>A: Authorize user/tenant, fetch shipment
    A-->>D: ToolResponse (success, data)
    D->>L: Tool result
    L-->>D: Final answer
    D->>D: Save conversation, complete run, complete idempotency record
    D-->>A: 200 { answer, run_id, … }
    A-->>FE: Return response
    FE-->>User: Display answer
```

### What happens in order

1. The user submits a message; the host backend authenticates the user and determines trusted user/tenant context.
2. The host calls `POST /v1/conversation/chat`, signed, with an `Idempotency-Key` and an existing `conversation_id`.
3. DispatchAI verifies the signature, checks the integration is active and holds `assistant:chat`, and confirms the conversation belongs to the integration.
4. It checks the idempotency key (replay, conflict, or new), reserves it, creates a run and moves it to `running`.
5. The orchestrator locks the conversation, loads prior messages, appends the user message (plus a system prompt for a new conversation), and starts the tool loop.
6. Each loop iteration sends the conversation and the integration's **enabled** tools to the LLM. If the model requests tools, DispatchAI runs each through its executor (tool idempotency → policy → host gateway) and feeds results back. The loop ends when the model returns no tool calls, or fails after **10 iterations**.
7. A requested tool call is only a *proposal*. The host independently validates and authorizes it in its gateway.
8. The conversation is saved, the run is marked `completed` with the answer, the idempotency record is completed, and the answer is returned in the **original HTTP response**. No callback is needed to receive it.

---

## API reference

Base URL: `http://localhost:8989` (the listen address is currently fixed at `:8989`).

| Method & path | Auth | Permission | Purpose |
|---|---|---|---|
| `POST /v1/admin/integrations` | Admin bearer | — | Create a pending integration + enrollment token |
| `POST /v1/admin/integrations/:client_id/toggle` | Admin bearer | — | Set status `active` / `suspended` |
| `POST /v1/admin/integrations/:client_id/permissions` | Admin bearer | — | Grant permissions |
| `GET /v1/admin/permissions` | Admin bearer | — | Permission catalog |
| `POST /v1/integrations/enroll` | Enrollment token | — | Register a public signing key |
| `POST /v1/integrations/tools` | Signed | `tools:register` | Register a tool |
| `GET /v1/integrations/tools/:toolID` | Signed | `tools:read` | Get a tool |
| `PATCH /v1/integrations/tools/:toolID` | Signed | `tools:update` | Update a tool definition |
| `PUT /v1/integrations/tools/:toolID/status` | Signed | `tools:enable` | Enable/disable a tool |
| `DELETE /v1/integrations/tools/:toolID?expected_version=N` | Signed | `tools:delete` | Soft-delete a tool |
| `POST /v1/conversation/conversation` | Signed | `assistant:chat` | Create a conversation |
| `GET /v1/conversation/:conversation_id` | Signed | `assistant:chat` | Fetch a conversation |
| `POST /v1/conversation/chat` | Signed | `assistant:chat` | Send a message, get the final answer |

`/v1/conversation/chat` is the only chat endpoint; there is no `/v1/assistant/chat` route. Admin and tool-registry routes limit request bodies (1 MiB admin, 64 KiB enrollment).

Admin requests send `Authorization: Bearer <admin token>`. An unconfigured admin token yields `503`; a wrong one `401`.

### Chat

Request body (unknown fields are rejected; a second JSON value is rejected):

```json
{ "conversation_id": "eba2eeef-3c79-480a-84c4-1a6d8b342f10", "message": "Find customers whose orders are delayed by more than 30 minutes and notify them." }
```

`conversation_id` (non-nil UUID) and a non-blank `message` are required. The body type also accepts optional `end_user` and `claims` objects; they take part in the request fingerprint but are **not** forwarded to the host and do not grant anything. Permissions sent to tools come from the integration's server-side grants.

`Idempotency-Key` header: required, non-empty, ≤ 200 characters. It is namespaced per integration, so two hosts can use the same value.

Success `200`:

```json
{
  "request_id": "0f6c…",
  "conversation_id": "eba2eeef-…",
  "run_id": "9b21…",
  "status": "completed",
  "answer": "Your shipment is in transit and is expected to arrive tomorrow."
}
```

| Status | Meaning |
|---|---|
| `200` | Completed (also returned when replaying a completed key) |
| `400` | Invalid body, missing/oversized `Idempotency-Key`, nil `conversation_id` |
| `401` / `403` / `503` | Signature failure / missing permission / auth backend unavailable |
| `404` | Conversation not found for this integration |
| `409` | Key reused with a different request, **or** the same request is still in progress (body carries `status: "pending"`, `request_id`, `run_id` if known) |
| `500` | Run or assistant failure. A persisted failure replays as `500` with `status: "failed"` and no `answer` |

### Tools

Tool definitions are plain JSON (`name`, `description`, `parameters`). Responses wrap the stored record as `{"tool": {...}}`; record fields (including `ID` and `Version`) are currently serialized with capitalised Go field names. Updates and status changes use **optimistic concurrency**: pass the `expected_version` you last read; a stale version returns `409`.

```http
PATCH /v1/integrations/tools/{id}
{ "expected_version": 1, "definition": { "name": "…", "description": "…", "parameters": { "type": "object" } } }

PUT /v1/integrations/tools/{id}/status
{ "expected_version": 2, "enabled": false }
```

Disabled or deleted tools are not offered to the LLM.

---

## Host tool gateway contract

When the LLM asks for a tool, DispatchAI sends:

```http
POST {PROJECT_BASE_URL}/tools/execute
Content-Type: application/json
Accept: application/json
```

```json
{
  "request_id": "0f6c…",
  "run_id": "9b21…",
  "conversation_id": "eba2eeef-…",
  "integration_id": "5d0e…",
  "tool_name": "get_shipment_status",
  "call_id": "call_123",
  "arguments": { "tracking_number": "ACME-1042" },
  "trace": { "trace_id": "", "span_id": "" }
}
```

The host must answer **HTTP 2xx** with:

```json
{
  "request_id": "0f6c…",
  "call_id": "call_123",
  "success": true,
  "data": { "status": "in_transit", "eta": "2026-10-20" },
  "metadata": { "execution_id": "exec-1", "duration_ms": 42 }
}
```

or, for a business-level failure (still HTTP 200):

```json
{
  "request_id": "0f6c…", "call_id": "call_123", "success": false,
  "error": { "code": "NOT_FOUND", "message": "Shipment not found", "retryable": false },
  "metadata": { "execution_id": "exec-2", "duration_ms": 5 }
}
```

Rules and behaviour:

- **Business failures → `success:false` with HTTP 200.** Any non-2xx status, timeout, or undecodable body is recorded as `PROJECT_CLIENT_ERROR` (retryable) and the LLM is told only that the host request failed.
- The `error.code` and `error.message` you return are passed to the LLM, so keep them safe to show to a model and free of secrets.
- **The host must validate everything.** DispatchAI offers the LLM only enabled tools, but it does not currently validate arguments against the registered JSON Schema or re-check the tool name before forwarding. Reject unknown tools (the reference host returns `UNKNOWN_TOOL`), validate and type-check `arguments` (treat `{}`/`null` as “no arguments”), and enforce authorization.
- **End-user identity is not sent.** The payload carries `integration_id`, `conversation_id`, `run_id`, and `request_id`, not the user or tenant. Record which user/tenant owns each `conversation_id` (and optionally each `Idempotency-Key`/`request_id` it sends) when you create the conversation or call chat, and resolve identity from that mapping in the gateway. Never take identity from `arguments`.
- **Deduplicate by `(run_id, call_id)`.** DispatchAI de-duplicates on its side, but an ambiguous timeout means the host may already have executed. The reference host (`examples/ecommerce/http`) shows a fingerprinted in-memory idempotency store; production hosts should persist theirs and protect side effects such as notifications.
- **Timeouts:** DispatchAI waits for `PROJECT_TIME_OUT` (10 s default if unset). A timed-out call is not retried automatically.
- **Authentication of this call is the host's responsibility today.** DispatchAI does not sign or add credentials to gateway calls. Restrict the endpoint to DispatchAI at the network layer (private network, allow-list, mTLS at the proxy).

Reference host: `examples/ecommerce` serves `POST /api/v1/ai/tools/execute`, so for it use `PROJECT_BASE_URL=http://localhost:8087/api/v1/ai`.

---

## Idempotency, runs and reliability

DispatchAI tracks **request idempotency** separately from **run lifecycle** and **tool-call idempotency**.

### Chat request idempotency

Keys are stored per integration with a request fingerprint (message, conversation, and the optional end-user/claims). Behaviour for a repeated key:

| Situation | Result |
|---|---|
| Same key, same request, previously completed | `200` with the stored response, no re-execution |
| Same key, same request, previously failed | `500` with the stored failed response; use a **new key** to retry the operation |
| Same key, same request, still pending | Recovered to `200` if its run already completed; otherwise `409` with `status: "pending"` |
| Same key, different request | `409` |

Clients should reuse a key only to retry the *same* logical operation, and handle `409` explicitly rather than retrying blindly.

### Run states

```text
created → running → completed
                 ↘  failed
created → cancelled;  running → waiting_for_tools | waiting_for_approval → running | failed | cancelled
```

Terminal states: `completed`, `failed`, `cancelled`. In the current chat flow runs move `created → running → completed | failed`; the waiting states and cancellation are defined in the state machine but not driven by the handler yet. Run records store the final answer, error code/message and timestamps (token counters exist but are not populated yet).

### Tool-call idempotency and policy

Each tool call is keyed by `(run_id, call_id)` and stored with its arguments, status, and result. A repeated call returns the stored result; reusing a `call_id` with a different tool or arguments returns `TOOL_CALL_CONFLICT`. Before dispatch the policy engine evaluates the call. The only rule currently installed is `MaxNotificationsPerRun(1)` for a tool named `send_notification`; the engine also supports `DENY` and `REQUIRE_APPROVAL` outcomes (the latter is returned to the LLM as `APPROVAL_REQUIRED` — there is no resume flow yet). If persisting a result fails after the host may have executed, the call returns a non-retryable `TOOL_*_PERSISTENCE_FAILED` error so it is never blindly re-run.

### Reliability principles

- Persist state before reporting durable completion.
- Bind idempotency keys to request fingerprints.
- Do not assume a timeout means the host tool did not execute.
- Do not repeat a non-idempotent business operation simply because DispatchAI failed to receive or persist its response.
- Give host-side side effects their own idempotency protection.
- Treat database/network errors at completion boundaries as potentially ambiguous until durable state is checked.
- Conversation-level locking serializes work on one conversation using a dedicated database connection; lock duration and pool usage must be measured and bounded.
- Idempotency in DispatchAI alone cannot guarantee exactly-once behavior across arbitrary host services or external APIs.

### Timing expectations

The Gemini adapter is configured with a process-wide limiter of one LLM request per 13 seconds and a 30 s HTTP timeout per call. A chat that needs one tool round-trip makes at least two LLM calls, so give your chat client a generous timeout (≥ 60–120 s) and rely on the idempotency key to retry safely.

---

## Security model

- **Signed requests** authenticate the *integration*: ECDSA P-256 / SHA-256 over a versioned canonical string, with timestamp skew limit (5 min), per-integration nonce replay protection, body-hash binding, key identity, and integration status checks.
- **Admin routes** use a static bearer token compared in constant time. Treat it as a high-value secret and rotate it; an unset token disables onboarding (`503`).
- **Enrollment tokens** are 256-bit random, hashed at rest, expire after 15 minutes, and are consumed atomically.
- **Permissions** are an explicit catalog granted per integration and enforced per route.
- **Tenant isolation:** conversations, runs, idempotency records, tools and nonces are scoped by `integration_id`; a conversation ID from another integration returns `404`.
- Signing authenticates the integration request; it does **not** replace the host's end-user authorization. The host must still decide whether the authenticated user can access a shipment, order, account, or other resource. Do not trust arbitrary `user_id`, `tenant_id`, roles, or permissions supplied in a request body.
- The LLM's output, including tool names and arguments, is untrusted input.

Never send a private signing key to DispatchAI. Never commit private keys, tokens, API credentials, `.env` files, or production secrets to source control.

### Required trust-boundary checks

- Verify the signed request, its canonicalization, timestamp/nonce protections, and key identity.
- Reject suspended or otherwise ineligible integrations.
- Check the integration's granted permission for the requested API operation.
- Validate every tool name and argument in the host gateway.
- Independently authorize the end user and tenant in the host gateway.
- Avoid logging signing material, bearer tokens, private data, or unrestricted tool payloads.
- Apply timeouts, request-size limits, rate limits, and network restrictions appropriate to the deployment.

---

## How DispatchAI is built

**Stack:** Go, Gin (HTTP), PostgreSQL (`database/sql` + `lib/pq`), `golang-migrate`, Viper (config), Zap (logging), Google Gemini as the active LLM provider.

**Startup (`cmd/assistant/main.go`):** load config → build logger → open PostgreSQL (creates the database if missing, **applies all migrations automatically**) → construct the Gemini client → wire stores and services (runs, conversations, tool registry/executor, request idempotency, onboarding/admin) → create the signed-request verifier and start the nonce-cleanup goroutine (every minute) → register routes → listen on `:8989`.

**Request path for chat:**

```text
Gin ─► SignedRequestAuth ─► RequirePermission("assistant:chat") ─► ConversationHandler.Chat
   ─► idempotency (GetState/Reserve) ─► RunService (create, start)
   ─► Assistant.HandleRequest ─► Orchestrator.Run
        lock conversation ─► load history ─► ToolLoop.Run (≤ 10 iterations)
             BuildRequest(enabled tools) ─► llm.Client.Generate
             └─ tool calls ─► Executer.Execute
                   tool idempotency ─► policy.Engine ─► ProjectToolGateway ─► host
        save conversation
   ─► RunService.Complete ─► idempotency Complete ─► JSON response
```

**Key design points**

- **Ports and adapters.** `internal/llm.Client` is the provider interface; `adapters/llm/gemini` implements it. Storage interfaces live next to their services and are implemented in `adapters/storage/postgres`.
- **Per-integration tools.** The LLM is offered only the calling integration's enabled, non-deleted tools. Tool definitions are versioned; every change writes an immutable snapshot.
- **System prompt** (`internal/orchestration/prompt.go`) tells the model that it has no direct data access, that authorization is enforced by the host, to ignore identity claims in user messages, and not to fabricate data.
- **Data model** (see `migrations/`): `host_integrations`, `integration_keys`, `integration_enrollment_tokens`, `integration_request_nonces`, `authorization_permissions`, `integration_permission_grants`, `tool_definitions`, `tool_definition_versions`, `tool_registration_idempotency`, `conversations`, `conversation_messages`, `conversation_runs`, `tool_executions`, `idempotency_keys`, plus `audit_events`, `approvals`, and `outbox_events` tables that exist for future use. `integration_credentials` is a legacy table no longer used by the signed-request flow.

### Implemented vs scaffolded

| Area | State |
|---|---|
| Admin onboarding, enrollment, activation, permissions | Implemented |
| Signed-request auth, nonce replay protection | Implemented |
| Tool registry (versioned, soft delete) and per-integration exposure | Implemented |
| Chat orchestration with tool loop, conversation locking | Implemented |
| Request and tool idempotency, run lifecycle | Implemented |
| Policy engine | Implemented; one rule installed |
| Gemini adapter | Implemented and wired |
| Anthropic / OpenAI adapters | Present in the tree, not wired |
| Approvals, audit, notification dispatch, knowledge/pgvector, Redis, outbox | Scaffolding only (contract comments / empty packages) |
| Observability (`internal/observability`) | Scaffolding; HTTP/Gin logging only |

---

## Repository layout

```text
.
├── adapters/
│   ├── llm/{gemini,anthropic,openai}/   LLM providers (Gemini is wired)
│   ├── storage/postgres/                Stores: runs, conversations, idempotency, tools, integrations
│   ├── storage/redis/ knowledge/ notification/   scaffolding
├── cmd/assistant/                       Server entry point
├── contracts/                           dispatch.proto, tool.proto, notification.proto (contract notes)
├── deployments/{docker,kubernetes}/     docker-compose for local PostgreSQL; Dockerfile/k8s manifests
├── examples/ecommerce/                  Reference host with a tool gateway (customers, orders, notifications)
├── internal/
│   ├── api/            Route registration
│   ├── authn/          Canonical string, signature + request verification
│   ├── conversation/   Conversation + chat handlers and service
│   ├── enums/ errors/ utils/ config/
│   ├── idempotency/    Request and tool idempotency
│   ├── integration/    Onboarding, enrollment, admin service and handlers
│   ├── middleware/     Signed auth, permission, admin, body limit
│   ├── orchestration/  Orchestrator, tool loop, prompt
│   ├── policy/         Decision engine and rules
│   ├── project/        HTTP client + gateway to the host tool endpoint
│   ├── run/            Run model, state machine, service
│   └── tools/          Registration service, executor, handlers
├── migrations/         001–013 (.up/.down SQL)
├── models/             Shared request/response/domain types
└── tests/              Reserved directories for contract/integration/load/failure tests
```

Verify the current layout with `find . -maxdepth 3 -type d`. A package existing in the tree does not mean it is enabled at runtime; check `cmd/assistant/main.go`.

---

## Configuration and local development

### Prerequisites

- A Go version compatible with `go.mod`.
- Docker and Docker Compose (for local PostgreSQL).
- A Gemini API key.
- A host tool gateway if your workflow invokes tools (the `examples/ecommerce` host works for trying it out).

### Start PostgreSQL

```bash
docker compose -f deployments/docker/docker-compose.yml up -d
# postgres:16 on localhost:5438, user dispatchai / password dispatchai, database dispatch
```

### Configure

Configuration is read with Viper from `.env.dev` by default, `.env.staging` when `ENV=staging`, or `.env.prod` when `ENV=prod`. Process environment variables override file values, and the server also runs with environment variables only if the file is absent. The exact keys are bound by the `models.Config` struct tags; the development file uses:

| Variable (dev file) | Purpose |
|---|---|
| `DATABASE_URL` (or `DATABASE_DSN`) | **Required.** e.g. `postgres://dispatchai:dispatchai@localhost:5438/dispatch?sslmode=disable` |
| `MIGRATION_PATH` | Migration directory (default `migrations`) |
| `MAX_OPEN_CONNS` / `MAX_IDLE_CONNS` | Pool sizes (defaults 50 / 10) |
| `LLM_MODEL`, `MAX_TOKENS`, `TEMPERATURE`, `MAX_CONTEXT_TOKENS` | Model settings (`MAX_TOKENS` must be > 0) |
| `GEMINI_API_KEY`, `GENERATEURL` | Gemini credentials and generate endpoint (a `{model}` placeholder or `…/models` base is supported) |
| `PROJECT_BASE_URL`, `PROJECT_TIME_OUT` | Host tool gateway base URL and timeout (e.g. `60s`; default 10 s) |
| `DISPATCHAI_ONBOARDING_TOKEN` | Admin bearer token for `/v1/admin/*` (empty ⇒ onboarding returns 503) |

The listen address is currently hard-coded to `:8989`; `HTTP_PORT` is not used by `cmd/assistant`. Never commit real values.

### Run

```bash
go mod download
go run cmd/assistant/main.go      # run from the repo root so .env.dev and migrations/ resolve
```

Migrations run automatically at startup. Startup logs print the config file used and `DispatchAI server is listening on :8989`.

### Try the whole flow locally

1. Start PostgreSQL and DispatchAI. 2. Create an integration, generate a key, enroll, activate, and grant permissions (see the [guide](#step-by-step-integration-guide)). 3. Start `examples/ecommerce/cmd` (port 8087) and set `PROJECT_BASE_URL=http://localhost:8087/api/v1/ai`. 4. Register the example tools (`get_customers`, `find_delayed_customers`, `send_notification`). 5. Create a conversation and send a chat request with the signing client.

---

## Testing

```bash
go test ./... -count=1
go vet ./...
git diff --check
```

Database-backed integration tests may require the local PostgreSQL container and an explicit opt-in environment variable; read the test setup before pointing them at a database whose data you care about.

Prioritize coverage for: canonicalization and invalid signatures; integration lifecycle and permission enforcement; enrollment-token expiry/reuse; tool registration, versioning, and disabled/deleted tools; policy denial and malformed tool calls; duplicate request keys and fingerprint mismatches; concurrent requests and conversation locking; run-state transitions; persistence errors at completion boundaries; host gateway timeouts and duplicate-side-effect prevention; cross-integration isolation.

A passing unit-test suite does not prove migration correctness, cross-instance locking, or recovery across the DispatchAI/host boundary. Test those against real infrastructure.

---

## Troubleshooting

Signature failures all return `401 invalid request signature`; the precise cause is in the **DispatchAI server log** (`signed request rejected … reason=…`, plus `timestamp diagnostic` and `body hash diagnostic` lines).

| Log reason | Likely cause | Fix |
|---|---|---|
| `header X-… must appear exactly once` / `is malformed` | Missing, duplicated, or badly formatted header (nonce shorter than 16 chars, uppercase hash, etc.) | Send each header once; match the patterns above |
| `request timestamp outside allowed window` | Host clock skew > 5 min, or timestamp in milliseconds | Sync NTP; use Unix **seconds** |
| `signing key not found or inactive` | Wrong `X-Client-Id`/`X-Key-Id`, integration still `pending`/`suspended`, key revoked or expired | Check ids; ask admin to activate |
| `invalid content SHA-256` | Hash computed over different bytes than sent (re-serialized JSON, trailing newline, proxy rewriting) | Hash the exact bytes; compare the `body hash diagnostic` log |
| `invalid request signature` | Canonical string differs (wrong path/query, field order, trailing LF, escaped vs raw path), DER vs raw r‖s signature, wrong key | Rebuild exactly as specified; sign with `ECDSA-P256-SHA256` DER; confirm the enrolled key matches the private key |
| `request replayed` | Nonce reused (including on retries) | New random nonce per attempt |
| `authentication backend unavailable` (503) | Database unreachable | Check PostgreSQL |
| `403 required permission not granted` | Permission missing | Admin grants it |
| `400 valid Idempotency-Key header is required` | Header missing, empty, or > 200 chars | Send a stable key per logical operation |
| `409` on chat | Key reused with a different body, or the original is still running | Use a new key for new content; wait and retry the same key for in-progress |
| `404` on chat | `conversation_id` not created by this integration | Create it via `/v1/conversation/conversation` |
| Chat answer reports tool failure `PROJECT_CLIENT_ERROR` | Gateway unreachable, non-2xx, timeout, or invalid JSON | Check `PROJECT_BASE_URL`, host logs, timeouts |
| Enrollment `401 token invalid, expired, or already used` | Token older than 15 min, reused, or integration not `pending` | Admin creates a new integration |
| Enrollment `400` | Public key is not Base64 of a valid 65-byte P-256 point, or bad `key_id` | Regenerate using the commands above |
| `409 invalid status transition` / `integration has no active signing key` | Illegal status change / activating before enrolling | Enroll first; follow the lifecycle |

---

## Production checklist

- [ ] The API contract and route/middleware configuration match the host client.
- [ ] Private signing keys, the admin token, and enrollment tokens are securely managed.
- [ ] Integration status and permissions are checked on protected requests.
- [ ] The host independently authorizes each user/tenant/resource operation and validates every tool call.
- [ ] The tool gateway endpoint is network-restricted to DispatchAI.
- [ ] Database migrations have been applied and verified.
- [ ] Request and tool idempotency behavior is covered by integration tests.
- [ ] Host operations with side effects are independently idempotent where necessary.
- [ ] LLM, database, and tool calls have bounded timeouts and cancellation behavior.
- [ ] Request size, conversation length, tool-call count, and token usage are bounded.
- [ ] Logs and traces redact secrets and sensitive tool data.
- [ ] Metrics and alerts cover provider failures, request conflicts, stuck runs, lock contention, and persistence failures.
- [ ] Backup, restore, shutdown, and ambiguous-operation recovery procedures are documented.
- [ ] Load, concurrency, and failure-injection tests have been run.

## Known limitations

DispatchAI is evolving. The presence of an abstraction does not by itself prove that a complete workflow is wired end to end. Verify the current implementation before depending on:

- Full durable human-approval and resume workflows (the `REQUIRE_APPROVAL` outcome exists; no approval flow resumes a run).
- Automatic recovery of every interrupted or ambiguous request.
- Exactly-once effects across host systems and third-party APIs.
- Production-grade audit export, tracing, alerting, and retention policies.
- Every provider/adapter being enabled in the active runtime configuration (only Gemini is wired).
- Outbound authentication of tool-gateway calls and forwarding of end-user identity to the host (not provided; see the gateway contract).
- JSON-Schema validation of tool arguments and registry re-checks at execution time (the host must validate).
- Signing-key rotation and revocation APIs (keys are enrolled once while an integration is `pending`).
- Listing an integration's granted permissions through the admin API; use `GET /v1/admin/permissions` for the catalog and the grant response for results.
- Configurable listen port, and a container image/Kubernetes manifests (`Dockerfile` is currently empty).

Track unfinished implementation and integration-test work in issues or a release checklist, and keep this README aligned with the handlers, models, migrations, and deployment configuration.

## Guiding principle

```text
DISPATCHAI
  Understand the user's intent
  Orchestrate the LLM/tool-call loop
  Enforce its integration-level controls
  Manage supported conversation/request/run state
  Return the assistant response

HOST APPLICATION
  Authenticate the user
  Establish trusted identity and tenant context
  Authorize access to each resource
  Enforce business rules
  Execute business operations
  Own transactional data and business truth
```

**DispatchAI does not own the host's business. It orchestrates the capabilities that the host deliberately exposes.**