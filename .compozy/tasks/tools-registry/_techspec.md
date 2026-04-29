# Tool Registry Foundation TechSpec

## Executive Summary

This TechSpec designs AGH's Tool Registry as a daemon-owned runtime service, not as a static list of built-in commands. The registry will unify tool identity, discovery, availability, policy, execution, hooks, telemetry, extension descriptors, MCP adapters, and session-visible exposure through one central dispatch pipeline.

There is no existing `_prd.md` for this task. The scope is based on the user request, competitor research under `.compozy/tasks/tools-registry/analysis/`, accepted ADRs under `.compozy/tasks/tools-registry/adrs/`, current AGH code exploration, and the prior autonomous skills/tools registry gap analysis.

The primary architectural trade-off is to make AGH-native tools visible to sessions through an AGH-hosted local MCP server in the MVP instead of trying to invent an ACP-specific registry. ACP does not define a callable tool registry; it defines session lifecycle, MCP bootstrap, permission callbacks, and tool-call observations. MCP provides the programmatic `Tool.name`, so AGH will expose its canonical `ToolID` directly as the hosted MCP tool name.

The foundation will support executable native/bundled tools, executable extension-host tools, and executable MCP-backed tools. Built-in AGH tools execute in-process through `native_go` handles compiled into the daemon. Third-party TypeScript and Go extension tools execute out-of-process through the existing extension runtime, a new `tool.provider` capability, `provide_tools` reconciliation, and `tools/call` RPC. MCP-backed tools execute through daemon-owned MCP clients that consume the existing MCP config/auth subsystem. Descriptor-only is an unavailable/error state, not the MVP contract for extension or MCP tools.

Implementation direction: AGH should use `github.com/mark3labs/mcp-go` for MCP protocol/session/transport behavior instead of planning hand-rolled MCP JSON-RPC, stdio, SSE, or streamable-HTTP plumbing inside the Tool Registry workstream. AGH still owns registry policy, canonical `ToolID`, approval bridging, UDS/session binding, provenance, redaction, and durable MCP auth state.

## MVP Boundary Statement

MVP boundary: implementation steps 1-16 build the Tool Registry foundation, AGH-hosted MCP session exposure, native bootstrap tools, executable TypeScript/Go extension-host tools, executable daemon-owned MCP call-through, shared CLI/HTTP/UDS surfaces, policy/availability enforcement, hooks, observability, docs, SDK updates, and verification. This MVP proves the registry as an executable daemon primitive without replacing every ACP provider-native tool.

Post-MVP work deferred to later TechSpecs:

- direct driver-specific tool injection outside hosted MCP;
- full shell/browser/file tool replacement for ACP runtimes;
- remote peer tool execution over AGH Network;
- provider-specific deferred schema loading such as Anthropic `tool_reference`;
- broad marketplace signing/trust overhaul;
- skill install/remove/update tools;
- bridge SDK executable tool adapters;
- direct in-process plugin loading for third-party Go or TypeScript code;
- client-supplied ACP `mcpServers` as session-scoped registry sources.

Explicitly out of scope for this TechSpec:

- in-process third-party extension handlers;
- silent compatibility aliases for dotted tool IDs;
- policy bypasses for `approve-all`;
- partial surface delivery where CLI/HTTP ships without UDS, docs, codegen, and hosted MCP parity;
- storing matchable ownership or authorization state inside opaque JSON metadata blobs.

Backend delivery boundary:

| Backend kind | MVP delivery | Invocation behavior |
|---|---|---|
| `native_go` | Descriptor, availability, policy, and full dispatch through `Registry.Call` | Executable in-process only for daemon-compiled AGH built-ins |
| `extension_host` | Manifest-authoritative descriptor, runtime reconciliation, extension health, source policy, and full dispatch through `Registry.Call` | Executable out-of-process through existing extension subprocess runtime, `tool.provider`, and `tools/call`; TypeScript and Go SDKs wrap handlers as functions |
| `mcp` | Descriptor discovery, source provenance, health/auth diagnostics, collision handling, source policy, and full dispatch through `Registry.Call` | Executable through daemon-owned MCP clients using existing MCP config and `internal/mcp/auth` status/token interfaces |
| `subprocess` | Not a public registry backend kind in MVP | Rejected by manifest validation; third-party Go/TypeScript tools use `extension_host`, whose implementation is subprocess-isolated |
| `bridge` | Reserved post-MVP backend kind | Rejected by MVP validation unless a later bridge TechSpec enables it |

AGH-hosted MCP is different from `mcp` backend tools. Hosted MCP is the session exposure transport for AGH registry tools; `mcp` backend tools are external tools contributed by MCP servers and are executable only through daemon-owned MCP client adapters after the same registry policy, source, approval, hook, and session-lineage gates pass.

## Architectural Boundaries

`internal/daemon` remains the only composition root. It may import and compose `internal/tools`, `internal/skills`, `internal/mcp`, `internal/extension`, `internal/hooks`, `internal/session`, `internal/network`, `internal/task`, `internal/api/*`, and config/resource stores. No package may import `internal/daemon`.

Package import boundaries:

- `internal/tools` owns `ToolID`, descriptors, backend kinds, providers, handles, registry, policy interfaces, availability, dispatch contracts, and result normalization. It must not import `internal/daemon`, `internal/api/*`, `internal/cli`, `internal/extension`, `internal/session`, `internal/network`, or `internal/task`.
- `internal/catalog`, if added, is a thin composition-facing facade over `internal/tools` and `internal/skills`. It must not own tool dispatch or policy.
- `internal/extension` may publish manifest-authoritative tool descriptors and expose live out-of-process extension tool invokers through public registry contracts. It must not execute third-party tool handlers in-process and must not import registry internals beyond public `internal/tools` descriptor/provider contracts.
- `internal/mcp` may adapt external MCP tools, call external MCP servers through daemon-owned clients, and host the AGH MCP stdio proxy. It wraps `mark3labs/mcp-go` behind AGH-owned interfaces and must keep all AGH-owned calls entering `internal/tools.Registry.Call` through UDS or an injected interface; MCP code must not duplicate dispatch policy.
- `internal/mcp/auth` already owns remote MCP OAuth 2.1 + PKCE, redacted status, token refresh/logout, and durable token storage through `internal/store/globaldb`. The Tool Registry may consume redacted auth status through a daemon-injected interface, but it must not reimplement OAuth flows, open the MCP auth token store directly, or persist remote MCP token material.
- `internal/api/core` owns transport-independent handlers. `internal/api/httpapi` and `internal/api/udsapi` only register routes and transport concerns.
- `internal/cli` calls UDS/HTTP client methods and does not import runtime registry implementations.
- `internal/hooks` owns typed hook payloads and execution. Hooks dispatch at the registry call site; no code may tail event tables to trigger tool hooks.
- `internal/store` may persist session lineage permission atoms and events. It must not decide tool policy; it validates and stores normalized atoms.
- `internal/session` can receive hosted MCP server config and session projections through interfaces. It must not implement a parallel tool registry.

Boundaries to update in implementation:

- If `internal/catalog` or a new `internal/mcp` subpackage is added, update `magefile.go` package boundary checks in the same change.
- Any OpenAPI/contract change must co-ship generated `openapi/agh.json` and `web/src/generated/agh-openapi.d.ts`.
- Any CLI surface must co-ship HTTP/UDS parity unless the spec explicitly marks the surface CLI-only. This TechSpec has no CLI-only tool surfaces.

## System Architecture

### Component Overview

| Component | Responsibility | Boundary |
|---|---|---|
| `internal/tools` runtime registry | Owns canonical `ToolID`, descriptors, backend kinds, providers, handles, availability projection, policy evaluation, dispatch, result normalization, and telemetry hooks | Does not import `daemon`, `api`, or `cli`; composed by `internal/daemon` |
| Cold `tool` resources | Persist desired-state tool metadata from extensions, bundles, and future dynamic producers | Metadata only; no function pointers or execution handles |
| Built-in `native_go` provider | Registers AGH-native tools such as tool search, skill view, network peers/send, and bounded task tools | In-process only because it ships inside the daemon binary |
| Extension-host provider | Converts extension-published tool resources into runtime descriptors, reconciles them with `provide_tools`, and invokes handlers over extension subprocess RPC | Does not execute extension code in-process; fails closed on manifest/runtime mismatch |
| Extension SDKs | TypeScript and Go helper APIs let extension authors define tools with functions while the runtime remains out-of-process | SDKs generate/reconcile manifest descriptors and register `tools/call` handlers |
| MCP adapter provider | Normalizes MCP-backed descriptors, health, auth status, source provenance, and executes calls through daemon-owned MCP clients wrapping `mark3labs/mcp-go` | Fails closed on health, auth, policy, approval, hook, or name collision problems |
| Existing MCP auth runtime | Supplies redacted remote MCP OAuth 2.1 + PKCE status for external MCP diagnostics | Owned by `internal/mcp/auth`; registry must not duplicate OAuth flow, token storage, or token refresh/logout |
| `internal/catalog` facade | Thin cross-domain list/search/view facade over tools and skills | Optional coordination layer; runtime tool dispatch remains in `internal/tools` |
| Policy engine | Combines ACP approval mode, session lineage, agent policy, source/risk defaults, registry allow/deny, toolsets, availability, and hooks | Produces structured effective decisions, never a single ambiguous boolean |
| AGH-hosted MCP proxy | Exposes session-callable AGH tools as MCP tools using canonical `ToolID` names | Runs through daemon-provided `agh tool mcp --session <id> --bind-nonce <nonce>`, uses `mark3labs/mcp-go` server/tool APIs over stdio, and proxies to daemon over UDS |
| API/CLI surfaces | Expose machine-readable list/search/info/invoke/status behavior | Shared contracts in `internal/api/contract`, handlers in `internal/api/core`, HTTP/UDS parity |
| Hook integration | Runs `tool.pre_call`, `tool.post_call`, and `tool.post_error` around registry dispatch | Hooks can deny, narrow, patch, redact, or annotate, but cannot bypass policy |
| Observability | Emits durable events and metrics for registration, projection, decisions, calls, failures, conflicts, truncation, and policy denials | Redacts secrets and raw tokens |

Data flow:

1. Extensions, built-ins, MCP servers, and future dynamic providers register cold descriptors and, where executable, runtime handles.
2. `internal/daemon` composes providers into `internal/tools.Registry`.
3. Registry indexes descriptors by canonical `ToolID`.
4. Operator surfaces can list all tools with status and reason codes.
5. Session/model-visible surfaces request a session projection and receive only callable tools for that effective context.
6. Every invocation enters `Registry.Call`, which validates schema, recomputes availability/policy, runs hooks, calls the `native_go`, `extension_host`, or `mcp` handle, normalizes output, persists/observes, and returns a bounded result.

## Implementation Design

### Core Interfaces

These are final-shape interface contracts for implementation planning. Implementers may add helper types, but registry dispatch must preserve these method responsibilities and must not reintroduce metadata-only runtime providers.

```go
type Registry interface {
	List(ctx context.Context, scope Scope) ([]ToolView, error)
	Search(ctx context.Context, scope Scope, q SearchQuery) ([]ToolView, error)
	Get(ctx context.Context, scope Scope, id ToolID) (ToolView, error)
	Call(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error)
}

type Handle interface {
	Descriptor() Descriptor
	Availability(ctx context.Context, scope Scope) Availability
	Call(ctx context.Context, req CallRequest) (ToolResult, error)
}
```

Supporting contracts:

```go
type BackendKind string

const (
	BackendNativeGo      BackendKind = "native_go"
	BackendExtensionHost BackendKind = "extension_host"
	BackendMCP           BackendKind = "mcp"
	BackendBridge        BackendKind = "bridge"
)
```

```go
type Provider interface {
	ID() SourceRef
	List(ctx context.Context, scope Scope) ([]Descriptor, error)
	Resolve(ctx context.Context, scope Scope, id ToolID) (Handle, bool, error)
}
```

```go
type NativeToolFunc func(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error)
```

```go
type ExtensionToolInvoker interface {
	ProvideTools(ctx context.Context, extensionID string) ([]ExtensionToolRuntimeDescriptor, error)
	CallTool(ctx context.Context, extensionID string, req ExtensionToolCallRequest) (ToolResult, error)
}
```

```go
type MCPCallExecutor interface {
	ListTools(ctx context.Context, source SourceRef) ([]MCPToolDescriptor, error)
	CallTool(ctx context.Context, source SourceRef, req MCPToolCallRequest) (ToolResult, error)
}
```

