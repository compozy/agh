# Tool Registry Canonical Surface TechSpec

## Executive Summary

This TechSpec defines the final canonical AGH tool surface as a follow-up over
the `tools-registry` foundation that is already implemented on this branch. The
problem is no longer registry existence. The current branch already ships the
registry core, policy engine, hosted MCP transport, approval bridge,
CLI/HTTP/UDS tool surfaces, and an intentionally narrow built-in MVP subset.
The remaining problem is surface ambiguity: AGH still has a partial built-in
tool surface, a broader CLI surface, no startup prompt section for tools, and
no final rule for which internal capabilities agents should access through
tools versus shell commands. This follow-up design removes that ambiguity.

There is no `_prd.md` for `tools-refac`. The source of truth for this TechSpec
is the accepted ADRs under `.compozy/tasks/tools-refac/adrs/`, the original
`.compozy/tasks/tools-registry/_techspec.md`, competitor references under
`.resources/`, code exploration of current prompt/task/MCP seams, and the prior
design discussion captured in ledgers.

Current branch baseline:

- `internal/tools` already owns `RuntimeRegistry`, `PolicyEvaluator`,
  result-limiting, provider dispatch, and the built-in/toolset catalog.
- `internal/daemon` already wires hosted MCP exposure, approval bridging, and
  daemon-native built-ins.
- The currently shipped AGH-native built-ins are limited to
  `agh__bootstrap`, `agh__catalog`, `agh__coordination`, and `agh__tasks`.

The primary trade-off is deliberate: AGH will spend more implementation effort
extending the shipped built-in surface, prompt guidance, and dynamic
policy-input resolution instead of leaning on CLI shell-out paths as an
implicit fallback. CLI remains a first-class operator surface, but AGH-internal
runtime capabilities become tool-first by convention for agents, including most
management-grade mutable operations. Interactive management flows such as MCP
OAuth login/logout and trust-root bootstrap mutations remain on operator
surfaces rather than being forced into the normal tool-call loop.

## MVP Boundary Statement

MVP boundary for this follow-up: implementation steps 1-10 in this document
extend the already-shipped tool-registry foundation into the canonical
agent/operator surface. That boundary includes default discovery toolsets, a
tools startup prompt section, a bundled `agh-tools-guide`, policy-input
resolution that incorporates current agent/session lineage, expanded AGH
built-in coverage for runtime domains, session-bound autonomy execution tools,
status-only MCP auth visibility for agents, CLI/HTTP/UDS contract alignment,
and the related docs/tests hard cuts. These ten steps are one merge unit. AGH
must not land a partial merge where raw-`claim_token` autonomy contracts are
removed from one surface while hosted MCP, docs, OpenAPI, CLI reference pages,
or transport parity still describe the old contract.

This TechSpec intentionally does **not** redefine the `tools-registry`
foundation. It reuses:

- canonical `ToolID` and `ToolsetID` grammar;
- hosted MCP as the model-visible session exposure path;
- daemon-owned MCP adapters and existing MCP auth/token storage;
- extension-host execution and manifest-authoritative descriptors;
- shared list/search/get/call projections and transport parity rules.

Post-MVP work deferred to later TechSpecs:

- driver-specific shell restrictions or ACP-driver Bash changes;
- direct ACP-native tool injection outside hosted MCP;
- remote peer tool execution over AGH Network;
- relaxing the one-active-lease-per-session autonomy invariant;
- converting MCP OAuth login/logout into fully self-healing agent flows;
- any bridge/SDK redesign outside the existing registry foundation.

Explicitly out of scope for this TechSpec:

- blocking `agh` inside shell tools;
- agent-callable spawn, cross-session terminal-state mutation, or daemon
  lifecycle control;
- raw `claim_token` in any AGH-owned tool, CLI, HTTP, UDS, SSE, log, or memory
  surface;
- partial delivery where built-in tools ship without prompt guidance, policy,
  docs, and transport parity;
- compatibility aliases that preserve the old ambiguous surface.

## Architectural Boundaries

`internal/daemon` remains the only composition root. This TechSpec extends the
existing registry composition; it does not introduce a parallel tool catalog,
parallel policy engine, or parallel task-execution coordinator.

Package boundaries:

- `internal/tools` owns tool descriptors, toolsets, operator/session
  projections, effective-policy recomputation, runtime default toolset overlay,
  dispatch, result budgeting, and built-in tool registration. It must remain
  transport-agnostic and must not import `internal/api/*` or `internal/cli`.
- `internal/daemon` wires the registry, hosted MCP exposure, prompt sections,
  session scope, auth/status adapters, and domain services. If new registry
  collaborators are introduced, they are injected here.
- `internal/skills` owns bundled skill assets and skill catalog prompt
  rendering. The new tools guidance content lives under bundled skills; the
  tools prompt section is composed by the daemon, not by the skills package.
- `internal/task` remains the single authority for claim, heartbeat, release,
  completion, and failure. The new autonomy tools may narrow or route calls, but
  they must not duplicate lease ownership logic.
- `internal/mcp/auth` remains the only owner of OAuth login/logout, refresh, and
  token storage. The new agent-visible MCP auth tool consumes redacted status
  only.
- `internal/session` remains the source of lineage and effective session scope.
  It must not implement its own tool policy or its own discovery defaults.
- `internal/api/core`, `internal/api/httpapi`, `internal/api/udsapi`, and
  `internal/cli` remain thin transport layers over the same registry or
  management services.
- `packages/site` owns the public docs and CLI reference updates. `web/` is
  impacted only if existing typed contracts or settings copy need adjustment; no
  new runtime UI is required by this spec.

YAGNI rule for this follow-up: do not add a new broad package such as
`internal/catalog` unless implementation proves the existing `internal/tools` +
`internal/daemon` split cannot absorb the work cleanly. The default design is
to extend current packages and current seams.

## System Architecture

### Component Overview

| Component | Responsibility | Boundary |
|---|---|---|
| Runtime default discovery overlay | Ensures every agent gets `agh__bootstrap` and `agh__catalog` unless effective policy narrows them | Computed inside `internal/tools`; not persisted into every agent definition |
| Tools startup prompt section | Teaches discovery and invocation loop at session startup | Composed in `internal/daemon` using bundled skill content |
| Bundled `agh-tools-guide` | Canonical written guidance for AGH tool usage and operator/tool split | Asset under `internal/skills/bundled/skills/` |
| Dynamic policy resolver | Recomputes effective policy for list/search/get/call using current runtime state | Shared by projections and dispatch; dispatch remains authoritative |
| Built-in domain tools | Covers AGH runtime capabilities that agents should use at runtime | Registered in `internal/tools` and routed into existing domain services |
| Session-bound autonomy bridge | Exposes claim/heartbeat/complete/fail/release as tools without leaking raw claim tokens | Routes into `internal/task` writers via daemon-resolved lease state |
| MCP auth status adapter | Exposes redacted auth status to agents and operator diagnostics | Reads `internal/mcp/auth` status only; login/logout stay outside tools |
| Operator management surfaces | Retain broader lifecycle/config/auth commands and endpoints | CLI/HTTP/UDS surfaces outside the normal tool family |

Data flow:

1. Session startup resolves prompt sections and now includes `tools` alongside
   `situation`, `memory`, `skills`, and `network`.
2. The agent sees default discovery tools even when the agent definition omits
   explicit `tools`/`toolsets`.
3. `list`, `search`, `get`, and hosted MCP projections call the same effective
   policy resolver that `call` will later re-run.
