# Extension Subprocess Protocol Specification

## Status

Draft

## Date

2026-04-10

## Purpose

This document is the **normative wire-level contract** for AGH subprocess extensions.
It complements:

- `_techspec.md` for architecture, package model, and Host API inventory
- `_examples.md` for illustrative flows and SDK usage
- ADR-003, ADR-004, and ADR-005 for security, lifecycle, and package-model decisions

If another document conflicts with this file on **transport framing, lifecycle, handshake fields, error codes, or method-direction semantics**, this file wins.

This specification applies to **persistent extension subprocesses** managed by the Extension Manager. It does **not** describe the legacy one-shot hook subprocess executor used by `internal/hooks/executor_subprocess.go`.

---

## 1. Transport

### 1.1 Base transport

- The protocol uses **JSON-RPC 2.0** over the subprocess `stdin`/`stdout` streams.
- Messages are encoded as **UTF-8 JSON**, **one JSON object per line**.
- `stdout` is reserved for protocol frames only.
- Human-readable logs and diagnostics must go to `stderr`.
- Blank lines on `stdout` must be ignored.
- JSON-RPC batch requests are **not supported** in v1.
- Method names beginning with `rpc.` are reserved and must not be used.
- **JSON-RPC notifications** (requests without an `id` field) are **not supported** in v1. All messages must be requests or responses with an `id`. Receivers must ignore notifications silently.
- The transport is **fully multiplexed**. Both peers may have multiple outstanding requests simultaneously. Responses may arrive in any order. Peers must correlate responses by `id`.

### 1.2 Framing rules

- Each line must contain exactly one JSON-RPC request, response, or notification object.
- Peers must ignore unknown fields for forward compatibility.
- Per-message encoded size must not exceed **10 MiB**. Messages exceeding this limit must be rejected; the receiver should close the transport connection.

### 1.3 Request identifiers

- AGH will use positive integer IDs.
- Extensions may use positive integer IDs or string IDs.
- Fractional numeric IDs must not be used.

### 1.4 Time encoding

- All timestamps are serialized as RFC3339Nano UTC strings, matching Go `time.Time` JSON encoding.

### 1.5 JSON encoding rules

- Struct fields tagged `omitempty` are omitted when zero-valued.
- Fields represented as `json.RawMessage` on the Go side must be serialized as embedded JSON values, not quoted strings.
- Unknown object members must be ignored unless the receiving method explicitly forbids them.

---

## 2. Roles and Method Directions

AGH is the **connection initiator** because it launches the subprocess, but after initialization the transport is **bidirectional peer-to-peer JSON-RPC**. Either side may originate requests.

### 2.1 Method families

| Direction | Family | Canonical names |
|---|---|---|
| AGH -> Extension | Base lifecycle methods | `initialize`, `execute_hook`, `health_check`, `shutdown`, `provide_tools` |
| Extension -> AGH | Host API actions | `sessions/*`, `memory/*`, `observe/*`, `skills/*` |
| AGH -> Extension | Extension service methods | Capability-specific methods such as `memory/store`, `memory/recall`, `memory/forget` when the extension provides `memory.backend` |

### 2.2 Naming conventions

- Base lifecycle methods use **snake_case**.
- Host API and extension service methods use **slash-separated** RPC names.
- Hook event names use **dotted** identifiers such as `turn.start` and `tool.pre_call`.

### 2.3 Direction disambiguates ownership

Some method names may appear in both directions.

Example:

- `memory/store` sent **AGH -> Extension** means "invoke the extension's `memory.backend` implementation"
- `memory/store` sent **Extension -> AGH** means "call AGH's Host API memory store"

This is valid because the transport is bidirectional. SDKs must expose these surfaces separately so that implementers do not confuse:

- `host.memory.store(...)`
- `extension.handle("memory/store", ...)`

---

## 3. Connection Lifecycle

The connection lifecycle has five phases:

1. **Spawn**: AGH starts the extension process and connects `stdin`/`stdout`.
2. **Initialize**: AGH sends `initialize`; the extension accepts or rejects the session contract.
3. **Ready**: Both peers may exchange operational requests.
4. **Draining**: A shutdown has been initiated; no new work should be accepted.
5. **Stopped**: The process exits and the transport closes.

### 3.1 Pre-ready rules

- Before `initialize` succeeds, the only valid request is `initialize`.
- Any other request before readiness must fail with `-32003 not_initialized`.
- AGH must not route hooks, Host API actions, or capability service calls before readiness.

### 3.2 Ready transition

There is **no separate `initialized` notification in v1**.
The connection enters **Ready** immediately after:

1. `initialize` returns success
2. AGH verifies the selected protocol version
3. AGH verifies the returned capability/method contract is a subset of the granted contract

### 3.3 Restart semantics

When an extension subprocess is restarted (due to crash recovery, manual re-enable, or daemon restart):

- A fresh subprocess is spawned. There is **no connection resumption** in v1.
- AGH must send a new `initialize` handshake from scratch.
- The extension must not assume any state from a previous session persists.
- AGH may re-register the extension's resources (hooks, skills) during the new initialization.

### 3.4 Draining rules

- After `shutdown` starts, the peer must stop accepting new operational requests.
- New requests during draining must fail with `-32004 shutdown_in_progress`.
- Responses for already accepted in-flight requests may still be delivered until the process exits.

---

## 4. Initialize Handshake

The `initialize` handshake establishes:

- protocol version compatibility
- runtime grants derived from the manifest and source-tier policy
- the method surfaces that may be used in this session
- runtime intervals and deadlines

### 4.1 Initialize request

AGH must send `initialize` as the first request.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocol_version": "1",
    "supported_protocol_versions": ["1"],
    "agh_version": "0.5.0",
    "extension": {
      "name": "pgvector-memory",
      "version": "0.2.0",
      "source_tier": "user"
    },
    "capabilities": {
      "provides": ["memory.backend"],
      "granted_actions": ["sessions/list", "sessions/events"],
      "granted_security": ["memory.read", "memory.write", "session.read"]
    },
    "methods": {
      "daemon_requests": ["execute_hook", "health_check", "shutdown"],
      "extension_services": ["memory/store", "memory/recall", "memory/forget"]
    },
    "runtime": {
      "health_check_interval_ms": 30000,
      "health_check_timeout_ms": 5000,
      "shutdown_timeout_ms": 10000,
      "default_hook_timeout_ms": 5000
    }
  }
}
```

### 4.2 Initialize request fields

| Field | Type | Required | Meaning |
|---|---|---|---|
| `protocol_version` | string | yes | AGH's preferred protocol version |
| `supported_protocol_versions` | array<string> | yes | Ordered list of versions AGH can speak |
| `agh_version` | string | yes | Daemon semver version string for diagnostics and compatibility checks (informational only) |
| `extension.name` | string | yes | Manifest name AGH loaded |
| `extension.version` | string | yes | Manifest version AGH loaded |
| `extension.source_tier` | string | yes | Source trust tier such as `bundled`, `user`, `workspace`, or `marketplace` |
| `capabilities.provides` | array<string> | yes | Capability interfaces AGH expects this extension to provide |
| `capabilities.granted_actions` | array<string> | yes | Host API methods this connection is authorized to call |
| `capabilities.granted_security` | array<string> | yes | Security grants enforced at dispatch and Host API boundaries |
| `methods.daemon_requests` | array<string> | yes | Base AGH -> extension methods available for this session |
| `methods.extension_services` | array<string> | yes | Capability service methods AGH may call on the extension |
| `runtime.health_check_interval_ms` | integer | yes | Periodic probe interval |
| `runtime.health_check_timeout_ms` | integer | yes | Per-probe timeout |
| `runtime.shutdown_timeout_ms` | integer | yes | Graceful shutdown deadline before signal escalation |
| `runtime.default_hook_timeout_ms` | integer | yes | Default timeout when a hook declaration omits one |

### 4.3 Initialize response

The extension must answer with the selected version and the accepted session contract.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocol_version": "1",
    "extension_info": {
      "name": "pgvector-memory",
      "version": "0.2.0",
      "sdk_name": "@agh/extension-sdk",
      "sdk_version": "0.1.0"
    },
    "accepted_capabilities": {
      "provides": ["memory.backend"],
      "actions": ["sessions/list", "sessions/events"],
      "security": ["memory.read", "memory.write", "session.read"]
    },
    "implemented_methods": ["memory/store", "memory/recall", "memory/forget", "health_check", "shutdown"],
    "supported_hook_events": ["prompt.post_assemble", "turn.start", "turn.end"],
    "supports": {
      "health_check": true,
      "provide_tools": false
    }
  }
}
```