`MCPCallExecutor` is implemented by `internal/mcp`, not by `internal/tools`. It resolves bearer material through `internal/mcp/auth` internally, never exposes `mcpauth.TokenRecord` or raw headers to registry code, and returns only normalized `ToolResult` plus wrapped backend errors.

### MCP Library Adoption

`internal/mcp` wraps `github.com/mark3labs/mcp-go` for MCP protocol/session/transport behavior. This is a protocol implementation choice, not a policy boundary change.

- MVP validation is pinned to `github.com/mark3labs/mcp-go v0.49.0`. Any library version upgrade requires fresh focused tests plus TechSpec/ADR review before adoption.
- Primary-source anchor points for this decision:
  - `README.md` and `www/docs/pages/transports/` document first-class `stdio`, `SSE`, and `streamable HTTP` support.
  - `mcp/tools.go` defines `Tool`, `RawInputSchema`, and `RawOutputSchema`, which align with AGH's descriptor-authored hosted schemas.
  - `server/server.go`, `server/session.go`, and `server/streamable_http.go` define hosted/server lifecycle, per-session tools, and `tools/list_changed` behavior.
  - `client/stdio.go`, `client/http.go`, and `client/sse.go` define the remote stdio, streamable HTTP, and SSE client paths.
- Hosted AGH MCP proxy uses `server.NewMCPServer`, explicit `mcp.Tool` definitions, and `server.ServeStdio`. AGH session projections still decide which tools are registered, and every tool call still re-enters `Registry.Call` through AGH-owned UDS/session binding.
- External stdio MCP servers use `client.NewStdioMCPClient`.
- External remote HTTP MCP servers use `client.NewStreamableHttpClient`.
- External remote SSE MCP servers use `client.NewSSEMCPClient`.
- Hosted MCP tool registration MUST construct `mcp.Tool` values with `RawInputSchema` and `RawOutputSchema` taken byte-for-byte from `Descriptor.input_schema` and `Descriptor.output_schema`. `WithInputSchema`, `WithOutputSchema`, and other reflection-based schema helpers are forbidden for AGH-hosted tools because AGH manifest/runtime digests treat descriptor schema bytes as authoritative.
- AGH keeps its stricter canonical `ToolID` grammar even though `mcp-go` accepts a broader MCP tool-name grammar. AGH `ToolID` remains a valid subset and is passed to the library as MCP `Tool.name`.
- Hosted MCP tool registration should use library list-change behavior rather than inventing a parallel MCP notification format. AGH still owns when a projection changes and which session is allowed to observe it.
- MVP does not rely on `WithContinuousListening()` or any long-lived upstream notification subscription from external MCP servers. Remote MCP sessions are on-demand, bounded by executor idle TTL, and use upstream `notifications/tools/list_changed` only as cache invalidation hints if they appear during an active session.
- AGH preserves the existing remote MCP config surface: `transport = "stdio" | "http" | "sse"`. `http` means streamable HTTP via `mcp-go`; `sse` means the library's explicit SSE client path. AGH must not silently rewrite one transport into another.

Remote auth integration contract:

- `internal/mcp/auth` remains the only authority for remote MCP login, refresh, logout, durable token storage, and redacted status.
- For remote HTTP and SSE requests, `internal/mcp` injects outbound headers from current AGH-owned credential state using library header hooks such as `WithHTTPHeaderFunc` and `WithHeaderFunc`. Raw access and refresh tokens never leave `internal/mcp`.
- Before a remote `tools/list` or `tools/call`, the executor inspects AGH's persisted auth state. If the state is `authenticated` or `expired` and the token is refreshable but no longer usable, the executor may attempt exactly one `internal/mcp/auth.Service.Refresh` before creating or retrying the outbound client request.
- If a remote transport still returns an auth failure after that bounded refresh path, the executor returns only redacted backend errors mapped to `mcp_auth_required`, `mcp_auth_expired`, `mcp_auth_invalid`, or `mcp_auth_refresh_failed`.
- AGH MUST NOT use `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, default `transport.NewOAuthHandler`, `MemoryTokenStore`, or any library-owned login/cache/refresh flow as the authority for remote MCP credentials.
- If a later design wants to use `mcp-go` OAuth helper types for transport convenience, it must do so only behind an AGH-owned `TokenStore` adapter and a follow-up ADR. MVP does not depend on that path.

Extension protocol additions:

```go
const (
	CapabilityToolProvider = "tool.provider"

	ExtensionServiceMethodProvideTools = "provide_tools"
	ExtensionServiceMethodToolsCall    = "tools/call"
)
```

```go
var capabilityServiceMethods = map[Capability][]ExtensionServiceMethod{
	CapabilityToolProvider: {
		ExtensionServiceMethodProvideTools,
		ExtensionServiceMethodToolsCall,
	},
}
```

Wire request/response contracts:

```go
type ExtensionProvideToolsResponse struct {
	Tools []ExtensionToolRuntimeDescriptor `json:"tools"`
}
```

```go
type ExtensionToolCallRequest struct {
	ToolID    ToolID          `json:"tool_id"`
	Handler   string          `json:"handler"`
	SessionID string          `json:"session_id"`
	Input     json.RawMessage `json:"input"`
}
```

```go
type ExtensionToolCallResponse struct {
	Result ToolResult `json:"result"`
}
```

```go
type MCPToolCallRequest struct {
	ToolID      ToolID          `json:"tool_id"`
	RawToolName string          `json:"raw_tool_name"`
	Input       json.RawMessage `json:"input"`
}
```

```go
type MCPToolCallResponse struct {
	Result ToolResult `json:"result"`
}
```

Wire-stable fields are `tool_id`, `handler`, `raw_tool_name`, schema digests, risk flags, and JSON input/result envelopes. Runtime-only fields such as latency, retry count, process id, transport connection id, and raw auth/header material must stay out of wire structs and are added by daemon telemetry or redacted diagnostics only.

```go
type MCPAuthStatus struct {
	ServerName   string
	Status       string
	AuthType     string
	ClientID     string
	Scopes       []string
	ExpiresAt    *time.Time
	Refreshable  bool
	TokenPresent bool
	Diagnostic   string
}

type MCPAuthStatusProvider interface {
	Status(ctx context.Context, source SourceRef) (MCPAuthStatus, error)
}

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, scope Scope, d Descriptor) (EffectiveToolDecision, error)
}
```

```go
type ResultLimiter interface {
	Apply(ctx context.Context, d Descriptor, result ToolResult) (ToolResult, error)
}