4. Tool dispatch routes into existing domain authorities: skills, network,
   tasks, config, workspace, hooks, automation, extensions, and MCP auth status.
5. Autonomy execution tools resolve lease state server-side and call the same
   task lease writers used by manual/operator paths.
6. Operator surfaces continue to own interactive or cross-session management
   operations such as OAuth login/logout and session lifecycle.

## Implementation Design

### Core Interfaces

```go
type Scope struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	Operator    bool   `json:"operator,omitempty"`
}
```

```go
type Registry interface {
	List(ctx context.Context, scope Scope) ([]ToolView, error)
	Search(ctx context.Context, scope Scope, q SearchQuery) ([]ToolView, error)
	Get(ctx context.Context, scope Scope, id ToolID) (ToolView, error)
	Call(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error)
}
```

```go
type PolicyInputResolver interface {
	Resolve(ctx context.Context, scope tools.Scope) (tools.PolicyInputs, error)
	DefaultToolsets(ctx context.Context, scope tools.Scope) ([]tools.ToolsetID, error)
}
```

```go
type AutonomyLeaseLookup interface {
	LookupActiveRunForSession(
		ctx context.Context,
		sessionID string,
		runID string,
	) (AutonomyLeaseHandle, error)
}

type AutonomyLeaseHandle struct {
	RunID          string
	SessionID      string
	Status         task.RunStatus
	LeaseUntil     time.Time
	ClaimToken     string
	ClaimTokenHash string
	WorkspaceID    string
	AgentName      string
}
```

```go
type AutonomyLeaseBridge interface {
	ClaimNextRun(ctx context.Context, req AutonomyClaimRequest, actor task.ActorContext) (*task.ClaimResult, error)
	Heartbeat(ctx context.Context, req AutonomyHeartbeatRequest, actor task.ActorContext) (*task.Run, error)
	Complete(ctx context.Context, req AutonomyCompletionRequest, actor task.ActorContext) (*task.Run, error)
	Fail(ctx context.Context, req AutonomyFailureRequest, actor task.ActorContext) (*task.Run, error)
	Release(ctx context.Context, req AutonomyReleaseRequest, actor task.ActorContext) (*task.Run, error)
}
```

```go
type MCPAuthStatusProvider interface {
	Status(ctx context.Context, source tools.SourceRef) (tools.MCPAuthStatus, error)
}
```

Contract notes:

- `tools.Scope` remains the canonical registry-facing caller context in this
  follow-up. This spec must not introduce a second parallel scope type for
  operator, session, or hosted-MCP flows.
- `tools.Registry` and the existing projection/call path remain authoritative.
  This follow-up extends inputs and built-in coverage; it does not replace the
  shipped registry interface.
- `PolicyInputResolver` is a daemon-side collaborator that resolves
  scope-dependent `tools.PolicyInputs` and runtime default toolsets before the
  existing policy evaluator runs. Caches may wrap it, but no caller may
  substitute a stale projection as authority.
- `AutonomyLeaseBridge` is a routing adapter, not a new writer. It resolves the
  session-bound active lease and then calls the existing task service lease
  writers.
- `AutonomyLeaseLookup` is internal-only. `ClaimToken` inside
  `AutonomyLeaseHandle` never crosses tool, CLI, HTTP, UDS, SSE, settings, or
  persisted diagnostic boundaries.
- Existing `session.PromptProvider` remains the prompt assembly seam.
  `internal/skills` does not need a registry-specific prompt interface.

### Data Models

#### Canonical Built-In Surface

Current shipped subset on this branch:

| Toolset | Shipped built-in tools | Notes |
|---|---|---|
| `agh__bootstrap` | `agh__tool_list`, `agh__tool_search`, `agh__tool_info` | Already shipped |
| `agh__catalog` | `agh__skill_list`, `agh__skill_search`, `agh__skill_view` | Already shipped |
| `agh__coordination` | `agh__network_peers`, `agh__network_send` | Already shipped; intentionally narrow MVP bundle |
| `agh__tasks` | `agh__task_list`, `agh__task_read`, `agh__task_create`, `agh__task_child_create`, `agh__task_update`, `agh__task_cancel`, `agh__task_run_list` | Already shipped; excludes lease-transition verbs |

Canonical expansion delivered by this follow-up:

| Toolset | Built-in tools | Notes |
|---|---|---|
| `agh__bootstrap` | `agh__tool_list`, `agh__tool_search`, `agh__tool_info` | Default for every agent |
| `agh__catalog` | `agh__skill_list`, `agh__skill_search`, `agh__skill_view` | Default for every agent |
| `agh__memory` | `agh__memory_list`, `agh__memory_read`, `agh__memory_write`, `agh__memory_delete`, `agh__memory_search` | Dedicated AGH memory runtime surface |
| `agh__sessions` | `agh__session_list`, `agh__session_status`, `agh__session_history`, `agh__session_events`, `agh__session_describe` | Read-only only |
| `agh__workspace` | `agh__workspace_list`, `agh__workspace_info`, `agh__workspace_describe` | Read-only only |
| `agh__config` | `agh__config_show`, `agh__config_list`, `agh__config_get`, `agh__config_set`, `agh__config_unset`, `agh__config_diff`, `agh__config_path` | Agent-manageable config mutation except trust-root and secret paths |
| `agh__tasks` | `agh__task_list`, `agh__task_read`, `agh__task_create`, `agh__task_child_create`, `agh__task_update`, `agh__task_cancel`, `agh__task_run_list` | Task management, not lease execution |
| `agh__autonomy` | `agh__task_run_claim_next`, `agh__task_run_heartbeat`, `agh__task_run_complete`, `agh__task_run_fail`, `agh__task_run_release` | Identity-bound execution tools |
| `agh__coordination` | `agh__network_status`, `agh__network_channels`, `agh__network_inbox`, `agh__network_peers`, `agh__network_send` | Extends the existing shipped coordination toolset rather than renaming it |
| `agh__hooks` | `agh__hooks_list`, `agh__hooks_info`, `agh__hooks_events`, `agh__hooks_runs`, `agh__hooks_create`, `agh__hooks_update`, `agh__hooks_delete`, `agh__hooks_enable`, `agh__hooks_disable` | Agents can manage config/overlay-backed hooks; source-owned hooks remain structurally immutable |
| `agh__automation` | `agh__automation_jobs_list`, `agh__automation_jobs_get`, `agh__automation_jobs_create`, `agh__automation_jobs_update`, `agh__automation_jobs_delete`, `agh__automation_jobs_trigger`, `agh__automation_jobs_history`, `agh__automation_triggers_list`, `agh__automation_triggers_get`, `agh__automation_triggers_create`, `agh__automation_triggers_update`, `agh__automation_triggers_delete`, `agh__automation_triggers_history`, `agh__automation_runs_list`, `agh__automation_runs_get` | Automation management is tool-callable; policy and approval contain blast radius |
| `agh__extensions` | `agh__extensions_search`, `agh__extensions_list`, `agh__extensions_info`, `agh__extensions_install`, `agh__extensions_remove`, `agh__extensions_update`, `agh__extensions_enable`, `agh__extensions_disable` | Extension lifecycle is agent-manageable subject to trust-source policy and approval |
| `agh__mcp_auth` | `agh__mcp_auth_status` | Status only; login/logout excluded |
| `agh__observe` | `agh__observe_events`, `agh__observe_metrics`, `agh__observe_search` | Read-only observability |
| `agh__bridges` | `agh__bridges_list`, `agh__bridges_status` | Read-only bridge inspection |

Operator-only surfaces remain outside the agent tool family:

- session creation/spawn/stop of other sessions;
- daemon lifecycle and destructive runtime repair;
- `agh mcp auth login` and `agh mcp auth logout`;
- writes of raw secret material, OAuth credentials, PKCE material, provider API
  key env bindings, and MCP auth config secrets;
- daemon socket/host/port, sandbox backend/runtime-root/provider transport
  bootstrap, and similar trust-root config;
- cross-session force-stop, kill, or terminal-state mutation without explicit
  lineage-bound authority;
- any cross-session terminal-state mutation;
- internal hosted-MCP bootstrap command behavior beyond daemon launch.

#### Mutable Surface Policy

The final-state rule is:

`mutable does not imply operator-only`

If AGH already has a writer or a validated management command for a domain, the
default expectation is to expose that capability as a tool and contain risk
through scope, policy, approval, trust-source checks, and deterministic error
codes. Operator-only is reserved for bootstrap-root, raw-secret, or
human-interactive boundaries.

##### `agh__config`

Direction:

- `agh__config` becomes default-open for validated config overlay mutation,
  rather than allow-list-only.
- Agents may call `show`, `list`, `get`, `set`, `unset`, `diff`, and `path`.
- `set` and `unset` route through the same validated config writer as the CLI.

Agent-callable by default:

- most workspace/global overlay fields that do not cross trust-root or
  raw-secret boundaries;
- fields under `[defaults]`, `[session.*]`, `[memory.*]`, `[skills.*]`,
  `[automation.*]`, `[network.*]`, and non-secret parts of feature config
  that the writer can validate structurally.

Still operator-only:

- `[daemon]`, `[http]`, `[permissions]`;
- raw bind/socket/port paths that govern daemon bootstrap;
- `memory.global_dir`;
- `[providers.<name>]` command and API-key env binding;
- `[[mcp_servers]]` transport definitions and `auth.*` secret-bearing fields;
- `[sandboxes.<name>]` backend/runtime-root/provider bootstrap settings;
- `[log]` and observability retention/config that control audit posture;
- any path that writes raw secret material or changes AGH trust roots.

Policy:

- write scope remains `global` or `workspace` overlay only;
- `agh__config_set` and `agh__config_unset` require mutating approval;
- deterministic denials:
  `CONFIG_PATH_FORBIDDEN`, `CONFIG_SECRET_PATH_FORBIDDEN`,
  `CONFIG_TRUST_ROOT_FORBIDDEN`, `CONFIG_SCOPE_NOT_ALLOWED`,
  `CONFIG_VALIDATION_FAILED`.

##### `agh__automation`

Direction:

- Automation definition management is tool-callable, not CLI-only.
- Agents may manage jobs and triggers through the same validated automation
  writers already used by CLI and host APIs.

Agent-callable:

- `jobs_list`, `jobs_get`, `jobs_create`, `jobs_update`, `jobs_delete`,
  `jobs_trigger`, `jobs_history`;
- `triggers_list`, `triggers_get`, `triggers_create`, `triggers_update`,
  `triggers_delete`, `triggers_history`;
- `runs_list`, `runs_get`.

Still operator-only:

- direct writes of webhook secret material;
- bootstrap-time automation scheduler trust-root settings that belong in config
  trust-root surfaces rather than runtime job mutation.

Policy:

- mutations require mutating approval;
- workspace/global scope and source policy continue to apply;
- config-backed, package-backed, and dynamic definitions all remain manageable,
  but the same domain validators still enforce legal field sets;
- deterministic denials:
  `AUTOMATION_SCOPE_FORBIDDEN`, `AUTOMATION_SECRET_INPUT_FORBIDDEN`,
  `AUTOMATION_VALIDATION_FAILED`, `AUTOMATION_APPROVAL_REQUIRED`.

##### `agh__extensions`

Direction:

- Extension lifecycle is agent-manageable through tools.
- Search/install/update/remove/enable/disable are tool-callable and reuse the
  same extension registry and marketplace services as the CLI.

Agent-callable:

- `search`, `list`, `info`, `install`, `remove`, `update`, `enable`,
  `disable`.

Still operator-only:

- changing extension marketplace trust roots through raw config/bootstrap paths;
- writing secret marketplace credentials directly if a source ever requires
  them.

Policy:

- install/update/remove/enable/disable require mutating approval;
- source trust policy filters allowed registries, versions, local-path installs,
  and unsigned assets before projection and again at dispatch;
- deterministic denials:
  `EXTENSION_SOURCE_FORBIDDEN`, `EXTENSION_APPROVAL_REQUIRED`,
  `EXTENSION_NOT_INSTALLED`, `EXTENSION_VALIDATION_FAILED`.

##### `agh__hooks`

Direction:

- Hook management becomes tool-callable for mutable, AGH-owned hook
  declarations.
- Read, create, update, delete, enable, and disable all become first-class
  tools instead of leaving hook management implicit in config-file editing.

Agent-callable:

- `list`, `info`, `events`, `runs`;
- `create`, `update`, `delete`, `enable`, `disable` for config-backed or
  overlay-backed hook declarations.

Structurally immutable through agent tools:

- extension-provided or source-owned hook declarations whose owning source is
  not the current mutable overlay;
- raw secret values embedded in hook executor environment fields.

Policy:

- hook mutation requires mutating approval;
- hook declarations must still pass the existing hook normalization and
  validation pipeline;
- executor commands, args, matcher fields, timeouts, priorities, and required
  flags are mutable when the declaration source is mutable and no secret/trust
  root is crossed;
- deterministic denials:
  `HOOK_SOURCE_IMMUTABLE`, `HOOK_SECRET_INPUT_FORBIDDEN`,
  `HOOK_VALIDATION_FAILED`, `HOOK_APPROVAL_REQUIRED`.

#### Imported Foundation Types

This TechSpec reuses existing or foundation-owned types rather than inventing
parallel model vocabularies:

- `ToolID`, `ToolsetID`, `EffectiveToolDecision`, `ProjectionQuery`, and
  `ToolView` remain registry-foundation types from the `tools-registry`
  workstream.
- `task.ActorContext`, `task.Run`, `task.ClaimResult`, `task.RunStatus`,
  `task.RunResult`, and `task.RunFailure` remain task-domain authorities.
- `mcpauth.Status` remains the redacted MCP auth authority model.
- `contract.TaskRunPayload`, `contract.TaskSummaryPayload`, and related task
  read models remain the transport response base where no new shape is needed.

#### Additional Typed Models

`HarnessPromptSection` gains:

- `tools`

```go
type AutonomyClaimRequest struct {
	WorkspaceID          string   `json:"workspace_id,omitempty"`
	RequiredCapabilities []string `json:"required_capabilities,omitempty"`
	PriorityMin          int      `json:"priority_min,omitempty"`
	LeaseSeconds         int64    `json:"lease_seconds,omitempty"`
	Wait                 bool     `json:"wait,omitempty"`
	IdempotencyKey       string   `json:"idempotency_key,omitempty"`
}
```

```go
type AutonomyHeartbeatRequest struct {
	RunID        string `json:"run_id"`
	LeaseSeconds int64  `json:"lease_seconds,omitempty"`
}
```

```go
type AutonomyCompletionRequest struct {
	RunID  string          `json:"run_id"`
	Result json.RawMessage `json:"result,omitempty"`
}
```

```go
type AutonomyFailureRequest struct {
	RunID    string          `json:"run_id"`
	Error    string          `json:"error"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}