### 4.4 Initialize response rules

- `protocol_version` must be one of the versions AGH offered.
- `accepted_capabilities.actions` must be a subset of `capabilities.granted_actions`.
- `accepted_capabilities.security` must be a subset of `capabilities.granted_security`.
- `accepted_capabilities.provides` must be a subset of `capabilities.provides`.
- `implemented_methods` must include every method required by the accepted `provides` contract.
- `supported_hook_events` must not advertise events outside AGH's known hook taxonomy.
- `supports.provide_tools=false` means AGH must treat `provide_tools` as unavailable for the session.

### 4.5 Capability negotiation semantics

Capability negotiation happens in two stages:

1. **Static declaration** in the manifest:
   - `capabilities.provides`
   - `actions.requires`
   - `security.capabilities`
2. **Runtime grant** in `initialize`:
   - AGH applies source-tier policy and startup validation
   - AGH sends the effective grants in the request
   - the extension either accepts them or rejects the session

If the extension requires capabilities that were not granted, it must reject initialization with `-32001 capability_denied`.

Example error:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32001,
    "message": "Capability denied",
    "data": {
      "missing_actions": ["sessions/events"],
      "missing_security": ["memory.write"]
    }
  }
}
```

### 4.6 Generic initialization failure

If the extension cannot initialize for application-level reasons (e.g., database unreachable, missing config), it must return `-32603 internal error` with structured `data`:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": {
      "reason": "database_unreachable",
      "detail": "Failed to connect to pgvector at localhost:5432"
    }
  }
}
```

### 4.7 Version mismatch

Unsupported protocol versions must use standard JSON-RPC `-32602 invalid params`, following the same pattern MCP uses for initialization failures.