type HookRunner interface {
	PreCall(ctx context.Context, call CallRequest) (CallRequest, EffectiveToolDecision, error)
	PostCall(ctx context.Context, call CallRequest, result ToolResult) (ToolResult, error)
	PostError(ctx context.Context, call CallRequest, err error) error
}
```

Error conventions:

- `ErrToolNotFound`
- `ErrToolConflict`
- `ErrToolUnavailable`
- `ErrToolDenied`
- `ErrToolApprovalRequired`
- `ErrToolInvalidInput`
- `ErrToolResultTooLarge`
- `ErrToolBackendFailed`

All production errors must wrap with `%w` where applicable and must map to deterministic API/CLI error codes.

### Data Models

`ToolID`

- Canonical public ID used by registry, policy, CLI, HTTP, UDS, hooks, telemetry, and hosted MCP.
- Format: `<segment>( "__" <segment> )*`
- Segment regex: `[a-z][a-z0-9_]*`
- Max length: 64.
- Lowercase ASCII only.
- `__` is reserved as namespace separator.
- No dots, hyphens, uppercase, empty segments, or dual wire aliases.
- External MCP/extension names that sanitize to more than 64 characters are rejected as conflicted with `id_too_long`; AGH does not truncate, hash-suffix, or create shadow aliases.

Examples:

- `agh__tool_list`
- `agh__tool_search`
- `agh__tool_info`
- `agh__skill_list`
- `agh__skill_search`
- `agh__skill_view`
- `agh__network_peers`
- `agh__network_send`
- `agh__task_list`
- `agh__task_read`
- `mcp__github__create_issue`
- `ext__linear__search`

`Descriptor`

- `id ToolID`
- `backend BackendRef`
- `display_title string`
- `description string`
- `input_schema json.RawMessage`
- `output_schema json.RawMessage`
- `source SourceRef`
- `visibility Visibility`
- `risk RiskClass`
- `read_only bool`
- `destructive bool`
- `open_world bool`
- `requires_interaction bool`
- `concurrency_safe bool`
- `max_result_bytes int64`
- `toolsets []ToolsetID`
- `tags []string`
- `search_hints []string`

`BackendRef`

- `kind BackendKind`
- `extension_id string`
- `handler string`
- `mcp_server string`
- `mcp_tool string`
- `native_name string`
- `requires_capabilities []string`

`ExtensionToolRuntimeDescriptor`

- `id ToolID`
- `handler string`
- `input_schema_digest string`
- `output_schema_digest string`
- `read_only bool`
- `risk RiskClass`
- `capabilities []string`

Schema digest contract:

- `input_schema_digest` and `output_schema_digest` are lowercase hex `sha256` digests over the JSON Schema subtree only.
- The bytes hashed are RFC 8785 JCS-canonicalized JSON. Object keys are sorted by the canonicalization algorithm, number/string escaping follows JCS, and `$ref` values are hashed literally rather than resolved during digesting.
- The digest excludes surrounding manifest metadata such as `ToolID`, handler, source, risk, and toolsets.
- TypeScript SDK, Go SDK, and daemon manifest validation must share byte-vector fixtures under `sdk/typescript/test-fixtures/digest/`, `sdk/go/test-fixtures/digest/`, and `internal/extension/testdata/digest/`.
- A digest mismatch is a hard `extension_runtime_mismatch`. There is no loose fallback, serializer-specific fallback, or warning-only mode.

`MCPToolDescriptor`

- `raw_name string`
- `title string`
- `description string`
- `input_schema json.RawMessage`
- `output_schema json.RawMessage`
- `source SourceRef`

`MCPToolDescriptor` is the daemon-internal normalized projection of one upstream MCP `Tool` before registry `Descriptor` synthesis. Hosted MCP descriptors preserve exact descriptor-authored schema bytes through `RawInputSchema` and `RawOutputSchema`. For external MCP discovery, AGH preserves raw schema bytes when the library surfaces them directly; otherwise it stores one canonical JSON encoding of the decoded schema and treats that canonical blob as authoritative for downstream digesting. AGH does not invent missing schemas or use reflection helpers during MCP normalization.

`SourceRef`

- `kind`: `builtin`, `mcp`, `extension`, `dynamic`
- `owner`: daemon, extension id, MCP server id, bundle id, skill id, or provider id
- `raw_server_name`
- `raw_tool_name`
- `resource_id`
- `resource_version`
- `workspace_id`
- `scope`

`Availability`

States are composable, not a single boolean:

- `registered`
- `enabled`
- `available`
- `authorized`
- `executable`
- `conflicted`

Reason codes include:

- `dependency_missing`
- `backend_unhealthy`
- `backend_not_executable`
- `extension_inactive`
- `extension_runtime_mismatch`
- `extension_capability_missing`
- `mcp_unreachable`
- `mcp_auth_unconfigured`
- `mcp_auth_required`
- `mcp_auth_expired`
- `mcp_auth_invalid`
- `mcp_auth_refresh_failed`
- `source_disabled`
- `policy_denied`
- `approval_required`
- `approval_timed_out`
- `approval_canceled`
- `session_denied`
- `hook_denied`
- `schema_invalid`
- `conflicted_id`
- `conflicted_sanitized_name`
- `id_too_long`
- `result_budget_exceeded`

MCP-backed descriptors may attach a redacted `MCPAuthStatus` to operator-visible tool views only. The status mirrors the existing `internal/mcp/auth.StatusValue` values (`unconfigured`, `needs_login`, `authenticated`, `expired`, `invalid`) plus registry reason codes, and may include `server_name`, `auth_type`, `client_id`, `scopes`, `expires_at`, `refreshable`, `token_present`, and `diagnostic`. It must never include access tokens, refresh tokens, OAuth authorization codes, PKCE verifiers, client secrets, approval tokens, or hosted MCP bind nonces. Session/model-visible projections do not include `MCPAuthStatus`; they hide or deny the tool through `Availability` reason codes.

`EffectiveToolDecision`

- `visible_to_operator bool`
- `visible_to_session bool`
- `callable bool`
- `approval_required bool`
- `system_permission_mode`
- `session_policy_result`
- `agent_policy_result`
- `registry_policy_result`
- `source_policy_result`
- `availability_result`
- `hook_result`
- `reason_codes []string`

`ToolResult`

- `content []ToolContent`
- `structured any`
- `preview string`
- `artifacts []ArtifactRef`
- `metadata map[string]any`
- `redactions []Redaction`
- `truncated bool`
- `bytes int64`
- `duration_ms int64`

`Toolset`

Toolsets are separate named resources/config entries, not overloaded as tools. They use a typed `ToolsetID`, include exact `ToolID`s and patterns, and may include other toolsets recursively. Policy fields must distinguish `tools` and `toolsets` to avoid ambiguity.

`ToolsetID` uses the same grammar as `ToolID`: lowercase ASCII segments separated by reserved `__`, maximum 64 characters, and no dots, hyphens, uppercase, or empty segments.

`dynamic` source kind is reserved because the existing source enum already has it, but MVP has no dynamic producer and no dynamic validation surface. Dynamic tools remain unavailable unless a later TechSpec defines their producer, policy, and provenance model.

### Data-Model Field Rationale

| Field or key | Shape | Purpose | Storage decision |
|---|---|---|---|
| `ToolID` | string, provider-safe `__`-segmented id | Single policy/dispatch/audit identity across registry, CLI, HTTP, UDS, hooks, telemetry, and hosted MCP | Typed field, never inferred from display title |
| `Descriptor.backend` | structured `BackendRef` | Binds the descriptor to the only allowed executable backend path | Typed struct; dispatch never infers backend from source prefix or metadata |
| `Descriptor.source` | structured `SourceRef` | Preserve raw external provenance without making prefixes the only source of truth | Typed struct; raw names are subfields, not alternate IDs |
| `Descriptor.input_schema` | JSON Schema object | Validate call input before dispatch | JSON schema payload is appropriate because schema content is externally structured and opaque to AGH queries |
| `Descriptor.output_schema` | JSON Schema object | Optional structured output contract | JSON schema payload is appropriate for the same reason as input schema |
| `Descriptor.visibility` | enum | Separate internal/operator/session/model surfaces | Typed enum, queryable and policy-relevant |
| `Descriptor.risk` | enum | Classify read/mutate/network/open-world behavior | Typed enum, never free-text metadata |
| `Descriptor.read_only` | bool | Drives `approve-reads` and safety checks | Typed bool; misclassification is a security bug |
| `Descriptor.destructive` | bool | Forces stricter policy and approval behavior | Typed bool, not metadata |
| `Descriptor.open_world` | bool | Marks tools that can reach arbitrary external state | Typed bool, not metadata |
| `Descriptor.requires_interaction` | bool | Distinguishes autonomous-safe calls from interactive calls | Typed bool, not metadata |
| `Descriptor.max_result_bytes` | int64 | Enforces result budget consistently across surfaces | Typed numeric field with config default |
| `Descriptor.toolsets` | `[]ToolsetID` | Supports recursive named bundles without overloading individual tools | Typed list; expanded to concrete `ToolID`s for lineage |
| `MCPAuthStatus` | redacted status object | Lets operator surfaces explain remote MCP login/expiry without exposing credentials | Derived from `internal/mcp/auth`; never persisted by the registry |
| `EffectiveToolDecision.reason_codes` | `[]string` enum values | Operator diagnostics and deterministic error contracts | Typed enum strings; no prose-only decisions |
| `[tools].enabled` | bool | Global registry execution switch | Config key with validation/defaults |
| `[tools].hosted_mcp_enabled` | bool | Allows session exposure through AGH-hosted MCP | Config key with validation/defaults |
| `[tools].default_max_result_bytes` | int64 | Default output cap when descriptor is silent | Config key with validation/defaults |
| `[tools.policy].external_default` | enum | Default executable policy for extension/MCP/dynamic tools | Config key, not hidden in metadata |
| `agent.tools` | `[]ToolID/pattern` | Allow concrete tools/patterns for an agent | Frontmatter/config field resolved to concrete lineage atoms |
| `agent.toolsets` | `[]ToolsetID` | Allow named bundles for an agent | Frontmatter/config field expanded before session lineage |
| `agent.deny_tools` | `[]ToolID/pattern` | Explicit narrowing layer for an agent | Frontmatter/config field, not runtime-only state |
| `extension.resources.tools.*.backend` | structured backend metadata | Declares the manifest-authoritative runtime binding for `extension_host` or `mcp` execution | Extension manifest fields, not in-process function pointers; daemon rejects mismatched runtime descriptors |
| `extension.resources.tools.*.handler` | string | Names the extension SDK handler used by `tools/call` | Manifest field validated against `provide_tools`; not executable by itself |
| `ExtensionToolRuntimeDescriptor.*_schema_digest` | lowercase hex SHA-256 | Reconciles live SDK handlers against manifest schemas across TypeScript, Go, and daemon validation | Runtime-only field computed from RFC 8785 canonical JSON fixtures |
| `ExtensionToolRuntimeDescriptor` | redacted runtime descriptor | Lets daemon confirm a running extension provides exactly the manifest-declared handler/schema/risk shape | Runtime-only reconciliation result; not persisted as source of truth |
| `MCPToolHandle` | daemon-owned client handle | Calls remote MCP tools without exposing remote credentials to descriptors or sessions | Runtime-only handle injected by `internal/mcp`; token material stays behind `internal/mcp/auth` |

No new SQLite columns are required for the MVP registry foundation. Existing session lineage stores concrete tool permission atoms; implementation should validate those atoms as canonical `ToolID`s. Existing remote MCP OAuth tokens already live in `globaldb.mcp_auth_tokens` through `mcpauth.TokenStore`; registry work must not add token fields to tool descriptors, resources, events, or metadata. Hosted MCP bind nonces are ephemeral launch correlation values and must not be stored in `mcp_auth_tokens`; if a later design needs durable hosted-proxy credential state, it must add a separate typed table with its own lifecycle and redaction tests. If a later task needs durable queryable tool-call history beyond append-only events, it must add a typed side table such as `tool_calls` rather than placing queryable call state in a session metadata JSON blob.

### Side-Table vs JSON Decisions

| Domain state | Decision | Rationale |
|---|---|---|
| Tool descriptors | Typed resource specs plus runtime descriptors | Descriptors are matchable by id/source/risk/visibility and must not live as opaque metadata blobs |
| Toolsets | Typed config/resource records | Toolset membership affects policy and lineage; it must be queryable and expandable deterministically |
| Tool call events | Append-only event payloads for MVP; typed side table only if queryable history is required | Events are the operational ledger; indexed call history would be matchable state and must not be hidden in JSON metadata |
| Source provenance | Structured `SourceRef` fields | Raw MCP/extension names are needed for debugging and collision handling |
| Input/output schemas | JSON Schema blobs | Schema contents are inherently opaque external contracts and are not AGH ownership state |
| Tool result structured payload | JSON payload plus typed envelope fields | Result body can be arbitrary, but status, bytes, truncation, redaction, and tool id are typed envelope fields |
| Policy decisions | Typed `EffectiveToolDecision` | Authorization is matchable and auditable; it cannot be a JSON bag |
| Availability reasons | Typed reason-code list | Operator diagnostics and tests need deterministic matching |

### API Endpoints

All endpoints are implemented once in `internal/api/core` and registered by HTTP and UDS transports.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/tools` | List operator-visible tools with availability/policy reason codes |
| `POST` | `/api/tools/search` | Search tools by id, title, description, source, tags, and toolsets |
| `GET` | `/api/tools/{id}` | Return descriptor, availability, policy view, schema, and source provenance |
| `POST` | `/api/tools/{id}/approvals` | Mint a single-use approval token for one concrete CLI/HTTP/UDS invocation |
| `POST` | `/api/tools/{id}/invoke` | Invoke a tool through registry dispatch |
| `GET` | `/api/sessions/{id}/tools` | Return session/model-visible callable projection |
| `POST` | `/api/sessions/{id}/tools/search` | Search only within effective session-callable projection |
| `GET` | `/api/toolsets` | List named toolsets and expansion status |
| `GET` | `/api/toolsets/{id}` | Inspect one toolset expansion and conflicts |

Invoke request:

```json
{
  "session_id": "sess_...",
  "workspace_id": "ws_...",
  "input": {},
  "approval_token": "optional-local-approval-reference"
}
```

`approval_token` is an opaque local approval reference issued by the daemon approval surface for CLI/HTTP/UDS calls. AGH stores only a hash, never logs or emits the raw value, redacts it from SSE/events/errors, scopes it to one tool decision, and treats it as separate from `claim_token`. Hosted MCP does not accept client-supplied `approval_token`; it uses the Hosted MCP Approval Bridge below.

### Approval Token Issuance

CLI/HTTP/UDS approval-required flows are two-step in MVP:

1. Request a daemon-issued approval token for one concrete invocation.
2. Replay that token in `POST /api/tools/{id}/invoke` or the equivalent UDS/CLI surface.

Issuance surfaces:

- HTTP: `POST /api/tools/{id}/approvals`
- UDS: matching `CreateToolApproval` client method
- CLI: `agh tool approve <tool-id> --session <id> --workspace <id> --input <json> -o json`

Issuance request fields:

- `session_id`
- `workspace_id`
- `input` or `input_digest`

Issuance response fields:

- `approval_token`
- `expires_at`
- `tool_id`
- `input_digest`

Issuance rules:

- Only daemon-authenticated operator surfaces may mint approval tokens. Hosted MCP never mints or accepts them.
- Approval tokens live only in daemon memory. Daemon restart invalidates all outstanding approval tokens.
- The daemon stores only a hash of the token plus typed binding fields: `tool_id`, `session_id`, `workspace_id`, `input_digest`, `issued_at`, and `expires_at`.
- TTL is exactly `[tools.policy].approval_timeout_seconds`.
- Tokens are single-use. Successful invoke consumption invalidates the token immediately.
- Replay, mismatch, expiration, or missing-token cases return deterministic reason codes: `approval_token_missing`, `approval_token_expired`, `approval_token_mismatch`, and `approval_token_replayed`.
- Raw approval tokens may traverse only the authenticated issuance response and the matching invoke request. They never appear in logs, events, SSE payloads, hosted MCP, persisted state, or diagnostics.

Invoke response:

```json
{
  "tool_id": "agh__skill_view",
  "status": "completed",
  "result": {},
  "truncated": false,
  "duration_ms": 23,
  "events": []
}
```

Status codes:

- `200` completed/listed.
- `202` approval required or async dispatch accepted, only if the tool is explicitly async.
- `400` invalid `ToolID`, invalid schema input, or malformed request.
- `403` denied by ACP ceiling, session lineage, registry policy, source policy, or hook.
- `404` not found or hidden from caller context.
- `409` conflicted canonical ID or sanitized external name.
- `422` registered but unavailable or not executable.
- `500` internal daemon error.
- `502` backend adapter failure.

CLI parity:

- `agh tool list -o json`
- `agh tool search <query> -o json`
- `agh tool info <tool-id> -o json`
- `agh tool invoke <tool-id> --input <json> -o json`
- `agh toolsets list -o json`
- `agh toolsets info <toolset-id> -o json`
- `agh tool mcp --session <session-id> --bind-nonce <nonce>` for the daemon-spawned hosted MCP stdio proxy

## Integration Points

### ACP

ACP does not impose a callable tool registry pattern. AGH must not use ACP `ToolCall.title` as a policy identity or dispatch key.

AGH will integrate with ACP by:

- passing the hosted AGH MCP server in ACP `mcpServers` during session creation/load when the selected agent supports MCP;
- preserving ACP tool-call observations as session events keyed by `toolCallId`;
- mapping AGH-owned tool calls back into ACP lifecycle updates where relevant;
- treating ACP `ToolKind` as risk/display metadata, not identity;
- keeping `permissions.mode` as the system/session approval ceiling.

Current-state caveat: `internal/acp.toSDKMCPServers` currently emits stdio-only `acpsdk.McpServer` values. MVP registry work must keep hosted AGH MCP as a stdio-only injected server and must not imply remote MCP HTTP ACP parity until a later implementation adds tested HTTP conversion, redacted Authorization/header handling, and provider capability checks.

### Hosted MCP

MVP exposure path:

```text
agent session -> ACP mcpServers -> agh tool mcp --session <id> --bind-nonce <nonce> -> UDS -> daemon Tool Registry
```

The hosted MCP server lists only session-callable tools. It exposes MCP `Tool.name` equal to AGH canonical `ToolID`. It does not expose unavailable, unauthorized, or conflicted tools to the model-visible surface.