```

```go
type AutonomyReleaseRequest struct {
	RunID  string `json:"run_id"`
	Reason string `json:"reason,omitempty"`
}
```

```go
type MCPAuthStatusRequest struct {
	ServerName string `json:"server_name"`
}
```

```go
type MCPAuthStatus struct {
	ServerName   string     `json:"server_name"`
	Status       string     `json:"status"`
	AuthType     string     `json:"auth_type,omitempty"`
	ClientID     string     `json:"client_id,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Refreshable  bool       `json:"refreshable"`
	TokenPresent bool       `json:"token_present"`
	Diagnostic   string     `json:"diagnostic,omitempty"`
}
```

The autonomy request models are intentionally free of raw `claim_token`. The
daemon resolves the session's active lease token internally before calling
`HeartbeatRunLease`, `CompleteRunLease`, `FailRunLease`, or `ReleaseRunLease`.

No new SQLite tables are required by this follow-up. Existing registry/toolset
data, session lineage atoms, task-run lease state, and MCP auth token storage
remain authoritative.

### Data-Model Field Rationale

Key field rationale:

- `HarnessPromptSectionTools` - enum/string marker that enables one additional
  startup prompt section without introducing free-form prompt toggles.
- `Autonomy*Request.run_id` - string identifier that lets AGH target one leased
  run while keeping the raw ownership token server-side.
- `MCPAuthStatus.*` - typed redacted diagnostic fields reused from the existing
  `tools.MCPAuthStatus` model so auth failures stay machine-readable without
  leaking token material.
- runtime default discovery toolsets - derived `[]ToolsetID` overlay because the
  default is policy, not durable config ownership.

| Field | Shape | Purpose | Storage decision |
|---|---|---|---|
| `HarnessPromptSectionTools` | enum/string | Adds explicit startup guidance for tool usage | Typed enum beside existing prompt sections |
| Runtime default discovery toolsets | `[]ToolsetID` derived at runtime | Makes discovery available even when agent definitions omit it | Computed in policy resolver; not persisted |
| `ToolView.reason_codes` | `[]string` | Deterministic explanation for projection and call decisions | Typed list, not prose-only diagnostics |
| `Autonomy*Request.run_id` | string | Identifies the leased run without exposing secret token material | Typed input field |
| Session-bound active lease lookup | derived from `task_runs` + session identity | Preserves authoritative claim-token fencing behind the daemon | Derived runtime state; no new table |
| `tools.Scope.Operator` | bool | Distinguishes operator projection from session-scoped projection | Existing registry scope field; hosted MCP remains a transport-level session projection |
| `MCPAuthStatus.*` | typed redacted status object | Lets tools and operators diagnose auth failures without token leakage | Existing `internal/tools` model derived from `internal/mcp/auth`; never persisted in registry metadata |
| `catalogUsageInstructions` for skills | rendered text | Teaches tool-first skill loading path | Prompt text, not config |

### Side-Table vs JSON Decisions

| Domain state | Decision | Rationale |
|---|---|---|
| Default discovery toolsets | Runtime-computed overlay, not persisted | They are policy defaults, not durable ownership state |
| Tools prompt guidance | Bundled skill asset + prompt section | Prompt guidance is content, not queryable runtime state |
| Session-bound autonomy lease lookup | Reuse existing `task_runs` lease state | Do not create a parallel queue or parallel lease map |
| MCP auth diagnostics | Derived redacted status object | Queryable token state already belongs to `internal/mcp/auth` and global DB |
| Tool coverage map | Typed built-in descriptors/toolsets | The tool surface must stay queryable and policy-aware |

### API Endpoints

This follow-up reuses the existing registry transports rather than adding a new
parallel API family.

Existing registry surfaces stay canonical:

- `GET /api/tools`
- `POST /api/tools/search`
- `GET /api/tools/{id}`
- `POST /api/tools/{id}/invoke`
- `GET /api/sessions/{id}/tools`
- `POST /api/sessions/{id}/tools/search`
- `GET /api/toolsets`
- `GET /api/toolsets/{id}`

Behavior changes required by this TechSpec:

1. `GET /api/sessions/{id}/tools` and hosted MCP projections must include the
   default discovery toolsets when the effective policy does not deny them.
2. `GET /api/tools` and `GET /api/tools/{id}` must surface the expanded
   built-in coverage map with deterministic availability/policy reasons.
3. `POST /api/tools/{id}/invoke` for autonomy tools must use the session-bound
   lease bridge and must never emit or accept raw `claim_token`.
4. `POST /api/tools/{id}/invoke` for `agh__mcp_auth_status` returns only
   redacted status. It must never trigger login/logout flows.

CLI parity:

- `agh tool list/search/info/invoke`
- `agh toolsets list/info`
- `agh mcp auth status`

CLI hard cut required for autonomy:

- `agh task next|heartbeat|complete|fail|release` must converge on the same
  session-bound lease contract as the new autonomy tools instead of requiring
  raw `claim_token` arguments or printing raw `claim_token` values.
- `agh task next` no longer returns a write credential. It returns the claimed
  run and lease summary only.
- `agh task heartbeat|complete|fail|release` accept `run_id` plus request body
  fields only; they do not accept `--claim-token`.

## Integration Points

### Hosted MCP

Hosted MCP remains the model-visible exposure transport already shipped by the
`tools-registry` foundation on this branch. This follow-up changes the content
of the exposed projection, not the transport/service model:

- default discovery tools are now visible by default;
- the new autonomy tool family may be exposed when the session projection
  allows it;
- raw `claim_token` may not traverse hosted MCP responses or requests;
- approval bridge semantics remain those defined by the foundation spec.

Hosted MCP authentication and session binding are inherited, not redesigned:

- the daemon mints a short-lived hosted MCP launch record per session/load;
- the launch record is keyed by `session_id`, `workspace_id`, a single-use
  `hosted_mcp_bind_nonce`, expiry, and expected AGH binary path;
- `hosted_mcp_bind_nonce` is a correlation nonce, not a bearer secret and not
  claim-token-equivalent;
- `agh tool mcp --session <id> --bind-nonce <nonce>` may bind only when the
  daemon validates all of the following at once:
  - the nonce matches a live launch record;
  - the record has not expired;
  - the Unix-domain socket peer credentials identify the same OS user;
  - the peer executable matches the expected AGH binary path;
- when peer credentials or executable validation are unavailable, hosted MCP
  fails closed and the session receives no hosted registry projection;
- after a successful bind, the daemon binds that UDS connection to exactly one
  session/workspace projection and rejects any later client-supplied
  `session_id` or `workspace_id`;
- the launch record is invalidated on first successful bind, session end, proxy
  disconnect, or TTL expiry, whichever happens first;
- a foreign local process that runs `agh tool mcp --session <id>` without a
  valid nonce plus matching UDS peer credentials must receive a deterministic
  permission failure and no tool projection.

Hosted MCP approval bridge is also inherited explicitly:

- hosted MCP `tools/list` includes only tools callable without a new approval
  prompt or tools whose session has a live daemon-mediated approval channel;
- when `EffectiveToolDecision.approval_required=true` and ACP
  `session/request_permission` is available, `Registry.Call` derives a context
  with `[tools.policy].approval_timeout_seconds`, issues the permission
  request, and blocks the MCP `tools/call` response until approved, denied,
  timed out, canceled, or the hosted MCP stdio/UDS connection closes;
- when no approval channel is available, hosted MCP hides the tool from
  `tools/list` when projection knows that upfront; if a call still reaches
  dispatch, it returns `ErrToolApprovalRequired` with reason codes
  `approval_required` and `approval_unreachable`;
- approval timeout returns `approval_required` plus `approval_timed_out`;
- hosted MCP proxy disconnect or stdio close cancels the derived context and
  returns `approval_required` plus `approval_canceled`;
- hosted MCP cannot satisfy approval using client-supplied arguments or local
  `approval_token`; CLI/HTTP/UDS may use `approval_token`, hosted MCP must use
  the daemon approval bridge.

### Task Service

`internal/task` remains the single authority for claim and lease transitions.
The autonomy built-in family is only a registry-managed surface over these
writers:

- `ClaimNextRun`
- `HeartbeatRunLease`
- `CompleteRunLease`
- `FailRunLease`
- `ReleaseRunLease`

The bridge must derive actor identity from the calling session and must resolve
active lease state internally before invoking the writer.

### Bootstrap Task Tools

The `agh__tasks` and `agh__autonomy` split is contractual in this follow-up.
The final AGH-owned task tool surface is:

```text
agh__tasks:
  agh__task_list
  agh__task_read
  agh__task_create
  agh__task_child_create
  agh__task_update
  agh__task_cancel
  agh__task_run_list

agh__autonomy:
  agh__task_run_claim_next
  agh__task_run_heartbeat
  agh__task_run_complete
  agh__task_run_fail
  agh__task_run_release
```

Required writer mapping:

| Tool ID | Writer / handler authority | Notes |
|---|---|---|
| `agh__task_list` | existing task read/list service and API handlers | Metadata/task inspection only |
| `agh__task_read` | existing task detail read service and API handlers | Metadata/task inspection only |
| `agh__task_create` | existing task create writer | Regular task authoring |
| `agh__task_child_create` | existing child-task create writer | Child creation only |
| `agh__task_update` | existing task update writer | Queue/task metadata mutation, not run lease mutation |
| `agh__task_cancel` | existing task cancel writer | Task cancellation only |
| `agh__task_run_list` | existing task-run list/read path | Run inspection only |
| `agh__task_run_claim_next` | `task.Service.ClaimNextRun` | Identity-bound next-work claim |
| `agh__task_run_heartbeat` | `task.Service.HeartbeatRunLease` | Session-bound lease extension |
| `agh__task_run_complete` | `task.Service.CompleteRunLease` | Session-bound terminal completion |
| `agh__task_run_fail` | `task.Service.FailRunLease` | Session-bound terminal failure |
| `agh__task_run_release` | `task.Service.ReleaseRunLease` | Session-bound requeue/release |

Hard exclusions in this spec:

- no generic `agh__task_claim`, `agh__task_next`, `agh__task_run_start`, or
  `agh__task_complete`/`fail`/`release` aliases outside `agh__autonomy`;
- no `agh__tasks` tool may mutate lease ownership, transition a claimed run, or
  bypass the session-bound autonomy bridge;
- no AGH-owned tool may call the unfenced operator-style run writers once the
  hard cut lands;
- spawn, cross-session terminal-state mutation, and scheduler-owned claim paths
  remain outside both `agh__tasks` and `agh__autonomy`.

Implementation hard cut:

- the existing agent-task HTTP/UDS semantics in `internal/api/core/agent_tasks.go`
  remain the behavioral source of truth for claim/heartbeat/complete/fail/release;
- this follow-up removes raw-`claim_token` exposure from those surfaces and
  converges the same semantics into registry-managed autonomy tools;
- the old raw-token DTOs in `internal/api/contract/agents.go` are delete
  targets, not compatibility surfaces.

### Session-Bound Autonomy Lookup

Autonomy write calls are valid only when the daemon can prove that the run is
the active lease for the calling session before it resolves the internal raw
token.

Required lookup primitive:

```go
type TaskLeaseAuthority interface {
	LookupActiveRunForSession(
		ctx context.Context,
		sessionID string,
		runID string,
	) (AutonomyLeaseHandle, error)
}
```

Required invariant order for `run_heartbeat`, `run_complete`, `run_fail`, and
`run_release`:

1. Resolve the caller session from `tools.Scope.SessionID`.
2. Reject empty session scope with `AUTONOMY_SESSION_REQUIRED`.
3. Call `LookupActiveRunForSession(ctx, scope.SessionID, req.RunID)`.
4. Reject no active lease with `AUTONOMY_NO_ACTIVE_LEASE`.
5. Reject active lease on a different `run_id` with `AUTONOMY_FOREIGN_RUN`.
6. Reject non-active or expired leases with `AUTONOMY_LEASE_EXPIRED`.
7. Verify `handle.SessionID == scope.SessionID` and the run status is one of
   `claimed|starting|running` before reading `handle.ClaimToken`.
8. Only then call the existing writer with the internal raw token.

The lookup is an authorization bridge, not the final authority. The existing
task writers remain authoritative because they revalidate token hash, run
state, and lease expiry inside their `BEGIN IMMEDIATE` transaction. If the
lease expires, changes owner, or otherwise becomes stale after lookup but
before the writer applies the mutation, the writer rejection maps back to the
same deterministic autonomy reason such as `AUTONOMY_LEASE_EXPIRED`, and no
raw token is echoed to the caller.

Required invariant for `run_claim_next`:

1. If the calling session already owns an active lease, reject with
   `AUTONOMY_LEASE_ALREADY_HELD`.
2. Otherwise call `task.Service.ClaimNextRun` with the current agent/session
   identity and return the claimed run without exposing the raw token.

The underlying persistence path may continue to use the existing
`idx_task_runs_session_status` index. This TechSpec does not require a new
schema migration unless benchmarks later show the session-bound lookup to be
insufficient under real concurrency.

### Existing MCP Config And Auth

This follow-up uses the existing MCP auth subsystem and the already-shipped
redacted `tools.MCPAuthStatus` model in two ways:

- operator-management login/logout keep using the current management surfaces;
- `agh__mcp_auth_status` reads the same redacted status model already exposed by
  `internal/mcp/auth` and settings.

No token storage, OAuth callback handling, or browser flow is moved into the
normal tool surface.

## Extensibility Integration Plan

### Extension Manifests And Runtime Extension Points

No new extension manifest shape is introduced by this follow-up. Extensions keep
using the registry foundation's descriptor and execution model. The follow-up
affects them indirectly by clarifying discovery defaults, prompt guidance, and
projection behavior.

### Hooks

No new hook family is added. Existing tool hooks continue to fire at registry
dispatch. Hook denial/narrowing must be visible in operator projections and must
be rechecked during dispatch.

### Skills, Tools, Resources, Bundles

This spec adds two AGH-owned assets:

- bundled skill `agh-tools-guide`;
- startup prompt section `tools`.

It also changes the existing shipped guidance in `internal/skills/catalog.go`
and `agh-agent-setup` so the tool-first path is explicit. When `agh__skill_view`
is available, the catalog guidance must point to it first; CLI remains an
operator fallback, not the primary agent path. When runtime policy denies
`agh__skill_view`, the rendered skills text may mention the CLI fallback
conditionally. It must not advertise the CLI path when the tool is present and
callable.

### Bridge SDKs And MCP Sidecars

No new bridge SDK or sidecar contract is introduced. Hosted MCP continues to
use the existing registry projection and call path.

### Docs Or Examples Extension Authors Need

Update docs to clarify:

- default discovery toolsets for agents;
- the difference between runtime tools and operator-only commands;
- the new `agh-tools-guide` and prompt section;
- the fact that MCP auth login/logout remain management flows;
- the autonomy tool family and the absence of raw `claim_token` on AGH-owned
  surfaces.

## Agent Manageability Plan

Agents must be able to discover and operate AGH runtime capabilities without
relying on the web UI and without guessing whether to use CLI or built-in tools.

Discovery:

- every agent gets `agh__bootstrap` + `agh__catalog` by default;
- startup prompt includes a dedicated `tools` section;
- bundled `agh-tools-guide` teaches `search -> info -> invoke`;
- `skills` catalog text points to `agh__skill_view` as the normal agent path
  when available.

Tool-first convention:

- AGH-internal runtime operations should use dedicated tools;
- shell/Bash may still exist because ACP drivers decide that surface;
- AGH documentation and prompt guidance must not present shelling out to
  `agh ...` as the preferred path when a dedicated tool exists.

Operator-management split:

- operators retain CLI/HTTP/UDS for lifecycle, destructive repair, and OAuth
  login/logout;
- agents may inspect and mutate runtime domains through the tool surface when
  the operation does not cross trust-root, raw-secret, or human-interactive
  boundaries;
- agents do not get agent-callable spawn of arbitrary sessions or browser auth
  tools in this TechSpec.

Deterministic errors:

- every projection and call denial reports `reason_codes`;
- operator projections identify the denying layer;
- `agh__mcp_auth_status` returns auth-specific diagnostics without token
  material;
- autonomy tools fail with deterministic errors for missing active lease,
  wrong `run_id`, expired lease, stale ownership, or permission mismatch.
- `agh__network_send` rejects raw token payloads or metadata with
  `network_raw_token_rejected`.

E2E parity checks:

- `agh tool list/search/info` matches HTTP and UDS for the same operator scope;
- hosted MCP `tools/list` equals the effective session projection exactly;
- default discovery tools appear for agents with empty tool declarations and
  disappear when denied by policy;
- autonomy tool calls and CLI task-run commands exercise the same task writers;
- no AGH-owned surface emits raw `claim_token`.

## Config Lifecycle

### Global `config.toml`

This follow-up adds **no new top-level config keys** beyond the
`tools-registry` foundation. It changes the meaning of existing tool policy only
in these ways:

- when an agent omits `tools` and `toolsets`, effective policy injects
  `agh__bootstrap` and `agh__catalog` as runtime defaults;
- explicit `deny_tools`, session lineage narrowing, source policy, approval
  ceiling, and hooks may still remove those defaults;
- no config key is added to disable only the default overlay independently of
  the general tool policy.

### Agent Definitions

Existing agent fields remain authoritative:

- `tools`
- `toolsets`
- `deny_tools`

Behavior changes:

- empty `tools`/`toolsets` no longer means "no discovery surface";
- the runtime adds discovery defaults at resolution time only;
- those defaults are not rewritten back into agent files or persisted lineage
  rows.

### Existing MCP Config And Auth Lifecycle

No new MCP auth config keys are added. Existing server definitions and token
storage remain authoritative. The only new agent-facing behavior is that
redacted auth status becomes available as a built-in tool.

Stdio MCP server declarations must fail validation when `env` contains
dangerous interpreter or dynamic-loader keys that can alter subprocess startup
semantics, including `NODE_OPTIONS`, `PYTHONPATH`, `PYTHONHOME`, `LD_PRELOAD`,
and `DYLD_*`. Remote MCP transports do not execute local subprocesses and are
not rejected by this stdio-specific env filter.

### Old vs New Effective Behavior

| Existing input | Old behavior | New behavior |
|---|---|---|
| Agent omits `tools` and `toolsets` | Discovery surface may be absent | Runtime injects `agh__bootstrap` + `agh__catalog` unless effective policy denies them |
| Agent explicitly denies discovery tools | Deny wins where honored | Deny still wins after default overlay |
| MCP auth missing for one source | Operators see status through CLI/settings only | Agents can inspect redacted auth status through `agh__mcp_auth_status`, but repair stays on management surfaces |
| Task-run lease mutation | AGH-owned CLI/API contracts use raw `claim_token` | AGH-owned tool/CLI/API contracts are session-bound and token-internal |
| Automation / extension / hook management | Often CLI-only or config-file-mediated | Tool-callable by default, with source policy and approval guarding mutation |
| Config mutation | Narrow allow-list only | Validated overlay mutation is broadly tool-callable except trust-root and secret paths |

### Docs And Generated Surfaces

Update:

- bundled skill assets and skill catalog wording;
- startup prompt docs and examples;
- CLI docs for task-run commands and runtime API docs for the session-bound
  autonomy contract;
- CLI docs and runtime docs for `agh tool`, `agh toolsets`, `agh mcp auth`,
  automation management, extension lifecycle, hook management, and config
  mutation;
- site docs explaining the operator/tool split and default discovery behavior;
- generated contracts because autonomy and management tool DTOs change.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|---|---|---|---|
| `internal/daemon` | modified | Startup prompt sections, session projections, and service wiring change | Add `tools` section wiring and policy inputs |
| `internal/tools` | modified | Becomes the authority for default discovery overlay and final built-in coverage map | Extend built-ins, policy resolver, and projections |
| `internal/skills` | modified | Bundled guide and catalog text change | Add `agh-tools-guide`; update usage instructions |
| `internal/task` | modified/consumed | Autonomy tool family must call existing lease writers without exposing tokens | Add session-bound bridge over existing writers |
| `internal/automation` | modified/consumed | Automation CRUD becomes tool-callable through shared writers | Reuse existing validators and lifecycle methods |
| `internal/extension` | modified/consumed | Extension lifecycle becomes tool-callable through shared registry/marketplace services | Reuse install/update/remove/enable/disable flows |
| `internal/config` | modified/consumed | Config overlay mutation becomes broadly tool-callable except trust-root paths | Keep validated writer as the only mutation path |
| `internal/hooks` | modified/consumed | Hook declaration management becomes tool-callable for mutable sources | Reuse normalization/validation and add ownership rules |
| `internal/cli` | modified | Task-run commands must converge on the new safe lease contract | Remove raw-token contract from AGH-owned surfaces |
| `internal/api/*` | modified/consumed | Existing invoke/projection surfaces enforce new semantics | Keep transport parity and deterministic errors |
| `internal/mcp/auth` | consumed | Status is surfaced through a built-in tool; login/logout remain external | Reuse redacted status only |
| `packages/site` | modified | Tool guidance, manageability, CLI docs, and MCP auth docs must align | Update docs and regenerate CLI reference |
| `web/` | no impact unless contract drift | No new mandatory runtime UI | Regenerate types only if DTOs change |

## Test Strategy

### Unit Tests

- prompt section resolver includes `tools` only when policy allows it;
- skills catalog text points to `agh__skill_view` when the tool exists;
- default discovery overlay applies only when the agent omitted discovery tools;
- explicit denies override defaults;
- operator and session projections diverge correctly for denied/unavailable
  tools;
- `agh__mcp_auth_status` redacts all secret material;
- autonomy bridge refuses to emit or accept raw `claim_token`;
- `run_heartbeat|complete|fail|release` reject a foreign `run_id` with
  `AUTONOMY_FOREIGN_RUN`;
- `run_claim_next` rejects when the session already holds an active lease with
  `AUTONOMY_LEASE_ALREADY_HELD`;
- `agh__network_send` rejects raw token payloads and metadata with
  `network_raw_token_rejected`;
- `agh__config_set|unset` accept validated non-root paths and reject
  trust-root/secret paths with deterministic denials;
- `agh__automation_*` create/update/delete validators match the existing
  automation domain rules;
- `agh__extensions_install|update|remove` respect source-trust policy;
- `agh__hooks_create|update` reject immutable-source and secret-bearing inputs
  correctly;
- event, observe, and task payload renderers never expose raw `claim_token`.

### Integration Tests

- HTTP/UDS/CLI parity for list/search/info/invoke over the expanded built-in
  surface;
- hosted MCP `tools/list` equals `GET /api/sessions/{id}/tools`;
- autonomy tools and CLI commands route to the same task writers and preserve
  lease invariants;
- concurrent heartbeats from different sessions for the same run yield one
  success path and one deterministic mismatch error;
- cross-session complete/fail/release attempts on the same `run_id` return
  `AUTONOMY_FOREIGN_RUN` or `AUTONOMY_NO_ACTIVE_LEASE`;
- automation job/trigger create/update/delete through tools reach the same
  validated writers as CLI/host API;
- extension install/update/remove/enable/disable through tools reach the same
  registry and marketplace services as the CLI;
- hook create/update/delete/enable/disable through tools preserve normalization
  and source-ownership rules;
- config show/get/set/unset/diff parity holds across tool, CLI, and HTTP/UDS
  management surfaces for the same caller scope;
- MCP auth status tool reflects the same redacted status as settings/CLI;
- prompt assembly includes the new tools section and bundled guide content.

### E2E Tests

- start a real session with empty `tools` declarations and verify discovery
  defaults are visible;
- deny `agh__catalog` or a discovery pattern and verify it disappears from the
  session projection while remaining debuggable to operators;
- claim a run through the autonomy tool family, heartbeat it, complete/fail it,
  and verify no surface leaks raw `claim_token`;
- run two isolated sessions in the same workspace, claim work in one, and prove
  the other cannot mutate that lease through tool, CLI, or hosted MCP paths;
- install or update an extension through the tool surface and verify policy,
  approval, and restart messaging stay aligned with the operator surface;
- create/update/delete automation jobs and hooks through the tool surface and
  verify the runtime behaves from real persisted definitions rather than mocks;
- exercise an expired or missing MCP login and verify the agent sees status
  diagnostics while repair remains on management surfaces;
- capture redaction snapshots from events, logs, task history, and observe
  surfaces and verify no `agh_claim_*` pattern survives.

## Implementation Steps

### Build Order

Steps 1-10 land in one PR. There is no supported merge point where autonomy
tool contracts, CLI contracts, HTTP/UDS contracts, hosted MCP projections,
OpenAPI, docs, and tests describe different write credentials.

1. Extend the shipped prompt/catalog assets with ADR-backed analysis artifacts,
   bundled `agh-tools-guide`, and the new `HarnessPromptSectionTools`
   enum/descriptor wiring - no dependencies.
2. Extend the shipped registry policy path with runtime default discovery
   overlay and scope-aware policy-input resolution for `list/search/get/call`
   - depends on step 1.
3. Expand the shipped built-in catalog and toolset membership from the current
   bootstrap/catalog/coordination/tasks subset across memory, sessions,
   workspace, config, tasks, autonomy, coordination, hooks, automation,
   extensions, MCP auth status, observe, and bridges - depends on step 2.
4. Update prompt/catalog guidance so AGH-internal discovery paths teach
   `search -> info -> invoke` and `agh__skill_view` first - depends on steps 1,
   2, and 3.
5. Implement the session-bound autonomy lease bridge over existing task writers
   with no raw token exposure - depends on steps 2 and 3.
6. Hard-cut AGH-owned autonomy CLI/API contracts away from raw `claim_token`
   inputs/outputs and align them with the new bridge - depends on step 5.
7. Add `agh__mcp_auth_status` as a built-in wrapper over the existing redacted
   auth service and keep login/logout on management surfaces - depends on steps
   2 and 3.
8. Align the already-shipped hosted MCP transport, operator diagnostics, and
   transport parity around the new defaults and expanded surface - depends on
   steps 2, 3, 4, 5, and 7.
9. Update docs, generated references, and examples - depends on steps 4, 6,
   and 8.
10. Run focused unit/integration/e2e verification for prompt guidance, policy,
   autonomy, and MCP auth status - depends on steps 1-9.

### Technical Dependencies

- Existing `tools-registry` foundation contracts and hosted MCP transport.
- Current task lease writers and identity helpers in `internal/task`.
- Current MCP auth status models and token store.
- Current session lineage resolution path in `internal/session`.
- Existing CLI docs generation pipeline in `packages/site`.

### Safety Invariants

1. Every AGH-owned tool invocation path still enters `Registry.Call`.
2. Discovery defaults improve UX only; they never bypass effective policy.
3. Operator projections may show unavailable or denied tools; session and hosted
   MCP projections may show only callable tools.
4. ACP permission mode remains the approval ceiling.
5. Spawn, cross-session terminal-state mutation, and daemon lifecycle stay
   operator-only.
6. Raw `claim_token` never appears in AGH-owned tool, CLI, HTTP, UDS, SSE, log,
   settings, or memory surfaces.
7. The autonomy tool family reuses the existing task lease writers; it does not
   create a second ownership path.
8. One active lease per session remains required for this design. If that
   invariant changes later, AGH must introduce a separate typed lease handle in
   a follow-up spec.
9. `agh__mcp_auth_status` is redacted status only; it must not trigger browser
   auth, token refresh side effects by default, or logout.
10. Prompt/catalog guidance must not instruct CLI-first use for AGH internals
    when a dedicated tool exists.
11. `agh__network_send` and every AGH-owned message surface reject raw
    `claim_token` fields in body or metadata with deterministic
    `network_raw_token_rejected` reason codes. The raw-token guard is a
    recursive JSON object-key check for keys equal to `claim_token`
    case-insensitively. It allows `claim_token_hash` and benign string values
    that merely mention `agh_claim_`.
12. Hosted MCP `tools/list` must equal the effective session projection exactly,
    not a superset or stale cache.
13. Mutable management tools must reuse the same domain writers and validators
    as CLI/HTTP/UDS management surfaces; tool-only mutation logic is forbidden.
14. Mutability is denied only when the operation crosses trust-root, raw-secret,
    or human-interactive boundaries, not merely because it is a write.
15. Hosted MCP bind requires a daemon-minted, single-use, session-bound bind
    nonce plus UDS peer-credential and AGH binary validation; foreign local
    processes fail closed.
16. Hosted MCP approval-required calls cannot succeed from client-supplied
    arguments alone; they must complete through ACP `session/request_permission`
    or fail closed with `approval_unreachable`, `approval_timed_out`, or
    `approval_canceled`.
17. `agh__tasks` is never allowed to grow lease-transition verbs; all
    claim/heartbeat/complete/fail/release semantics live only in
    `agh__autonomy`.
18. Stdio MCP declarations reject unsafe interpreter and dynamic-loader env
    keys during config validation instead of silently stripping them or passing
    them to child processes.

## Monitoring and Observability

- metric: projection recompute count and latency by surface;
- metric: default-discovery overlay applied/denied count;
- metric: autonomy tool invocation count by method and outcome;
- metric: MCP auth status tool calls by status value;
- log fields: `tool_id`, `surface`, `session_id`, `workspace_id`, `reason_codes`,
  `approval_mode`, `claim_token_hash`, `run_id`, `lease_until`, `actor_kind`,
  `actor_id`, `release_reason`, `mcp_server_name`;
- redaction tests and structured log assertions for `claim_token`, OAuth
  material, and other secret fields;
- observe/event surfaces remain already-redacted projections and must not add a
  second serialization path that could reintroduce secret fields;
- operator diagnostics should make it obvious whether a missing tool came from
  policy, source health, auth, hook denial, or operator-only classification.

## Technical Considerations

### Key Decisions

- Decision: AGH is tool-first by convention, not by shell blocking.
  - Rationale: Bash behavior belongs to ACP drivers; AGH controls its own
    structured surfaces and guidance.
  - Trade-off: shell escape still exists when the driver exposes it.
  - Alternative rejected: sandbox-level `agh` blocking.

- Decision: `agh__bootstrap` + `agh__catalog` are default discovery toolsets.
  - Rationale: agents need a minimal self-describing surface even when authors
    forget to declare it.
  - Trade-off: default surface is slightly broader.
  - Alternative rejected: keep discovery entirely opt-in.

- Decision: effective policy is recomputed for `list/search/get/call`.
  - Rationale: discovery and execution must reflect current runtime state.
  - Trade-off: projection paths need caching discipline.
  - Alternative rejected: call-time-only or boot-time-static policy.

- Decision: identity-bound task execution becomes a dedicated tool family.
  - Rationale: manual and autonomous paths should converge on the same
    primitives without forcing shell-outs for AGH internals.
  - Trade-off: AGH must implement a safe session-bound lease bridge.
  - Alternative rejected: keep the lease path CLI-only.

- Decision: MCP auth is status-only on the agent tool surface.
  - Rationale: agents need diagnostics, but OAuth login/logout remains an
    interactive management workflow.
  - Trade-off: agents cannot fully self-repair missing OAuth login through
    normal tool calls alone.
  - Alternative rejected: full login/logout tool family.

### Known Risks

- Risk: agents may still shell out to `agh ...` when a shell tool exists.
  - Mitigation: explicit prompt guidance, default discovery, and broad built-in
    coverage make the structured path easier and more visible.

- Risk: per-call policy recomputation adds latency.
  - Mitigation: cache normalized inputs and projection results behind explicit
    invalidation keys, but never let caches become authority. The cache key must
    include surface, session ID, workspace ID, agent name, lineage tuple
    (session/parent/root), agent-definition hash, tool-catalog revision,
    availability revision, hook revision, approval ceiling, and query
    fingerprint. Invalidate on agent reload, lineage change, toolset or
    descriptor reload, extension/skill registry change, hook reload, MCP auth
    health change that affects availability, and config overlay changes that
    affect tool policy.

- Risk: autonomy bridge design depends on one active lease per session.
  - Mitigation: encode the invariant explicitly and require a follow-up ADR
    before relaxing it.

- Risk: a broader mutable tool surface increases the blast radius of agent
  mistakes.
  - Mitigation: policy and approval become the containment layer; operator-only
    is reserved for trust-root and secret boundaries rather than used as a
    blanket substitute for policy.

- Risk: docs drift can reintroduce CLI-first examples or stale opt-in guidance.
  - Mitigation: site updates, CLI docs regeneration, and prompt-assembly tests
    ship in the same implementation sequence.

## Post-Implementation Residual Checks

These checks replace the original delete-target list because the raw-token hard
cut has already landed on this branch. The implementation must verify that no
residual compatibility paths remain and that the retained guard helpers continue
to enforce the safety invariants.

- `internal/cli/task.go`
  - verify `newTaskHeartbeatCommand`, `newTaskCompleteCommand`,
    `newTaskFailCommand`, and `newTaskReleaseCommand` expose no
    `--claim-token` flag;
  - verify `requiredAgentTaskRunToken(...)` is absent;
  - verify examples never instruct users to pass `$CLAIM_TOKEN`.
- `internal/api/contract/agents.go`
  - verify AGH-owned autonomy DTOs expose only `run_id`-keyed request shapes
    and never public raw-token request or response fields;
  - retain `ErrRawClaimTokenMetadata`, `ContainsRawClaimTokenField`, and
    `ValidateNoRawClaimTokenField` as safety guards for metadata/message
    ingress.
- `internal/api/core/agent_tasks.go`
  - verify request parsing never forwards a public `req.ClaimToken` into task
    lease writers;
  - verify all AGH-owned autonomy writes route through the session-bound lookup
    bridge and then through the existing task writers.
- `internal/api/udsapi/agent_tasks_test.go`,
  `internal/api/udsapi/agent_tasks_bindings_test.go`,
  `internal/api/contract/agents_test.go`, and `internal/api/spec/spec_test.go`
  - verify assertions cover session-bound and redaction behavior rather than
    raw `claim_token` request or response bodies.
- `packages/site/content/runtime/cli-reference/task/heartbeat.mdx`
- `packages/site/content/runtime/cli-reference/task/complete.mdx`
- `packages/site/content/runtime/cli-reference/task/fail.mdx`
- `packages/site/content/runtime/cli-reference/task/release.mdx`
- `packages/site/content/runtime/core/autonomy/task-runs-and-leases.mdx`
  - verify examples and prose do not instruct users to copy raw claim tokens
    across AGH-owned CLI/API mutations.
- `packages/site/content/runtime/core/configuration/agent-md.mdx`
  - verify prose does not frame discovery as entirely opt-in or treat CLI-first
    skill loading as the canonical agent path;
  - keep `tools`, `toolsets`, and `deny_tools` as supported policy grammar.
- `packages/site/content/runtime/core/configuration/config-toml.mdx`
  - verify prose does not imply agents have no structured MCP auth status
    visibility or no runtime discovery defaults.
- `packages/site/content/runtime/core/agents/definitions.mdx`
  - verify examples do not treat CLI shell-out as the preferred AGH internal
    path when a dedicated tool exists.
- `internal/skills/catalog.go`
  - verify the unconditional `agh skill view` usage string is not the only
    catalog guidance;
  - verify tool-first conditional guidance is keyed by callable
    `agh__skill_view`.
- `internal/skills/bundled/skills/agh-agent-setup/SKILL.md`
  - verify examples do not imply `agh__catalog` is only an opt-in discovery
    path when the default runtime overlay makes it callable.
- `internal/daemon/prompt_sections.go`
  - verify the startup assembly includes `tools` and is no longer the default
    four-section-only composition.
- `.compozy/tasks/tools-registry/_techspec.md` and
  `.compozy/tasks/tools-registry/adrs/adr-004-mvp-native-tool-scope.md`
  - verify text has been superseded where it permanently excludes
    claim/heartbeat/complete/fail/release from the agent-callable surface.
- docs and examples that frame automation, extension lifecycle, or mutable hook
  management as inherently CLI-only or config-file-only when a dedicated tool
  exists in the new canonical surface.
- startup prompt and skills catalog wording
  - verify text does not teach only `agh skill view` or frame `agh__catalog` as
    opt-in-only when the tool is callable by default.

## Architecture Decision Records

- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md) — AGH-internal runtime capabilities become tool-first by convention and every agent gets discovery defaults.
- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md) — discovery is a UX filter; runtime dispatch remains authoritative.
- [ADR-003: Identity-Bound Task Execution Uses Dedicated Agent Tools](adrs/adr-003-identity-bound-autonomy-tools.md) — task lease execution becomes a first-class tool family while spawn/lifecycle stays operator-only.
- [ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces](adrs/adr-004-mcp-auth-status-tool.md) — agent tools get diagnostics, not browser OAuth flows.
- [ADR-005: Autonomy Tool Surfaces Are Session-Bound And Never Expose Raw Claim Tokens](adrs/adr-005-session-bound-autonomy-surface.md) — lease tokens stay internal even when autonomy execution becomes tool-callable.
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md) — mutation is contained by policy and approval, while operator-only remains reserved for trust-root and secret boundaries.