Example:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "reason": "unsupported_protocol_version",
      "requested": "2",
      "supported_protocol_versions": ["1"]
    }
  }
}
```

---

## 5. Operational Requests

### 5.1 Base methods

The following AGH -> extension methods are part of the base protocol in v1:

| Method | Required | Purpose |
|---|---|---|
| `execute_hook` | yes | Dispatch one hook invocation with a typed payload |
| `health_check` | yes | Probe liveness/readiness of the running extension |
| `shutdown` | yes | Begin graceful drain and exit |
| `provide_tools` | optional | Request tool definitions when negotiated |

### 5.2 Host API methods

The canonical Host API method inventory (Extension -> AGH):

| Method | Capability |
|---|---|
| `sessions/list` | `session.read` |
| `sessions/create` | `session.write` |
| `sessions/prompt` | `session.write` |
| `sessions/stop` | `session.write` |
| `sessions/status` | `session.read` |
| `sessions/events` | `session.read` |
| `memory/recall` | `memory.read` |
| `memory/store` | `memory.write` |
| `memory/forget` | `memory.write` |
| `observe/health` | `observe.read` |
| `observe/events` | `observe.read` |
| `skills/list` | `skills.read` |

See `_techspec.md` Host API section for parameter and result schemas.

This protocol file adds the normative rules around:

- authorization: every call checked against `granted_actions` (method-level) AND `granted_security` (family-level). Both must be satisfied. `granted_actions` is the fine-grained allowlist; `granted_security` is the coarse-grained family gate.
- error codes: unauthorized calls return `-32001 capability_denied`
- timeout behavior: Host API calls use the daemon's default request timeout
- rate limiting: per-extension rate limits return `-32002 rate_limited`
- startup gating: calls before `initialize` return `-32003 not_initialized`
- shutdown gating: calls during drain return `-32004 shutdown_in_progress`

### 5.3 Capability service methods

Capability service methods are AGH -> extension requests enabled by `capabilities.provides`.
In v1, the only normatively grounded service surface is the memory backend family shown in `_techspec.md` and `_examples.md`:

- `memory/store`
- `memory/recall`
- `memory/forget`

The wire framing, timeouts, and error rules for those calls are identical to any other operational JSON-RPC request.

### 5.4 `provide_tools` (optional)

When an extension declares `supports.provide_tools: true` during initialization, AGH may call `provide_tools` to request tool definitions.

**Request:**
```json
{"jsonrpc":"2.0","id":10,"method":"provide_tools","params":{}}
```

**Response:**
```json
{"jsonrpc":"2.0","id":10,"result":{
  "tools":[
    {
      "name":"pgvector_search",
      "description":"Semantic search over stored memories",
      "input_schema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]},
      "read_only":true
    }
  ]
}}
```

Tool definitions follow the `Tool` struct defined in `_techspec.md`. AGH may cache the result and re-request periodically or after extension restart.

---

## 6. Hook Dispatch: `execute_hook`

`execute_hook` is the canonical AGH -> extension hook invocation method.

### 6.1 Request shape

```json
{
  "jsonrpc": "2.0",
  "id": 42,
  "method": "execute_hook",
  "params": {
    "invocation_id": "hook-01JRFV8A2M0N6H7R6P6D7M0E7F",
    "hook": {
      "name": "workspace-context",
      "event": "prompt.post_assemble",
      "mode": "sync",
      "required": false,
      "timeout_ms": 5000,
      "source": "extension",
      "metadata": {
        "extension_name": "prompt-enhancer"
      }
    },
    "payload": {
      "event": "prompt.post_assemble",
      "timestamp": "2026-04-10T14:03:00.123456Z",
      "session_id": "sess_123",
      "turn_id": "turn_456",
      "prompt": "Explain the current failing test.",
      "context_blocks": []
    }
  }
}
```

### 6.2 Request fields

| Field | Type | Required | Meaning |
|---|---|---|---|
| `invocation_id` | string | yes | Opaque identifier for one hook invocation. Extensions must not parse or rely on its internal structure. |
| `hook.name` | string | yes | Resolved hook declaration name |
| `hook.event` | string | yes | Canonical hook event name |
| `hook.mode` | `sync` or `async` | yes | Dispatch mode selected by AGH |
| `hook.required` | boolean | yes | Whether a failure blocks the pipeline |
| `hook.timeout_ms` | integer | yes | Effective timeout AGH will enforce for this invocation |
| `hook.source` | string | yes | Human-readable source label for telemetry and diagnostics |
| `hook.metadata` | map&lt;string, string&gt; | no | Optional extension-specific key-value metadata copied from declaration/runtime. Values are always strings. |
| `payload` | object | yes | Event-specific payload object |

### 6.3 Response shape

```json
{
  "jsonrpc": "2.0",
  "id": 42,
  "result": {
    "patch": {
      "prompt": "Explain the current failing test. Also mention the workspace README."
    }
  }
}
```

### 6.4 Response rules

- A successful response must return an object.
- `{}` means **no-op**.
- `{"patch": {}}` also means **no-op**.
- `patch` must match the event's patch schema.
- If the extension has nothing to change, it should prefer `{}`.

### 6.5 Hook payload serialization

AGH serializes hook payloads exactly according to the JSON field names already defined by the runtime hook types.

#### Event matrix

| Event(s) | Payload schema | Patch schema | Sync eligible | Mutable |
|---|---|---|---|---|
| `session.pre_create` | `SessionPreCreatePayload` | `SessionCreatePatch` | yes | yes |
| `session.post_create`, `session.pre_resume`, `session.post_resume`, `session.pre_stop`, `session.post_stop` | `SessionLifecyclePayload` | `SessionCreatePatch` | yes | observe-only |
| `input.pre_submit` | `InputPreSubmitPayload` | `InputPreSubmitPatch` | yes | yes |
| `prompt.post_assemble` | `PromptPayload` | `PromptPatch` | yes | yes |
| `event.pre_record`, `event.post_record` | `EventRecordPayload` | `EventRecordPatch` | no | observe-only |
| `agent.pre_start` | `AgentPreStartPayload` | `AgentStartPatch` | yes | yes |
| `agent.spawned`, `agent.crashed`, `agent.stopped` | `AgentLifecyclePayload` | `AgentLifecyclePatch` | yes | observe-only |
| `turn.start`, `turn.end` | `TurnPayload` | `TurnPatch` | yes | observe-only |
| `message.start` | `MessagePayload` | `MessagePatch` | yes | yes |
| `message.delta` | `MessagePayload` | `MessagePatch` | no | observe-only |
| `message.end` | `MessagePayload` | `MessagePatch` | yes | yes |
| `tool.pre_call` | `ToolPreCallPayload` | `ToolCallPatch` | yes | yes |
| `tool.post_call` | `ToolPostCallPayload` | `ToolResultPatch` | yes | yes |
| `tool.post_error` | `ToolPostErrorPayload` | `ToolResultPatch` | yes | yes |
| `permission.request` | `PermissionRequestPayload` | `PermissionRequestPatch` | yes | yes |
| `permission.resolved` | `PermissionResolutionPayload` | `PermissionResolvedPatch` | no | observe-only |
| `permission.denied` | `PermissionResolutionPayload` | `PermissionDeniedPatch` | no | observe-only |
| `context.pre_compact`, `context.post_compact` | `ContextCompactPayload` | `ContextCompactionPatch` | yes | yes |

**Mutable** = returned patches are applied to the live pipeline payload. **Observe-only** = patches are accepted but only recorded for telemetry; they do not mutate the pipeline. Extensions returning patches for observe-only events should expect no visible effect.

### 6.6 Sync versus async semantics

- **Sync hooks** are on the request's critical path. AGH waits for the JSON-RPC response and may apply the returned patch.
- **Async hooks** are out of band. AGH may dispatch them only after the sync phase has succeeded.
- Returned patches from **async hooks are ignored for live mutation in v1**. AGH may retain them for telemetry/debug only.
- Async hook failures must not block the originating runtime event.
- If AGH cannot enqueue an async hook locally because of backpressure, the invocation is dropped locally and no `execute_hook` request is sent.

### 6.7 Deny semantics

Patch types that embed a `deny` / `deny_reason` surface may block an operation only when:

- the invocation is `sync`
- the event is semantically blockable

Invalid deny attempts are treated as patch rejection by AGH.

### 6.8 Failure semantics

- If a **required sync hook** returns a JSON-RPC error or times out, AGH must fail the pipeline.
- If a **non-required sync hook** fails, AGH records the failure and continues.
- If a patch is structurally valid JSON but semantically invalid for that event, AGH marks it as `rejected`.
- Patch rejection is a daemon-side semantic outcome, not a second JSON-RPC error response.

---

## 7. Health Protocol: `health_check`

`health_check` is the AGH-specific liveness/readiness probe used for persistent extension subprocesses. v1 keeps this method instead of adopting MCP `ping` because AGH needs structured health state, not only round-trip reachability.

### 7.1 Request

```json
{
  "jsonrpc": "2.0",
  "id": 90,
  "method": "health_check",
  "params": {}
}
```

### 7.2 Response

```json
{
  "jsonrpc": "2.0",
  "id": 90,
  "result": {
    "healthy": true,
    "message": "",
    "details": {
      "active_requests": 0,
      "queue_depth": 0
    }
  }
}
```

### 7.3 Response fields

| Field | Type | Required | Meaning |
|---|---|---|---|
| `healthy` | boolean | yes | Whether the extension considers itself ready to serve requests |
| `message` | string | no | Human-readable summary for diagnostics |
| `details` | object | no | Optional structured metrics such as queue depth or active requests |

### 7.4 Probe policy

- Default interval is the manifest's `health_check_interval`, or **30s** if omitted.
- Default timeout is **5s**.
- A transport timeout, disconnect, or JSON-RPC error counts as a failed probe.
- `healthy: false` counts as a failed probe and includes the extension's self-reported reason.

### 7.5 Unhealthy threshold

AGH marks the extension **unhealthy** when either condition occurs:

1. one successful response explicitly returns `healthy: false`
2. two consecutive probes fail because of timeout, disconnect, or JSON-RPC error

When an extension becomes unhealthy, AGH must:

1. stop routing new requests to it
2. log the failure
3. begin shutdown/restart recovery as defined by the Extension Manager

---

## 8. Graceful Shutdown: `shutdown`

`shutdown` is AGH's cooperative drain request. It exists in addition to OS signals.

### 8.1 Request

```json
{
  "jsonrpc": "2.0",
  "id": 99,
  "method": "shutdown",
  "params": {
    "reason": "daemon_shutdown",
    "deadline_ms": 10000
  }
}
```

### 8.2 Response

```json
{
  "jsonrpc": "2.0",
  "id": 99,
  "result": {
    "acknowledged": true
  }
}
```

### 8.3 Shutdown rules

- The extension must answer `shutdown` promptly.
- After answering, it must stop accepting new operational requests.
- It may complete in-flight work until `deadline_ms` expires.
- After the `shutdown` response is received, AGH should close the extension's `stdin` to signal that no more requests will arrive.
- It should then close its protocol streams and exit cleanly with status `0`.

### 8.4 Signal escalation

If the process does not exit after the cooperative shutdown deadline:

1. AGH ensures the extension's `stdin` is closed.
2. AGH sends `SIGTERM` to the managed process group on Unix, or the platform-equivalent process termination on Windows.
3. AGH waits a short post-signal grace period.
4. If the process is still alive, AGH sends `SIGKILL` on Unix, or the platform-equivalent forced termination on Windows.

### 8.5 Default timing

- Default graceful shutdown deadline is the manifest's `shutdown_timeout`, or **10s** if omitted.
- The post-`SIGTERM` grace period is implementation-defined but should be short and bounded.

---

## 9. Error Model

The protocol uses JSON-RPC 2.0 error objects.

### 9.1 Standard JSON-RPC errors

| Code | Message | Use |
|---|---|---|
| `-32700` | `Parse error` | Invalid JSON on the wire |
| `-32600` | `Invalid request` | Invalid JSON-RPC envelope |
| `-32601` | `Method not found` | The receiving peer does not implement the method |
| `-32602` | `Invalid params` | Invalid method parameters, including unsupported protocol version during `initialize` |
| `-32603` | `Internal error` | Unhandled receiver-side failure |

### 9.2 AGH-defined server errors

| Code | Message | Use |
|---|---|---|
| `-32001` | `Capability denied` | Method/event/security grant not authorized for this session |
| `-32002` | `Rate limited` | Local backpressure or explicit per-extension rate limit |
| `-32003` | `Not initialized` | Request arrived before successful `initialize` |
| `-32004` | `Shutdown in progress` | Receiver is draining and will not accept new work |

### 9.3 `Method not found` versus `Capability denied`

Use `-32601 method not found` when:

- the receiver does not recognize the method string at all
- the method is optional and was never implemented on that peer

Use `-32001 capability denied` when:

- the method exists, but the caller was not granted that action
- the hook/event family exists, but was not negotiated for this session
- the source-tier policy removed the grant even though the manifest requested it

### 9.4 Error data

Errors should include structured `data` when helpful.

#### Capability denied

```json
{
  "code": -32001,
  "message": "Capability denied",
  "data": {
    "method": "sessions/create",
    "required": ["session.write"],
    "granted": ["session.read"]
  }
}
```

#### Rate limited

```json
{
  "code": -32002,
  "message": "Rate limited",
  "data": {
    "scope": "host_api.sessions/create",
    "retry_after_ms": 1000,
    "limit": 10,
    "burst": 20
  }
}
```

#### Not initialized

```json
{
  "code": -32003,
  "message": "Not initialized",
  "data": {
    "allowed_methods": ["initialize"]
  }
}
```

#### Shutdown in progress

```json
{
  "code": -32004,
  "message": "Shutdown in progress",
  "data": {
    "deadline_ms": 10000
  }
}
```

### 9.5 Transport failures versus JSON-RPC errors

The following are **transport failures**, not JSON-RPC error responses:

- peer disconnects before a response arrives
- probe/request timeouts
- OS-level process termination

Callers must treat these as failed requests and apply the lifecycle/recovery rules from this specification.

---

## 10. Rate Limiting and Backpressure

AGH may protect Host API surfaces with per-extension rate limits.

### 10.1 Receiver behavior

- When a peer is willing to reject and retry later, it should return `-32002 rate_limited`.
- `data.retry_after_ms` should be present whenever the receiver can estimate a retry delay.

### 10.2 Caller behavior

- Callers should not immediately retry a `rate_limited` request.
- SDKs should expose `retry_after_ms` to extension authors.

### 10.3 Async hook backpressure

AGH's internal async hook queue is local implementation detail, but v1 defines the observable contract:

- queue saturation before wire send results in a **local drop**
- a local drop does not generate a JSON-RPC request
- a local drop should be recorded as hook outcome `dropped`

---

## 11. Protocol Versioning

### 11.1 Version token

- v1 uses the exact string `"1"`.
- Protocol versions are exact-match string tokens, not numeric comparisons.

### 11.2 Negotiation

- AGH sends its preferred version in `protocol_version`.
- AGH also sends all supported versions in `supported_protocol_versions`.
- The extension must either:
  - return a supported `protocol_version` in the response
  - or reject initialization with `-32602 invalid params` and include `supported_protocol_versions`

### 11.3 Forward compatibility

Within the same protocol version:

- receivers must ignore unknown fields
- optional fields may be added
- new optional methods may be added if they are negotiated explicitly during initialization

A new protocol version is required when:

- a required field is removed or renamed
- method semantics change incompatibly
- an existing success/error contract changes incompatibly

### 11.4 AGH version versus protocol version

`agh_version` and `protocol_version` are separate:

- `agh_version` identifies the daemon build
- `protocol_version` identifies the subprocess wire contract

Extensions must not infer protocol compatibility from `agh_version` alone.

---

## 12. Conformance Rules

An extension is v1-conformant only if it satisfies all of the following:

- speaks JSON-RPC 2.0 over line-delimited UTF-8 JSON on `stdin`/`stdout`
- emits protocol frames only on `stdout`
- implements `initialize`, `health_check`, and `shutdown`
- implements `execute_hook` if it accepts hook execution
- honors negotiated action/security grants
- returns standard JSON-RPC errors for envelope/params failures
- returns AGH custom errors for capability/rate-limit/lifecycle gating failures
- exits cooperatively after `shutdown`, or tolerates signal escalation

AGH is v1-conformant only if it satisfies all of the following:

- sends `initialize` first
- never routes operational requests before readiness
- enforces granted actions/security at the Host API boundary
- enforces hook dispatch deadlines
- marks extensions unhealthy according to this spec
- performs cooperative shutdown before signal escalation

---

## 13. Notes for Follow-up Docs

This file resolves several ambiguities that should later be aligned back into `_techspec.md` and `_examples.md`:

- persistent extension subprocesses are distinct from the one-shot hook subprocess executor
- method direction determines ownership when names overlap, especially `memory/*`
- async `execute_hook` responses do not mutate live runtime payloads in v1
- unsupported protocol versions use `-32602 invalid params`, not a custom code