The proxy process is an `mcp-go`-based stdio MCP server: `agh tool mcp` constructs a `server.NewMCPServer(...)`, registers one `mcp.Tool` per session-callable registry descriptor using the descriptor's exact schema bytes through `RawInputSchema` and `RawOutputSchema`, and serves over `server.ServeStdio`. The UDS bind, session/workspace scoping, approval bridge, result limiting, and dispatch callback remain AGH-owned.

Hosted MCP authentication:

- On session creation/load, the daemon records a short-lived hosted MCP launch record keyed by `session_id`, `workspace_id`, an opaque non-secret `hosted_mcp_bind_nonce`, expiry, and expected AGH binary path.
- Hosted MCP launch records live only in daemon memory. Daemon restart invalidates all pending launch records and forces session/load to mint a fresh nonce.
- The `hosted_mcp_bind_nonce` is a correlation nonce, not a bearer secret and not claim-token-equivalent. It may traverse ACP `mcpServers[].args` because it is insufficient without UDS peer credentials; raw `claim_token`, remote MCP OAuth tokens, and approval tokens never traverse this path.
- At startup, `agh tool mcp --session <id> --bind-nonce <nonce>` performs a UDS bind RPC. The daemon accepts the bind only when the nonce matches a live launch record, the Unix-domain socket peer credentials identify the same OS user, the peer executable matches the expected AGH binary, and the record has not expired.
- If the platform cannot provide peer credentials or executable validation, hosted MCP binding fails closed and the session receives no hosted registry projection on that platform.
- The daemon binds the UDS connection to exactly one session/workspace projection and rejects any later client-supplied `session_id` or `workspace_id`.
- The launch record is invalidated on first successful bind, session end, proxy disconnect, or TTL expiry, whichever happens first.
- A foreign process calling `agh tool mcp --session <id>` without a valid nonce plus matching UDS peer credentials receives a deterministic permission error and no tool projection.
- Redaction tests must cover ACP payload diagnostics, process diagnostics, crash bundles, logs, SSE/events, settings output, tool results, and MCP responses. The nonce may appear in AGH-owned diagnostics as a redacted/correlatable launch id, but it must never be described as a claim-token-equivalent bearer secret.

Hosted MCP approval bridge:

- Hosted MCP projections include only tools that are callable without a new approval prompt or tools whose session has a live daemon-mediated approval channel.
- When `EffectiveToolDecision.approval_required=true` and ACP `session/request_permission` is available, `Registry.Call` derives a context with `[tools.policy].approval_timeout_seconds`, issues the ACP permission request, and blocks the MCP `tools/call` response until approved, denied, timed out, canceled, or the hosted MCP stdio/UDS connection closes.
- Hosted MCP request contexts must reserve a guard band above approval waiting. The library-facing hosted MCP request deadline must be at least `[tools.policy].approval_timeout_seconds + 5s` so a valid daemon approval bridge wait cannot be preempted by an earlier transport deadline.
- When no approval channel is available, hosted MCP hides the tool from `tools/list` if that can be determined during projection. If a call still reaches dispatch, it returns `ErrToolApprovalRequired` with reason codes `approval_required` and `approval_unreachable`.
- Approval timeout returns `ErrToolApprovalRequired` with `approval_required` and `approval_timed_out`. Hosted MCP proxy disconnect or stdio close cancels the derived context and returns `approval_required` plus `approval_canceled`.
- Hosted MCP cannot satisfy approval using client-supplied arguments. CLI/HTTP/UDS may use `approval_token`; hosted MCP must use the daemon approval bridge.
- Remote MCP outbound call deadlines are derived independently from backend call policy and must not inherit the hosted MCP approval-wait deadline after approval completes.

Hosted MCP lifecycle:

- The stdio proxy is spawned by the ACP runtime from AGH-provided `mcpServers` config and is scoped to one AGH session.
- After successful bind, the proxy opens a daemon-owned UDS projection stream for that bound session. The daemon pushes add/remove/update deltas for session-callable tools; the proxy translates them into `mcp-go` add/remove/replace operations so the library emits `notifications/tools/list_changed` and `tools/list` stays equal to the daemon projection.
- If the projection stream drops, the proxy must fail closed by closing the MCP session instead of continuing to serve a stale tool catalog.
- The proxy exits when stdio closes, when the session stops, or when the launch record expires before successful bind.
- On ACP `session/load`, the daemon mints a fresh bind nonce and provides a fresh hosted MCP entry for that resumed session.
- The proxy never accepts a client-supplied workspace id. The daemon derives workspace id from the bound session at projection time and dispatch time.

### Existing MCP Config And Auth

AGH already has an MCP server configuration and remote-auth subsystem. The Tool Registry must consume those surfaces instead of defining a parallel MCP model:

- `internal/config/provider.go` currently defines `MCPServer`, `MCPServerTransport` (`stdio`, `http`, `sse`), and `MCPAuthConfig` for OAuth 2.1 + PKCE metadata/client settings. Token material is explicitly outside config. This TechSpec keeps that remote transport surface and maps it directly onto `mcp-go` client transports.
- `internal/config/mcpjson.go` loads `mcp.json` sidecars using `mcpServers` or `mcp_servers`.
- `internal/config/mcp_resource.go` validates `mcp_server` desired-state resources.
- `internal/mcp/auth` owns metadata discovery, PKCE state, authorization-code exchange, refresh, redacted status, logout/revocation, and `StatusValue`.
- `internal/store/globaldb/global_db_mcp_auth.go` persists remote MCP OAuth tokens behind `mcpauth.TokenStore` with encryption/redaction boundaries.
- `internal/cli/mcp_auth.go` provides the existing agent-operable commands `agh mcp auth login`, `agh mcp auth status`, and `agh mcp auth logout`.
- `internal/settings` and `internal/api/contract/settings.go` already expose redacted MCP auth status for settings surfaces.

The hosted MCP bind nonce described above is not a remote MCP OAuth token and is not a bearer credential by itself. It is a daemon-minted correlation value for AGH's local stdio proxy, validated together with UDS peer credentials and the expected AGH binary path. Remote MCP OAuth tokens remain owned by `internal/mcp/auth` and `globaldb`. Registry descriptors, events, tool results, and MCP proxy arguments must never mix those credentials or reuse one lifecycle for the other.

Remote MCP call-through uses an `MCPCallExecutor` implemented inside `internal/mcp`. `internal/tools` depends only on the executor interface and redacted `MCPAuthStatusProvider`; it must not import `internal/mcp/auth`, open `mcpauth.TokenStore`, receive raw bearer strings, or construct Authorization headers. The executor resolves bearer material internally, applies transport-specific headers in memory for the outbound MCP request, and maps failures back to redacted registry errors.

Implementation direction for the executor:

- stdio MCP servers use `client.NewStdioMCPClient`;
- remote HTTP MCP servers use `client.NewStreamableHttpClient`;
- remote SSE MCP servers use `client.NewSSEMCPClient`;
- `MCPCallExecutor` maintains one `mcp-go` client/session per `SourceRef` with a bounded idle TTL, recreates it after transport failure or auth-status change, and applies registry-derived per-call deadlines to `ListTools` and `CallTool`;
- for remote HTTP and SSE, outbound auth headers are injected from current AGH-owned credential state using library header hooks inside `internal/mcp`;
- the executor may attempt at most one `internal/mcp/auth.Service.Refresh` for refreshable `authenticated` or `expired` states before client creation or one retry after an auth failure; it must never bootstrap a new login flow and must return only redacted reason-mapped errors otherwise;
- AGH must not instantiate `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, default `transport.NewOAuthHandler`, `MemoryTokenStore`, or any other library-managed OAuth flow as the authority for remote MCP credentials;
- MVP external MCP descriptor freshness is on-demand, not subscription-based: the daemon refreshes upstream MCP descriptors during projection rebuilds and before a call when the cached descriptor set is missing or stale. It does not rely on long-lived upstream `notifications/tools/list_changed` subscriptions from external MCP servers in MVP.
- If an upstream external MCP server still delivers `notifications/tools/list_changed` during an active client session, AGH treats it only as a local cache-invalidation hint. It must not mutate registry structure from the notification alone and must not maintain standalone notification subscriptions in MVP.
- no new manual MCP message framing, request/response routing, or transport-specific retry layer should be introduced outside the `mcp-go` wrapper without a later ADR.

External MCP-backed tool availability must derive auth diagnostics from the existing auth service:

| `internal/mcp/auth.StatusValue` | Registry reason code | Session projection behavior |
|---|---|---|
| `unconfigured` | `mcp_auth_unconfigured` | Hide external MCP tools unless the server is public and executable support exists |
| `needs_login` | `mcp_auth_required` | Hide from model-visible projection; operator view points to `agh mcp auth login <server>` |
| `authenticated` | none | Auth does not block availability; calls may proceed only through the daemon-owned MCP adapter after registry policy passes |
| `expired` | `mcp_auth_expired` | Hide from model-visible projection; operator view points to `agh mcp auth status --refresh <server>` |
| `invalid` | `mcp_auth_invalid` | Hide from model-visible projection; operator view points to logout/login repair |

Implementation correction required before the registry consumes MCP resource catalogs: current `internal/daemon/tool_mcp_resources.go` clones MCP server records through `cloneDaemonMCPServer`, which preserves only `Name`, `Command`, `Args`, and `Env`. Registry work that depends on remote MCP resources must update that clone path and its tests to preserve `Transport`, `URL`, and `Auth`; otherwise remote MCP auth metadata will be silently dropped from tool diagnostics.

### Extensions

Extension-installed tools are possible and should be first-class.

Current foundation already has:

- `extension.toml` `resources.tools`
- `resources.publish.families = ["tools"]`
- resource projection into daemon tool records
- extension health/status infrastructure
- Host API capability checks

This TechSpec extends that by adding manifest-authoritative backend metadata, runtime reconciliation, and executable out-of-process handlers:

```toml
[resources.tools.search]
id = "ext__linear__search"
description = "Search Linear issues"
read_only = true
backend.kind = "extension_host"
backend.handler = "search"
toolsets = ["linear__read"]
```

TypeScript extensions define the matching runtime handler through `@agh/extension-sdk`:

```ts
extension.tool("search", {
  readOnly: true,
  inputSchema: z.object({ query: z.string() }),
}, async ({ input, context }) => {
  return { content: [{ type: "text", text: await searchLinear(input.query, context) }] };
});
```

Go extensions define the same handler through the public Go extension SDK:

```go
ext.Tool("search", aghsdk.ToolOptions{
	ReadOnly: true,
	InputSchema: searchInputSchema,
}, func(ctx context.Context, req aghsdk.ToolRequest[SearchInput]) (aghsdk.ToolResult, error) {
	return searchLinear(ctx, req.Input)
})
```

Extensions must not freely impersonate `agh__*` or another extension namespace. Raw manifest names remain in `SourceRef`. `extension.toml` is the source of truth; SDK registration is runtime proof that the live extension process implements the manifest-declared handler and compatible schemas.

Execution boundaries:

- `native_go`: full MVP dispatch through in-process daemon code compiled with AGH.
- `extension_host`: full MVP dispatch through out-of-process extension runtime, `tool.provider`, `provide_tools`, and `tools/call`.
- `mcp`: full MVP dispatch through daemon-owned MCP client adapters using existing MCP config/auth.
- `subprocess`: not a separate public backend kind; TypeScript and Go extension SDKs use the existing subprocess runtime behind `extension_host`.
- `bridge`: future bridge adapter, rejected by MVP validation unless a later TechSpec enables it.

No in-process third-party extension handlers in MVP. A Go function authored by an extension runs in the extension's subprocess binary through the Go SDK, not inside the daemon.

### MCP Sources

MCP-backed tools use:

```text
mcp__<server>__<tool>
```

The registry must preserve raw server/tool names in `SourceRef`. Sanitization collisions fail closed and mark the candidate tool `conflicted`.

Canonicalization contract:

```go
func Canonicalize(rawServer, rawTool string) (ToolID, error)
```

Rules:

- trim surrounding ASCII whitespace from `rawServer` and `rawTool`; either empty result is invalid;
- lowercase ASCII letters;
- replace `-` and `.` with `_` inside each raw segment;
- reject any rune outside ASCII letters, digits, `_`, `-`, and `.` rather than transliterating or dropping it;
- reject any normalized segment that starts with a digit, becomes empty, or contains `__` after normalization;
- reject any normalized segment whose leading/trailing `_` would make the reserved `__` separator ambiguous;
- assemble the result as `mcp__<server>__<tool>`;
- if the assembled ID exceeds 64 characters, fail with `id_too_long`;
- if two distinct raw `(server, tool)` pairs normalize to the same assembled ID, keep raw provenance in `SourceRef` and fail closed with `conflicted_sanitized_name` at registry indexing time.

`Canonicalize` is the only allowed raw-name to canonical-`ToolID` transform for MCP sources. `MCPCallExecutor`, MCP descriptor normalization, hosted MCP registration, and config/resource validation must all reference the same helper and share one fixture set proving byte-stable results.

AGH-managed MCP sources in MVP are the existing validated projections, not raw file scans:

- top-level and workspace MCP config plus global/workspace `mcp.json` sidecars;
- provider and agent MCP server declarations;
- skill MCP declarations resolved by `internal/skills.MCPResolver`, including the existing `allowed_marketplace_mcp` trust gate;
- extension `resources.mcp_servers` records resolved by `internal/extension/resource_publication.go`;
- future client-supplied ACP `mcpServers` only if a later TechSpec makes them session-scoped tool sources with explicit source trust and collision handling.

Top-level, provider, agent, and `mcp.json` declarations use `aghconfig.MCPServer` and may carry remote `transport`, `url`, and `auth` fields. Current skill and extension MCP declarations are stdio subprocess declarations with `name`, `command`, `args`, and `env`; registry work must not infer remote OAuth support from them until their manifests are explicitly extended.

External MCP descriptor discovery must consume existing `aghconfig.MCPServer` config/resource projections and `internal/mcp/auth` redacted status. It must not read raw config files directly, bypass strict `mcp.json` decoding, bypass skill sidecar symlink hardening, bypass marketplace MCP consent, or bypass extension resource grants.

External MCP-backed tools are executable in the MVP only through daemon-owned MCP client adapters. They are session-callable when descriptor discovery succeeds, the source is explicitly allowed, existing MCP auth status is usable, the registry policy/approval/session/hook gates pass, and dispatch can call the remote MCP server without exposing token material. Operator surfaces still show unavailable MCP tools with deterministic diagnostics when health, auth, source, policy, collision, or schema checks fail.

### Hooks

Existing tool hook concepts become part of central registry dispatch:

- `tool.pre_call`
- `tool.post_call`
- `tool.post_error`

Payloads should use canonical `tool_id`. Existing `tool_name` / `tool_namespace` usage should be replaced in the registry path to avoid dual identity.

### Skills

Built-in skill tools prove progressive disclosure:

- `agh__skill_list`
- `agh__skill_search`
- `agh__skill_view`

These call into `internal/skills.Registry`, respect workspace overlays, reuse content verification, and enforce result budgets. Install/remove/update tools are intentionally out of MVP unless supply-chain policy/scanning is expanded.

`agh__skill_view` applies registry result budgeting. If content exceeds the descriptor limit, the result returns `truncated=true`, a typed `next_offset`, and an artifact/reference strategy rather than silently dropping content.

### Network And Tasks

MVP network tools:

| ToolID | Read-only | Destructive | Open-world | Authority route |
|---|---:|---:|---:|---|
| `agh__network_peers` | true | false | false | Existing network peer/list service |
| `agh__network_send` | false | false | true | Existing network send service with channel/session policy checks |

MVP task tools:

| ToolID | Read-only | Destructive | Open-world | Authority route |
|---|---:|---:|---:|---|
| `agh__task_list` | true | false | false | `task.Service.ListTasks` |
| `agh__task_read` | true | false | false | `task.Service.GetTask` |
| `agh__task_create` | false | false | false | `task.Service.CreateTask` |
| `agh__task_child_create` | false | false | false | `task.Service.CreateChildTask` |
| `agh__task_update` | false | false | false | `task.Service.UpdateTask` |
| `agh__task_cancel` | false | true | false | `task.Service.CancelTask` |
| `agh__task_run_list` | true | false | false | `task.Service.ListTaskRuns` |

Excluded task tools:

- `agh__task_claim`
- `agh__task_release`
- `agh__task_complete`
- `agh__task_fail`
- `agh__task_run_start`
- `agh__task_run_complete`
- `agh__task_run_cancel`

Those excluded tools cross claim/lease/session lifecycle authority. They require a separate task-execution TechSpec because `task.Service.ClaimNextRun`, `Spawn`, and session manager terminal-state transitions are authoritative primitives and must not be wrapped by generic agent-callable tools.

All network and task tools must route through existing network/task services and existing authorization rules. Mutating tools must not be classified as read-only.

`agh__task_child_create` must call `task.Service.CreateChildTask`, and lineage subset enforcement remains in that service-level authority path. Registry policy may narrow the call before dispatch, but it must not become the authoritative child-permission expander or allow a child task/session to widen beyond the parent.

## Extensibility Integration Plan

### Extension Manifests

Update extension manifest tool declarations with runtime metadata:

- `id`
- `backend.kind`
- `backend.handler`
- `backend.server`
- `backend.tool`
- `requires_env`
- `required_capabilities`
- `risk`
- `destructive`
- `open_world`
- `requires_interaction`
- `max_result_bytes`
- `toolsets`
- `tags`
- `visibility`

Extension-published descriptors are installed through existing resource publication, then normalized by the registry provider. For `extension_host` tools, the extension process must advertise `tool.provider`, expose `provide_tools`, and implement `tools/call`. The daemon marks the tool executable only when the runtime descriptor matches the manifest-authoritative `id`, handler name, schema digests, and risk flags.

Extension-published MCP servers currently use `resources.mcp_servers` with stdio-only `command`, `args`, and `env` fields through `internal/extension/resource_publication.go`. This TechSpec does not add remote OAuth fields to extension MCP server declarations in MVP. If a future extension wants to publish remote authenticated MCP servers, that future TechSpec must extend the extension manifest schema to mirror `aghconfig.MCPServer` transport/auth fields and reuse `internal/mcp/auth`; it must not introduce extension-local token storage.

For `mcp` extension tool descriptors, `backend.server` must resolve to an existing authorized MCP server source in the same extension/config scope, and `backend.tool` must match a discovered MCP tool. Missing, unauthorized, unhealthy, unauthenticated, or conflicted backend servers keep the tool operator-visible but unavailable with deterministic reason codes. A missing or unauthorized backend server cannot make the tool session-callable.

For `extension_host` descriptors, `backend.handler` must match a runtime handler exposed by the extension SDK. Missing handlers, schema digest mismatches, risk flag mismatches, inactive extensions, or missing `tool.provider` grants keep the tool operator-visible but unavailable with `extension_runtime_mismatch`, `extension_capability_missing`, or `extension_inactive`.

### Hooks

Add or update hook payload schemas to include:

- `tool_id`
- `display_title`
- `source`
- `risk`
- `read_only`
- `destructive`
- `open_world`
- `session_id`
- `workspace_id`
- `decision`
- `reason_codes`
- `input_digest`
- `result_digest`

Hooks can deny or patch only through typed return contracts. They cannot raise permissions above ACP/session policy.

### Skills, Tools, Resources, Bundles

Add toolsets as named resources/config entries:

- built-in `agh__bootstrap`
- built-in `agh__catalog`
- built-in `agh__coordination`
- built-in `agh__tasks`
- extension-provided toolsets such as `linear__read`

Bundles may include toolsets in the future, but must expand to concrete `ToolID`s during session projection to preserve lineage narrowing.

Skill MCP sidecars (`mcp.json`) remain MCP server declarations, not executable registry tools by themselves. Current skill MCP declarations carry only `name`, `command`, `args`, and `env`; they are stdio declarations. The registry may use those declarations as external MCP descriptor sources only after preserving skill sidecar symlink hardening, applying the existing skill trust gate (`allowed_marketplace_mcp`), and adding remote auth diagnostics from `internal/mcp/auth` only when the underlying source is an auth-capable `aghconfig.MCPServer`.

### Bridge SDKs

No direct bridge SDK execution in MVP. The registry design reserves a `bridge` backend kind, but the adapter is not required until bridge-managed tools need execution.

### AGH Network

No remote peer tool execution in MVP. Peer discovery may later advertise loaded toolsets or tool summaries, but remote dispatch requires a separate trust and authorization design.

### Docs For Extension Authors

Add docs covering:

- canonical `ToolID` rules;
- extension tool manifest shape;
- TypeScript `extension.tool(...)` authoring;
- Go SDK function-based tool authoring;
- backend kinds;
- why third-party handlers are out-of-process;
- manifest/runtime reconciliation failures and how to debug `provide_tools`;
- result budgets and redaction;
- availability reason codes;
- how to debug conflicted/unavailable tools;
- CLI/HTTP/UDS management paths.

## Agent Manageability Plan

Agents must be able to inspect and operate the registry without the web UI.

CLI:

- `agh tool list -o json`
- `agh tool search <query> -o json`
- `agh tool info <tool-id> -o json`
- `agh tool invoke <tool-id> --input <json> -o json`
- `agh toolsets list -o json`
- `agh toolsets info <toolset-id> -o json`
- `agh tool mcp --session <id> --bind-nonce <nonce>`
- Existing remote MCP auth commands remain the management path for external MCP credentials: `agh mcp auth login <server> -o json`, `agh mcp auth status [server] -o json`, and `agh mcp auth logout <server> -o json`.

HTTP and UDS parity:

- Same contract types.
- Same reason codes.
- Same redaction.
- Same policy decisions.
- UDS is the preferred local machine path for CLI and hosted MCP proxy.
- Existing MCP settings endpoints remain the management path for server config/status: `GET /api/settings/mcp-servers`, `PUT /api/settings/mcp-servers/:name`, and `DELETE /api/settings/mcp-servers/:name` over HTTP and UDS.
- Tool registry operator views may embed or link redacted settings `auth_status`; they must not create duplicate MCP auth commands or expose token material.

Discovery behavior:

- Operator surfaces show all registered tools, including unavailable/unauthorized/conflicted entries.
- Session/model-visible surfaces show only callable tools for that effective session.
- Dispatch always recomputes policy and availability even if discovery already hid unsafe tools.

Deterministic errors:

- Errors include `code`, `message`, `tool_id`, `reason_codes`, and redacted structured details.
- Policy errors must identify the denying layer: `system_permission_mode`, `session_lineage`, `agent_policy`, `registry_policy`, `source_policy`, `availability`, or `hook`.
- MCP auth errors identify the MCP server name and redacted status/reason code, never token material. Tool registry surfaces may recommend the existing `agh mcp auth ...` repair command, but they do not start OAuth login/logout flows in MVP.

E2E manageability checks:

- CLI list/search/info matches HTTP and UDS for the same workspace/session.
- Hosted MCP `tools/list` equals `GET /api/sessions/{id}/tools`.
- Denied tools are visible in operator list but absent from session projection.
- Extension-installed tool descriptor appears after install and disappears after disable/remove.

## Config Lifecycle

### Global `config.toml`

Add:

```toml
[tools]
enabled = true
hosted_mcp_enabled = true
default_max_result_bytes = 262144

[tools.hosted_mcp]
bind_nonce_ttl_seconds = 30

[tools.policy]
external_default = "disabled"
approval_timeout_seconds = 120
trusted_sources = []
```

Semantics:

- `tools.enabled=false` disables AGH-owned registry dispatch and hosted MCP exposure, but operator diagnostics can still show static resources where safe.
- `hosted_mcp_enabled=true` allows AGH to inject/offer the local hosted MCP proxy for sessions.
- `bind_nonce_ttl_seconds=30` bounds the hosted MCP launch record lifetime before UDS peer-credential binding.
- `default_max_result_bytes` applies when a descriptor does not specify a smaller limit.
- `external_default="disabled"` means extension/MCP/dynamic executable tools are registered and operator-visible, but not session-callable until enabled by explicit tool, toolset, source-tier, or agent grants. Built-in AGH bootstrap tools remain enabled by default subject to ACP/session policy.
- `approval_timeout_seconds=120` bounds daemon-mediated approvals for hosted MCP, CLI, HTTP, and UDS calls.
- `trusted_sources=[]` is an explicit source allowlist for external read-only auto-approval. Empty means no extension/MCP source can rely on `approve-reads` without an explicit per-tool, toolset, source, or agent grant.
- Hosted MCP launch records and CLI/HTTP/UDS approval tokens both live only in daemon memory. Restarting the daemon invalidates all pending nonces/tokens and forces fresh issuance.

Allowed `external_default` values:

- `disabled`
- `ask`
- `enabled`

MVP default is `disabled`.

`approve-reads` does not auto-approve `extension` or `mcp` source tools unless the source is present in `trusted_sources`, even when the descriptor declares `read_only=true`. Mutating, destructive, open-world, or interaction-requiring extension/MCP tools cannot become callable through `approve-reads`. They require explicit policy grants by `ToolID`, toolset, source, or agent plus the effective ACP/session ceiling, approval bridge when required, session lineage, and hook revalidation.

### Existing MCP Config And Auth Lifecycle

No new `config.toml` keys are added for remote MCP OAuth tokens. Existing MCP lifecycle remains authoritative:

- MCP server definitions continue to come from top-level `[mcp_servers]`, provider `[providers.<name>.mcp_servers]`, agent-local `mcp_servers`, global/workspace `mcp.json`, skill sidecars, and extension `resources.mcp_servers`.
- Remote MCP auth configuration continues to use `MCPAuthConfig` fields (`type`, metadata/issuer/authorization/token/revocation URLs, `client_id`, `client_secret_env`, `scopes`) on remote MCP servers.
- Access tokens and refresh tokens continue to live only in the `internal/mcp/auth` token store backed by `internal/store/globaldb`; they are not copied into registry config, session lineage, tool descriptors, events, or extension manifests.
- The registry may read redacted MCP auth status to produce operator diagnostics and availability reason codes, but cannot mutate auth state. Login, refresh, and logout remain `agh mcp auth ...` operations in MVP.
- The hosted MCP bind nonce is ephemeral process/session launch state, not `config.toml` state and not part of the MCP OAuth token store.

### Agent Definitions

Keep the existing `tools` field but harden its meaning:

- `tools`: exact canonical `ToolID`s or approved wildcard patterns.
- `toolsets`: named toolset IDs.
- `deny_tools`: exact IDs or patterns that always narrow permissions.

Session lineage should persist concrete resolved `ToolID` atoms, not broad unresolved wildcard patterns. Child session permissions must remain subsets of parent session permissions.

Invalid existing lineage atoms reject session spawn/load with a typed validation error. Greenfield posture applies: AGH does not silently normalize old atoms, and local databases that predate this TechSpec require a fresh `AGH_HOME` rather than compatibility migration shims.

### Tool Pattern Grammar

Allowed policy pattern forms:

- exact canonical IDs, for example `agh__skill_view`;
- namespace-prefix wildcards ending in `*`, for example `agh__skill_*` or `mcp__github__*`;
- toolset IDs in `toolsets`, never in `tools`.

Disallowed forms:

- regular expressions;
- suffix wildcards such as `*__search`;
- mid-segment wildcards such as `agh__*__view`;
- uppercase, dots, hyphens, or empty segments;
- wildcard forms that would match across a reserved `__` boundary ambiguously.

Pattern matching runs against canonical `ToolID` only. Display titles, raw MCP tool names, and extension manifest names do not participate in policy matching.

### Validation

Config validation must reject:

- invalid `ToolID` patterns;
- unknown toolset IDs when a config is resolved in a concrete workspace;
- `__` misuse;
- extension attempts to publish under reserved `agh__*`;
- transport values outside the explicit set `{stdio, http, sse}`; AGH does not silently rewrite one transport into another;
- global defaults that would expose external tools without source policy support;
- `trusted_sources` entries that do not resolve to known extension/MCP source refs;
- approval timeouts or hosted MCP bind nonce TTLs outside daemon min/max bounds;
- result byte limits below zero or above a daemon maximum;
- any descriptor or resource publication attempt that uses `source.kind = dynamic` in MVP.

Daemon min/max bounds:

- `[tools.hosted_mcp].bind_nonce_ttl_seconds` must be within `[5, 300]`
- `[tools.policy].approval_timeout_seconds` must be within `[10, 1800]`

### Docs And Generated Surfaces

Update:

- CLI docs for `agh tool` and `agh toolsets`;
- existing MCP auth CLI docs when registry diagnostics reference `agh mcp auth ...`;
- settings docs for `permissions.mode` to clarify ceiling behavior;
- settings MCP server docs for redacted `auth_status` reuse in tool diagnostics;
- extension author docs;
- site docs for Tool Registry architecture;
- OpenAPI contract and generated web types.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|---|---|---|---|
| `internal/tools` | Modified/new | Becomes runtime registry owner, not just metadata definitions | Add `ToolID`, descriptors, providers, registry, policy, dispatch |
| `internal/config` | Modified/consumed | Existing `MCPServer` transport/auth config is the source of truth for MCP resources, including remote `stdio`, `http`, and `sse` | Preserve `transport`, `url`, and `auth`; do not rewrite one transport into another or move OAuth config under `[tools]` |
| `internal/resources` | Modified | Cold tool resource remains desired state but must carry canonical ID/source metadata | Update codecs, validators, tests |
| `internal/extension` | Modified | Extension tools gain backend metadata, manifest/runtime reconciliation, and executable out-of-process invocation | Extend manifest types, protocol capabilities, `provide_tools`, `tools/call`, validation, lifecycle, and publication tests |
| `internal/mcp` | Modified/new | Hosted MCP proxy exposes registry tools; MCP adapter normalizes and executes external tools | Add MCP list/call bridge through UDS/registry and daemon-owned remote MCP client call-through |
| `github.com/mark3labs/mcp-go` | New dependency | MCP protocol/session/transport implementation used by `internal/mcp` and hosted MCP proxy | Add through `go get` pinned to `v0.49.0`; keep usage wrapped behind AGH-owned boundaries |
| `internal/mcp/auth` | Consumed | Existing remote MCP OAuth/PKCE status drives external MCP availability diagnostics | Inject redacted status provider; do not duplicate token store or OAuth flows |
| `internal/acp` | Modified | Session creation/load must include hosted AGH MCP where applicable; permission mode becomes registry ceiling; current MCP conversion is stdio-only | Wire session projection, keep hosted MCP stdio-only in MVP, and avoid implying remote HTTP ACP parity |
| `internal/store` | Modified | Session lineage `Tools` atoms become canonical resolved `ToolID`s | Validate IDs and preserve subset checks |
| `internal/hooks` | Modified | Tool hook payloads should use canonical `tool_id` | Update payloads, matchers, docs, tests |
| `internal/api/contract` | New/modified | Shared DTOs for tools/toolsets/calls/errors | Add contract types and codegen |
| `internal/api/core` | New/modified | Transport-independent tool handlers | Implement list/search/info/invoke/session projection |
| `internal/api/httpapi` | Modified | Register HTTP routes | Thin transport registration only |
| `internal/api/udsapi` | Modified | Register UDS routes | Thin transport registration only |
| `internal/cli` | Modified/consumed | Agent-manageable `agh tool` and `agh toolsets` commands; existing `agh mcp auth` remains the MCP credential path | Add structured output and UDS client methods; link diagnostics to existing auth commands |
| `internal/settings` | Consumed | Existing MCP server list includes redacted `auth_status` | Reuse status shape for operator diagnostics; no duplicate settings status model |
| `internal/skills` | Modified | Skill list/search/view tools call into skills registry; skill MCP sidecars may inform external MCP descriptor sources | Preserve skill sidecar symlink hardening and `allowed_marketplace_mcp` trust filtering |
| `internal/network` | Modified | Network tools call peers/send through existing service | Ensure mutating calls enforce policy |
| `internal/task` | Modified | Bounded task tools call task service | Keep TaskManager authority model intact |
| `sdk/typescript` | Modified | Extension authors define tools using TypeScript functions | Add `extension.tool(...)`, schema digesting, `provide_tools`, and `tools/call` handler support |
| `sdk/go` | New | Extension authors define tools using Go functions in subprocess extensions | Add public Go extension SDK mirroring TypeScript tool-provider APIs |
| `sdk/create-extension` | Modified | Templates should scaffold executable tool providers | Add TypeScript and Go tool-extension templates plus manifest examples |
| `web/` | Modified | Settings/help surfaces may display registry policy state, tool diagnostics, and existing redacted MCP `auth_status` | Only render truthful daemon-backed status; no invented remote login controls |
| `packages/site` | Modified | Public docs for registry, extension tools, approval ceiling | Add docs and CLI reference updates |
| `.compozy/tasks/tools-registry/*` | New | Research, ADRs, final TechSpec | Keep analysis paths referenced in later tasks |

## Test Strategy

### Unit Tests

Test:

- `ToolID` validation, parsing, wildcard matching, and collision rejection.
- External name sanitization and fail-closed collision behavior, including one shared `Canonicalize(rawServer, rawTool)` fixture set.
- `Descriptor` validation and schema size limits.
- Availability state transitions and reason-code composition.
- Policy matrix across `deny-all`, `approve-reads`, and `approve-all`.
- Source defaults for built-in, extension, MCP, and dynamic tools.
- Agent allow/deny/toolset expansion.
- Session lineage concrete `ToolID` subset validation.
- Dispatch pipeline ordering.
- JSON schema input validation.
- Result truncation and redaction.
- Hook deny/patch/result behavior.
- Extension manifest backend validation.
- MCP auth status mapping from `internal/mcp/auth.StatusValue` to registry availability reason codes.
- Remote HTTP/SSE auth injection stays inside `internal/mcp`, performs at most one `internal/mcp/auth.Service.Refresh`, never starts a new login flow, and returns deterministic redacted reason codes when refresh is impossible.
- MCP server resource cloning/projection preserves `Transport`, `URL`, and `Auth` when remote MCP resources flow into registry diagnostics.
- Hosted MCP stdio proxy uses the `mcp-go` server path and proves `tools/list` / `tools/call` parity without manual MCP protocol fixtures.
- External stdio MCP call-through uses the `mcp-go` stdio client path and preserves AGH redaction/policy boundaries.
- External remote HTTP MCP call-through uses the explicit `mcp-go` streamable HTTP client path while keeping AGH auth ownership and redacted diagnostics.
- External remote SSE MCP call-through uses the explicit `mcp-go` SSE client path while keeping AGH auth ownership and redacted diagnostics.
- `mcp_server.transport = "sse"` remains accepted and maps to the library SSE client path; AGH must not silently reinterpret it as `http`.
- A focused unit test asserts the pinned `mcp-go` version in `go.mod` so unreviewed dependency drift is caught early.

Mocks are acceptable for provider I/O boundaries, but policy/dispatch correctness must be tested with real registry instances.

### Integration Tests

Test:

- Extension manifest declares a tool and it appears in operator registry projection.
- Disabling/removing an extension removes or marks the tool unavailable.
- TypeScript extension declares a manifest-authoritative `extension_host` tool, registers a matching SDK function, and dispatch succeeds through `Registry.Call`.
- Go extension declares a manifest-authoritative `extension_host` tool, registers a matching SDK function, and dispatch succeeds through `Registry.Call`.
- Extension runtime descriptor mismatches for handler, schema digest, risk flags, or missing `tool.provider` grant keep the tool operator-visible and session-hidden with deterministic reason codes.
- MCP-backed extension tool resolves to an authorized MCP source and dispatch succeeds through the daemon-owned MCP adapter when auth, source policy, approval, hooks, and session lineage pass.
- MCP-backed extension tool remains operator-visible but unavailable when its backend server is missing, unauthorized, unhealthy, unauthenticated, or conflicted.
- Remote MCP server with `needs_login`, `expired`, and `invalid` auth statuses appears only in operator diagnostics with redacted `MCPAuthStatus` and is hidden from session/model projections.
- Existing `agh mcp auth status --refresh <server> -o json` changes operator diagnostics without requiring a registry-owned OAuth flow.
- `agh tool info <mcp-tool>` and `GET /api/tools/{id}` show redacted MCP auth diagnostics that match `/api/settings/mcp-servers` `auth_status` for the same server.
- Remote OAuth token values never appear in tool CLI/API/UDS/MCP responses, SSE payloads, event payloads, logs, or process diagnostics.
- A fake remote MCP server that requires `Authorization` proves the header is injected only inside `internal/mcp` and never appears in `internal/tools` errors, logs, events, or result envelopes.
- If an upstream remote MCP server emits `notifications/tools/list_changed`, AGH treats it as cache invalidation only and still refreshes descriptors on-demand before mutating registry projection.
- Hosted MCP bind nonces never grant access without UDS peer credentials, and AGH-owned diagnostics never describe them as claim-token-equivalent bearer secrets.
- Hosted MCP binding fails closed when UDS peer credentials or executable validation are unavailable.
- Remote MCP configs are not converted to blank stdio ACP servers; hosted-session injection remains the AGH stdio proxy while remote MCP calls happen inside the daemon MCP adapter.
- Built-in `agh__skill_view` calls real skills registry content.
- CLI/HTTP/UDS list/search/info parity.
- `agh tool mcp --session <id> --bind-nonce <nonce>` `tools/list` matches session projection.
- Hosted MCP rejects a proxy bind without the session-bound bind nonce plus successful UDS peer credential validation.
- Hosted MCP derives workspace id from session id and rejects client-supplied workspace context.
- Hosted MCP routes approval-required calls through ACP `session/request_permission` when available and fails closed with `approval_unreachable` when unavailable.
- Hosted MCP approval-required calls time out with `approval_timed_out` and cancel with `approval_canceled` when the proxy disconnects mid-approval.
- Mid-session projection changes such as extension disable or MCP auth degradation propagate through the hosted MCP projection stream, fire library `tools/list_changed`, and keep `tools/list` equal to `GET /api/sessions/{id}/tools`.
- `approve-reads` exposes read-only tools but blocks mutating tools without approval.
- Mutating, destructive, and open-world extension/MCP tools execute only with explicit `ToolID`/toolset/source/agent grants plus ACP ceiling, approval bridge, session lineage, and hook revalidation.
- `approve-reads` does not auto-approve external read-only tools from untrusted extension/MCP sources.
- `approve-all` does not bypass explicit registry denies or session lineage narrowing.
- `deny-all` blocks execution while operator surfaces still show diagnostics.
- The concrete MVP task tools call only their listed `task.Service` methods; excluded claim/complete/release tools are absent.
- `agh__task_child_create` cannot widen child permissions beyond parent lineage because `task.Service.CreateChildTask` remains the enforcement point.
- Hooks can deny pre-call and redact post-call output.
- Conflicted tool IDs are operator-visible and session-hidden.
- Result budget truncation is identical across CLI, HTTP, UDS, and MCP.

### E2E Tests

Use the AGH runtime E2E harness:

- Start daemon with isolated `AGH_HOME`.
- Create a session with hosted AGH MCP enabled.
- Verify the agent session receives registry-backed MCP tools.
- Invoke a safe built-in tool through the hosted MCP path.
- Verify events, telemetry, CLI status, HTTP status, and UDS status agree.
- Install a test extension publishing a tool descriptor and MCP backend metadata.
- Install TypeScript and Go test extensions publishing executable `extension_host` tools.
- Verify operator diagnostics, runtime reconciliation, session visibility, successful invocation, disable/remove lifecycle, and conflict handling.
- Configure a local MCP test server and verify descriptor discovery plus a real `tools/call` through the daemon MCP adapter.
- Configure a remote OAuth-backed MCP server in isolated `AGH_HOME`, exercise `agh mcp auth login/status/logout` against a local OAuth test server, and verify registry tool diagnostics never expose access tokens, refresh tokens, authorization codes, PKCE verifiers, or approval tokens.
- acpmock fixtures and Playwright selectors for hosted MCP tool-call lifecycle ship in the same PR; matchers use structured `tool_id` metadata, never rendered prompt substrings.
- Per-package coverage must stay at or above 80%; race-sensitive packages run under `-race` with `CGO_ENABLED=1` in the Linux-Race CI lane.

Run full `make verify` before completing implementation tasks.

## Implementation Steps

### Build Order

Ordered implementation sequence respecting dependencies:

1. Add canonical `ToolID`, descriptor, backend kind, source, availability, result, and error contracts in `internal/tools` - no dependencies.
2. Replace metadata-only provider semantics with executable runtime provider/handle interfaces - depends on step 1.
3. Implement registry indexing, collision detection, MCP auth diagnostic mapping, and operator/session projections - depends on steps 1-2 and existing `internal/mcp/auth`.
4. Implement policy evaluator with ACP ceiling, agent policy, session lineage, source defaults, allow/deny, and toolsets - depends on step 3.
5. Implement dispatch pipeline with schema validation, availability recheck, hooks, budgets, handle call, normalization, and telemetry seams - depends on steps 3-4.
6. Add built-in provider for bootstrap AGH tools: `agh__tool_*`, `agh__skill_*`, `agh__network_*`, and only the enumerated MVP task tools - depends on step 5.
7. Add extension manifest backend metadata, manifest-authoritative validation, and runtime reconciliation contracts - depends on steps 1-3 and existing extension resources.
8. Add extension protocol capability `tool.provider`, wire-stable `provide_tools`/`tools/call` request-response structs, schema digest conformance fixtures, and invocation through the existing subprocess manager - depends on steps 5 and 7.
9. Add TypeScript SDK `extension.tool(...)`, schema digesting, and handler registration - depends on step 8.
10. Add public Go extension SDK with function-based tool helpers equivalent to TypeScript SDK - depends on step 8.
11. Add daemon-owned MCP descriptor discovery and `MCPCallExecutor` call-through adapter using existing MCP config/auth, token redaction boundaries, `mcp-go` version `v0.49.0`, canonical `Canonicalize(rawServer, rawTool)` normalization, and explicit stdio/streamable-HTTP/SSE client transports - depends on steps 3-5 and existing `internal/mcp/auth`.
12. Add hosted MCP stdio proxy command `agh tool mcp --session --bind-nonce`, implemented with `mcp-go` server/tool APIs over `server.ServeStdio`, exact descriptor schema bytes through `RawInputSchema`/`RawOutputSchema`, a daemon→proxy projection stream, and UDS peer-credential bind plus approval bridge timeout/cancellation and existing MCP resource/auth preservation - depends on steps 3-6 and 11.
13. Add API contract DTOs and `internal/api/core` handlers - depends on steps 3-6, 8, and 11.
14. Wire HTTP, UDS, CLI commands, and UDS client methods - depends on step 13.
15. Wire hooks and canonical `tool_id` payload updates end-to-end, including typed hook payloads, matchers, fixture builders, extension-author docs, and no dual identity mid-PR - depends on step 5.
16. Add config lifecycle, validation, generated docs, CLI docs, site docs, integration/E2E coverage, and run `make verify` - depends on all prior steps.

### Technical Dependencies

Blocking dependencies that must be resolved before implementation:

- Existing ACP `permissions.mode` behavior in `internal/acp/permission.go`.
- Existing session lineage permission atoms in `internal/store/session_lineage.go`.
- Existing extension resource publication and `resources.publish.families = ["tools"]`.
- Existing extension subprocess lifecycle, JSON-RPC `process.Call`, Host API capability checks, and TypeScript `Extension.handle(...)` handler pattern.
- Existing hooks payload system.
- Existing MCP server configuration/resource lifecycle in `internal/config/provider.go`, `internal/config/mcpjson.go`, `internal/config/mcp_resource.go`, `internal/skills/mcp.go`, and `internal/extension/resource_publication.go`.
- Existing MCP auth lifecycle in `internal/mcp/auth`, `internal/store/globaldb/global_db_mcp_auth.go`, `internal/cli/mcp_auth.go`, `internal/settings`, and `internal/daemon/settings.go`.
- Existing skills registry APIs.
- Existing task/network services.
- OpenAPI/codegen path for web contract updates.

### Safety Invariants

1. Every AGH-owned tool call enters `internal/tools.Registry.Call`; CLI, HTTP, UDS, hosted MCP, extension, and session paths cannot bypass the registry dispatch pipeline.
2. Dispatch recomputes availability and `EffectiveToolDecision` at call time, even when discovery already filtered the tool.
3. ACP `permissions.mode` is always a ceiling. Registry, source, agent, session, and hook policy can narrow authority but cannot raise it above the effective ACP/session mode.
4. `approve-all` skips approval prompts only for otherwise allowed tools; it does not bypass explicit denies, source grants, session lineage, conflicts, unavailable backends, or hooks.
5. `approve-reads` applies only to registry-classified read-only tools. Extension/MCP read-only tools also require an explicit trusted source or per-tool/toolset/source/agent grant. Mutating, destructive, open-world, network-send, and task-write tools cannot inherit read approval by display title or ACP kind.
6. Session lineage stores concrete canonical `ToolID` atoms after toolset expansion; child sessions can only receive a subset of parent concrete atoms.
7. Tool ID collisions fail closed. A conflicted tool is operator-visible with reason codes and absent from session/model-visible projections.
8. Extension-installed tools can become executable in MVP only when the manifest-authoritative descriptor, source policy, extension health, runtime `provide_tools` descriptor, and `tools/call` handler all agree.
9. Third-party extension tool handlers never run in-process in the daemon during MVP; TypeScript and Go function handlers run inside supervised extension subprocesses.
10. Hooks dispatch at the call site and cannot tail event tables, spawn parallel queues, or mutate durable ownership state outside typed hook contracts.
11. Tool result limiting and redaction run before results cross CLI, HTTP, UDS, MCP, SSE, logs, memory, or event payloads.
12. Raw `claim_token`, MCP auth tokens, OAuth codes, PKCE verifiers, and secret bindings never appear in tool inputs/outputs persisted or emitted by AGH-owned surfaces.
13. Hosted MCP `tools/list` is a projection of `GET /api/sessions/{id}/tools`; divergence is a test failure.
14. Operator-visible diagnostics are not security boundaries. Hidden session projection plus dispatch-time revalidation is the security boundary.
15. Toolsets are expanded deterministically and cycle-checked before session projection; recursive expansion cannot happen lazily during dispatch.
16. Hosted MCP cannot bind to a session without a daemon-minted, single-use, session-bound bind nonce plus successful UDS peer credential and AGH binary validation. The nonce is not a bearer secret, is never accepted from tool input, and its launch record is invalidated on bind/session end/disconnect or `[tools.hosted_mcp].bind_nonce_ttl_seconds`.
17. Hosted MCP `tools/call` cannot pass an `approval_required` decision unless the daemon completes an ACP/session-mediated approval equivalent to CLI/HTTP/UDS approval semantics within `[tools.policy].approval_timeout_seconds`; timeout and proxy disconnect return `approval_timed_out` or `approval_canceled`.
18. No `agh__task_*` tool may bypass `task.Service.ClaimNextRun`, `Spawn`, session manager terminal-state authority, or task lifecycle authority. Claim/release/complete/fail/run-start operations are excluded from MVP tools.
19. External `extension_host` and `mcp` backend tools are executable only through their registered runtime handles; missing handlers, missing MCP clients, missing capabilities, source denies, auth failures, or runtime mismatches fail closed before user code or remote tools run.
20. Remote MCP OAuth/PKCE credentials are owned only by `internal/mcp/auth` and its `TokenStore`; the registry may consume redacted status and call through a narrow `internal/mcp/auth`-owned interface, but cannot persist, log, refresh, revoke, or copy access/refresh tokens. Raw tokens stay out of descriptors, resources, events, API responses, CLI output, MCP responses, and tool results.
21. Hosted MCP bind nonces and remote MCP OAuth tokens have separate issuers, storage, lifetimes, redaction labels, and failure codes. A `hosted_mcp_bind_nonce` is not sufficient to bind an AGH hosted MCP proxy without UDS peer credential validation, must never satisfy a remote MCP server auth check, and a remote MCP OAuth token must never bind an AGH hosted MCP proxy.
22. Hosted MCP `mcp.Tool` registration uses `Descriptor.input_schema` and `Descriptor.output_schema` bytes as authoritative through `RawInputSchema` and `RawOutputSchema`. `WithInputSchema`, `WithOutputSchema`, and reflection-based schema generation helpers are forbidden for AGH-hosted tools.
23. AGH must not use `client.NewOAuthStreamableHttpClient`, `client.NewOAuthSSEClient`, default `transport.NewOAuthHandler`, `MemoryTokenStore`, or any library-owned token cache/login/refresh flow for remote MCP credentials. Remote auth remains `internal/mcp/auth` owned, outbound headers are injected only inside `internal/mcp`, and retry logic may attempt at most one `internal/mcp/auth.Service.Refresh`; it must never start a new login flow.
24. Hosted MCP must maintain a live daemon→proxy projection stream after bind. If the stream fails, the proxy closes the MCP session rather than serving stale tools.
25. AGH-owned approval timeouts govern hosted MCP approval waits; library-side request deadlines must not preempt a still-valid daemon approval bridge wait. Hosted MCP request deadlines therefore include at least a 5-second guard band above `[tools.policy].approval_timeout_seconds`, while remote MCP outbound deadlines are derived independently.
26. Post-change remote MCP config accepts `stdio`, `http`, and `sse`. `http` maps to the explicit `mcp-go` streamable HTTP client path, `sse` maps to the explicit `mcp-go` SSE client path, and neither transport may be silently rewritten into the other.
27. CLI/HTTP/UDS `approval_token` values are daemon-issued, single-use, bound to one `tool_id` + `session_id` + `workspace_id` + `input_digest`, stored only as hashes in daemon memory with TTL `[tools.policy].approval_timeout_seconds`, and may traverse only the authenticated issuance response plus the matching invoke request. They never appear in logs, events, SSE, hosted MCP, persisted state, or diagnostics.

## Monitoring and Observability

Events:

- `tool.registry.refresh_started`
- `tool.registry.refresh_completed`
- `tool.registered`
- `tool.updated`
- `tool.removed`
- `tool.conflicted`
- `tool.availability_changed`
- `tool.policy_evaluated`
- `tool.call_started`
- `tool.call_completed`
- `tool.call_failed`
- `tool.call_denied`
- `tool.result_truncated`

Required fields:

- `tool_id`
- `display_title`
- `source_kind`
- `source_owner`
- `workspace_id`
- `session_id`
- `parent_session_id`
- `root_session_id`
- `agent_name`
- `risk`
- `read_only`
- `destructive`
- `open_world`
- `approval_mode`
- `decision`
- `reason_codes`
- `duration_ms`
- `result_bytes`
- `truncated`
- `correlation_id`

Metrics:

- registered tools by source kind
- conflicted tools by source kind
- available/callable tools by session
- calls started/completed/failed/denied
- approval-required counts
- result truncation counts
- backend latency by source kind
- hook-denied counts
- policy-denied counts

Redaction:

- Never log raw claim tokens, MCP auth tokens, OAuth codes, PKCE verifiers, secret bindings, or full tool payloads when marked sensitive.
- Use digests for large inputs/results.
- Preserve enough metadata to debug policy and availability without exposing secrets.

## Technical Considerations

### Key Decisions

Decision: Tool Registry is an AGH daemon runtime service, not an ACP registry.

Rationale: ACP has no callable tool registry and relies on MCP for tool discovery.

Trade-off: AGH must maintain its own registry semantics and expose them through MCP/session surfaces.

Decision: Use one canonical `ToolID` with `__` namespace separators.

Rationale: Avoid dotted/internal plus wire alias ambiguity and stay compatible with stricter provider naming limits.

Trade-off: `agh__skill_view` is less visually elegant than `agh.skill.view`, but it prevents dual identity bugs.

Decision: Built-ins execute in-process; TypeScript and Go extension tools execute out-of-process through `extension_host`.

Rationale: Daemon safety and extension isolation matter more than plugin convenience.

Trade-off: Extension tool latency and setup are higher, but failure containment is better and extension authors still get function-based APIs.

Decision: Remote MCP-backed tools execute in MVP through daemon-owned MCP clients.

Rationale: MCP config/auth already exists and the registry must not become useful only for descriptors.

Trade-off: The MVP must implement stricter auth redaction and adapter tests, but operators get one coherent tool model.

Decision: ACP `permissions.mode` is the approval ceiling.

Rationale: Avoid contradictory policy layers and keep existing settings truthful.

Trade-off: Registry policy must compute effective decisions rather than simple allow/deny flags.

Decision: Operator and session projections differ.

Rationale: Operators need diagnostics; models should see only callable tools.

Trade-off: More projection logic, but less model confusion and fewer unsafe calls.

Decision: Keep cold resource records separate from runtime handles.

Rationale: Resource/projector system is good for installed metadata; executable dispatch needs function/protocol handles and live health.

Trade-off: More types, but cleaner lifecycle and safer extension support.

### Known Risks

Risk: Mutating tools mislabeled as read-only.

Mitigation: Descriptor validation, review tests, policy matrix tests, and hook/audit visibility.

Risk: External tool name collisions force late breaking changes.

Mitigation: Enforce canonical `ToolID` grammar and fail-closed collision handling in MVP.

Risk: Hosted MCP path diverges from CLI/HTTP/UDS behavior.

Mitigation: Hosted MCP proxies through UDS into the same registry dispatch path.

Risk: Extension tools become visible before safe runtime execution is confirmed.

Mitigation: Operator-visible only until manifest/runtime reconciliation, availability, source policy, and backend handle all pass.

Risk: `approve-all` is misunderstood as "execute everything."

Mitigation: Docs and UI copy must clarify it auto-approves otherwise allowed calls; it does not bypass explicit denies, source grants, availability, lineage, or hooks.

Risk: Toolsets create ambiguous policy language.

Mitigation: Keep `tools` and `toolsets` as separate typed fields; expand toolsets to concrete `ToolID`s for session lineage.

Risk: Result payloads leak secrets or overwhelm context.

Mitigation: Central result limiter, redaction metadata, digesting, and output budgets.

### Delete Targets

Because AGH is greenfield alpha, the implementation should hard-cut ambiguous old tool concepts instead of adding compatibility bridges:

- Replace metadata-only `ToolProvider.Tools(ctx)` as the runtime extension point with provider/descriptor/handle contracts.
- Remove descriptor-only MVP wording and `backend_not_implemented` behavior for `extension_host` and `mcp` tools.
- Remove any public standalone `subprocess` backend in favor of `extension_host` subprocess isolation.
- Replace `internal/tools.Tool.Name` as a registry identity with canonical `ToolID` in new public contracts.
- Remove any new public use of dotted tool IDs or separate MCP wire aliases.
- Delete legacy dotted-form `ToolID` atoms in `session_lineage.Tools`. No normalization shim is allowed; pre-TechSpec local databases require a fresh `AGH_HOME`.
- Remove any new forced hard-cut of `mcp_server.transport = "sse"` or any compatibility shim that silently rewrites `sse` to `http`; the runtime must either support the declared transport explicitly or fail with a typed configuration error.
- Delete any new hand-rolled MCP protocol/session/transport implementation for hosted MCP or remote MCP call-through in favor of the `mark3labs/mcp-go` wrapper inside `internal/mcp`, including locally-authored MCP message structs, ad-hoc stdio framers, or custom bearer-header injectors outside `internal/mcp/auth`.
- Rewrite `internal/daemon/tool_mcp_resources.go:cloneDaemonMCPServer` so registry/resource projections preserve `Transport`, `URL`, and `Auth` instead of leaving a partial MCP clone path in tree.
- Replace hook policy identity based on `tool_name` + `tool_namespace` with canonical `tool_id` for registry-owned tool calls, including `internal/hooks/payloads.go` `ToolPreCallPayload`, `ToolPostCallPayload`, and `ToolPostErrorPayload`.
- Update docs, tests, CLI/API examples, and task artifacts that refer to dotted IDs or dual aliasing.

## Architecture Decision Records

- [ADR-001: Extension Tool Execution Boundary](adrs/adr-001-extension-tool-execution-boundary.md) - extension tools are manifest-first, executable, and out-of-process in MVP.
- [ADR-002: Session Tool Exposure Path](adrs/adr-002-session-tool-exposure-path.md) - expose AGH registry tools through hosted local MCP plus shared CLI/HTTP/UDS.
- [ADR-003: Runtime Registry Package Boundary](adrs/adr-003-runtime-registry-package-boundary.md) - `internal/tools` owns runtime registry and dispatch; `internal/catalog` remains thin.
- [ADR-004: MVP Native Tool Scope](adrs/adr-004-mvp-native-tool-scope.md) - bootstrap catalog/skill tools plus selected network/task tools.
- [ADR-005: ACP Approval Policy Integration](adrs/adr-005-acp-approval-policy-integration.md) - ACP approval mode is the system/session ceiling.
- [ADR-006: Tool Visibility By Surface](adrs/adr-006-tool-visibility-by-surface.md) - operator projections show diagnostics; model projections show callable tools only.
- [ADR-007: Canonical Tool ID Format](adrs/adr-007-canonical-tool-id-format.md) - one provider-safe `ToolID` using reserved `__` namespace separators.
- [ADR-008: Manifest-Authoritative Extension Tool Descriptors](adrs/adr-008-manifest-authoritative-extension-tool-descriptors.md) - `extension.toml` is source of truth and runtime descriptors reconcile against it.
- [ADR-009: Public Go Extension Tool SDK](adrs/adr-009-public-go-extension-tool-sdk.md) - Go extensions get function-based subprocess SDK APIs equivalent to TypeScript.
- [ADR-010: Remote MCP Call-Through](adrs/adr-010-remote-mcp-call-through.md) - remote MCP tools are executable in MVP through daemon-owned MCP adapters.
- [ADR-011: Use `mark3labs/mcp-go` For MCP Protocol And Transport](adrs/adr-011-mark3labs-mcp-go.md) - hosted MCP and remote MCP call-through use `mcp-go` instead of hand-rolled protocol plumbing.

## Nits

Peer review round 1 nits and disposition:

- `N-001` ToolsetID grammar: addressed in Data Models by sharing the `ToolID` grammar.
- `N-002` Tool pattern grammar: addressed in Config Lifecycle with explicit allowed/disallowed pattern forms.
- `N-003` `approval_token` semantics: addressed in API Endpoints and Hosted MCP Approval Bridge.
- `N-004` `dynamic` source kind: addressed in Data Models as reserved with no MVP producer.
- `N-005` hosted MCP lifecycle: addressed in Hosted MCP lifecycle.
- `N-006` `agh__skill_view` result budget: addressed in Integration Points / Skills.
- `N-007` hook identity migration co-ship: addressed in Implementation Steps step 11.
- `N-008` invalid existing session lineage atoms: addressed in Config Lifecycle / Agent Definitions.
- `N-009` hosted MCP workspace resolution: addressed in Hosted MCP lifecycle/authentication.
- `N-010` MVP tool risk classification: addressed in Network And Tasks tables.

Peer review round 2 blockers and nits disposition:

- `B-001` Extension wire contracts: addressed in Core Interfaces with protocol constants, capability-method mapping, and `provide_tools` / `tools/call` request-response structs.
- `B-002` Schema digest canonicalization: addressed in Data Models with RFC 8785 JCS canonicalization, lowercase SHA-256 digests, and shared SDK/daemon fixtures.
- `B-003` Remote MCP bearer injection: addressed in Core Interfaces and Existing MCP Config/Auth with `MCPCallExecutor` owned by `internal/mcp`.
- `B-004` Hosted MCP bind-token contradiction: addressed by replacing bearer bind tokens with non-secret bind nonces plus UDS peer credential and AGH binary validation.
- `B-005` Approval bridge wait: addressed with `[tools.policy].approval_timeout_seconds`, `approval_timed_out`, `approval_canceled`, and proxy-disconnect cancellation behavior.
- `N-001` Approval timeout and bind nonce TTL defaults: addressed in Config Lifecycle and Safety Invariants.
- `N-002` Go SDK path: addressed in ADR-009 by committing to `sdk/go`.
- `N-003` Runtime contract fixture updates: addressed in Test Strategy with acpmock and Playwright fixture requirements.
- `N-004` Coverage/race discipline: addressed in Test Strategy.
- `N-005` Long sanitized external IDs: addressed in ToolID and reason codes with `id_too_long`.
- `N-006` Hook payload delete targets: addressed in Delete Targets.
- `N-007` External read-only trust: addressed in Config Lifecycle with `trusted_sources`.
- `N-008` Child task lineage authority: addressed in Network And Tasks and Integration Tests.

Peer review round 6 blockers and nits disposition:

- `B-001` `sse` validation contradiction: addressed in Config Lifecycle / Validation by preserving the explicit `{stdio, http, sse}` transport set and forbidding silent transport rewrites.
- `B-002` undefined `approval_token` producer: addressed in API Endpoints with `POST /api/tools/{id}/approvals`, CLI/UDS parity, typed issuance rules, and Safety Invariant 27.
- `N-001` launch-record storage: addressed in Hosted MCP authentication and Config Lifecycle by stating launch records are daemon-memory-only.
- `N-002` min/max bounds: addressed in Config Lifecycle / Validation with explicit TTL and timeout bounds.
- `N-003` legacy dotted session-lineage atoms: addressed in Delete Targets.
- `N-004` `dynamic` source rejection: addressed in Config Lifecycle / Validation.
- `N-005` `internal/catalog` conditional: accepted as non-blocking; MVP keeps `internal/catalog` optional and composition-facing only.
- `N-006` approval/projection events: deferred as non-blocking observability enrichment for implementation tasks.
- `N-007` projection-stream wire spec: deferred as a non-blocking implementation-detail refinement; MVP already requires a live daemon→proxy stream plus fail-closed behavior.
- `N-008` `agh__task_child_create` lineage verification: deferred to implementation task coverage because the TechSpec already requires subset-preserving lineage semantics and integration tests.
- `N-009` real-scenario QA helper co-ship: accepted as non-blocking because the task pipeline already carries the mandatory QA pair and runtime/acpmock coverage requirements.
- `N-010` early MCP clone rewrite step: accepted as non-blocking because step 11 already makes `cloneDaemonMCPServer` correction part of the MCP execution workstream and the delete targets call it out explicitly.
