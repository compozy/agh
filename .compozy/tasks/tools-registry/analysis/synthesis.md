# Tool Registry Synthesis and Proposed Direction

## Executive Summary

AGH should build a Tool Registry as a daemon-owned runtime service that composes tool metadata, availability, policy, execution, hooks, telemetry, and extension adapters.

The foundation should not be "add many built-in tools." The foundation should be:

1. a first-class runtime tool contract,
2. a registry that aggregates built-ins, MCP, extensions, and dynamic providers,
3. a policy/availability projection for each agent/session,
4. a single dispatch pipeline,
5. an extension-safe execution boundary,
6. agent-manageable CLI/HTTP/UDS/session surfaces,
7. toolsets/bundles comparable to skills.

OpenFang was not analyzed because `.resources/openfang` is not present in this checkout.

## Recommended Architecture

### 1. Split descriptor, runtime handle, and resource record

Keep the existing `internal/tools.Tool` resource shape as the cold catalog/desired-state record, but introduce a runtime contract with separate types:

- `ToolID`: stable namespaced id such as `agh.skill.view`, `mcp.github.create_issue`, `ext.linear.search`.
- `Descriptor`: identity, description, input schema, optional output schema, read-only/destructive/open-world/concurrency metadata, source/provenance, visibility, tags, owner, result budget.
- `Handle`: descriptor plus `Availability(ctx, ToolContext)` and `Call(ctx, ToolCall)` for executable tools.
- `Provider`: contributes descriptors/handles and can refresh.
- `Registry`: owns provider registration, indexing, listing, search, policy projection, and dispatch.
- `ToolResult`: structured output, preview, artifacts, redactions, bytes, display title, metadata.

This avoids overloading the desired-state resource with function pointers while still allowing resource records to feed the runtime registry.

### 2. Use manifest-first extension tools

Extension manifests should continue to declare tool metadata statically. Add enough metadata to connect the declaration to a backend:

- backend kind: `mcp`, `extension_host`, `subprocess`, or `builtin` where appropriate;
- namespace/owner;
- visibility;
- risk class;
- required config/env/capabilities;
- optional toolset memberships.

The registry can list these tools without executing extension code. A tool becomes executable only when its backend adapter is healthy, authorized, and has a callable handle.

Recommendation for MVP: no in-process third-party handlers. Built-in Go tools can register in-process. Extension tools should execute through MCP or an extension sidecar/Host API adapter.

### 3. Make availability a state machine, not a boolean

Use explicit status:

- `registered`: descriptor exists.
- `enabled`: operator/session policy has not disabled it.
- `available`: dependencies are present and backend is healthy.
- `authorized`: caller policy permits visibility/use.
- `executable`: there is a live handle for dispatch.
- `conflicted`: id/name collision requires resolution.

Discovery can hide unavailable/unauthorized tools from agents while operator surfaces show reasons. Dispatch must recheck availability and authorization.

### 4. Centralize dispatch

Every AGH-owned tool call should pass through:

1. resolve tool id/alias in context,
2. validate input against schema,
3. compute availability,
4. evaluate policy and session permission atoms,
5. run `tool.pre_call` hooks,
6. enforce concurrency/rate/result budgets,
7. call provider adapter,
8. normalize result,
9. redact/truncate/persist output,
10. run `tool.post_call` or `tool.post_error` hooks,
11. emit telemetry.

No CLI, HTTP, UDS, MCP, extension, or session path should bypass this pipeline.

### 5. Model policy as overlays

Use one policy engine that combines:

- daemon defaults,
- workspace config,
- extension grants,
- agent definition,
- session lineage `SessionPermissionPolicy.Tools`,
- skill/command scoped grants where relevant,
- explicit allow/deny patterns,
- named toolsets,
- risk defaults.

Toolsets should be recursive resources/config entries. This copies Hermes' strongest idea while fitting AGH's resource model.

### 6. Provide a small bootstrap native toolset

The TechSpec should not enumerate every future AGH tool. It should require a small proving set:

- `agh.tool.list`
- `agh.tool.search`
- `agh.tool.info`
- `agh.skill.list`
- `agh.skill.view`

Optional later groups:

- `agh.skill.install`
- `agh.network.peers`
- `agh.network.send`
- `agh.task.*`
- `agh.extension.*`

The bootstrap set proves discovery, schema loading, skill body loading, policy, result budget, and telemetry without overcommitting the whole daemon.

### 7. Expose agent-manageable surfaces

The registry should have shared contract types used by:

- CLI: `agh tool list/search/info/invoke`.
- HTTP: `/api/tools`, `/api/tools/{id}`, `/api/tools/{id}/invoke`.
- UDS: same operations for local agents and internal tools.
- Session-visible tool surface: either an AGH-hosted MCP server, driver-specific ACP tool injection where possible, or a fallback where agents can use `agh` CLI/UDS through their runtime.

The TechSpec should pick one MVP path and keep the others as contract-compatible surfaces.

### 8. Treat Tool Search as provider-neutral

Claude Code's `tool_reference` mechanism is useful but not portable. AGH should implement registry search as a normal catalog operation first:

- search over name, namespace, description, tags, source, toolset, and search hints;
- return metadata first;
- load schema/details on demand;
- optionally persist discovered state per session later.

Driver-specific schema-on-demand integration can be a future enhancement.

### 9. Reuse existing AGH infrastructure

Build on:

- `internal/tools` for contracts and registry,
- `internal/resources` for desired-state records,
- `internal/extension` manifest publication,
- `internal/hooks` for pre/post/error dispatch gates,
- `internal/store.SessionPermissionPolicy` for lineage constraints,
- `internal/skills.Registry` for skill listing/content,
- `internal/toolruntime` for subprocess ownership if extension tools need process handles,
- `internal/api/contract` for shared HTTP/UDS payloads.

Avoid a large generic `internal/catalog` at first unless it only coordinates cross-domain search. The runtime tool registry belongs in or near `internal/tools`; a catalog facade can compose tools and skills for `agh.tool.*` / `agh.skill.*`.

## Proposed MVP Scope

### In scope

- Runtime tool registry contract and central dispatch pipeline.
- Built-in provider for the small `agh.tool.*` and `agh.skill.*` set.
- Resource-backed descriptors from existing `tool` records.
- Extension manifest backend metadata for future executable extension tools.
- MCP adapter design, even if full MCP call-through is deferred.
- Context-specific list/search/info APIs.
- Tool policy with allow/deny patterns and named toolsets.
- Availability model and reason codes.
- Hook integration for pre/post/error.
- Telemetry events for list/search/info/call and failures.
- CLI/HTTP/UDS contract surfaces.

### Out of scope for MVP

- Full shell/browser/file tool replacement for ACP runtimes.
- Provider-specific Anthropic `tool_reference` integration.
- In-process third-party extension handlers.
- Large catalog of AGH-native tools.
- Network peer tool execution.
- Marketplace signing/trust overhaul, except for explicit risk hooks needed by extension tools.

## Critical Decisions Before TechSpec

1. Extension execution boundary: out-of-process only, metadata-only first, or trusted in-process handlers.
2. Session exposure path: hosted MCP, direct ACP extension, CLI/UDS fallback, or all in phases.
3. Package boundary: runtime registry in `internal/tools` with a catalog facade, or a new `internal/catalog` owning tools and skills together.
4. MVP tool set: only list/search/info/view, or include mutating install/network/task tools.
5. Policy defaults: external tools disabled, ask, or visible-but-not-callable until granted.
6. Availability visibility: hide unavailable tools from agents, show unavailable tools with reasons, or configurable by surface.
7. Naming/collision policy: always namespaced ids with optional display aliases, or global names with precedence.

## Competitor Pattern Matrix

| Pattern | Hermes | Claude Code | GoClaw | OpenClaw | AGH Recommendation |
|---|---:|---:|---:|---:|---|
| Single normalized tool contract | Yes | Yes | Yes | Yes | Required |
| Runtime executable registry | Yes | Distributed | Yes | Yes | Required |
| Manifest-first extension discovery | Partial | Plugin metadata | Partial | Strong | Required |
| MCP as adapter | Yes | Strong | Yes | Strong | Required |
| Availability gating | Strong discovery | `isEnabled` + MCP state | Policy/lazy checks | Lifecycle state | Required at discovery and dispatch |
| Central dispatch | Mostly | Strong | Mostly | Gateway + adapters | Required with no bypass |
| Toolsets/bundles | Strong | Policy lists | Groups | Policy groups | Required |
| Deferred search | Partial | Strong | Search helpers | Partial | Provider-neutral MVP |
| Concurrency metadata | Partial | Strong | Partial | Partial | Required metadata, scheduling can evolve |
| Extension in-process handlers | Yes | No native direct tools | Some | Plugin API | Avoid for MVP |

## Implementation Shape to Explore in TechSpec

```go
type Descriptor struct {
    ID          ToolID
    DisplayName string
    Description string
    InputSchema  json.RawMessage
    OutputSchema json.RawMessage
    Source      SourceRef
    Visibility  Visibility
    Risk        RiskClass
    ReadOnly    bool
    Destructive bool
    OpenWorld   bool
    ConcurrencySafe bool
    MaxResultBytes int64
    Toolsets []string
    Tags []string
}

type Handle interface {
    Descriptor() Descriptor
    Availability(ctx context.Context, call ToolContext) Availability
    Call(ctx context.Context, call ToolCall) (ToolResult, error)
}

type Provider interface {
    ID() string
    ListTools(ctx context.Context) ([]Descriptor, error)
    Resolve(ctx context.Context, id ToolID) (Handle, bool, error)
}

type Registry interface {
    List(ctx context.Context, scope Scope) ([]ToolView, error)
    Search(ctx context.Context, scope Scope, query SearchQuery) ([]ToolView, error)
    Get(ctx context.Context, scope Scope, id ToolID) (ToolView, error)
    Call(ctx context.Context, scope Scope, req CallRequest) (ToolResult, error)
}
```

The exact Go shape can change, but the separation should hold.

## Risks

If AGH exposes tool invocation before policy and availability are in place, it will create a broader attack surface than today's ACP-delegated tools.

If extension tools can run in-process, one bad extension can compromise the daemon.

If the registry only lists resources but does not dispatch, AGH will still lack the "last mile" that motivated the feature.

If the registry only works through one surface, agents will not be able to manage it consistently.

If name collisions are postponed, extension/MCP adoption will force a breaking change later.

## Evidence Index

- AGH current state: `analysis_agh_current_state.md`.
- Hermes reference: `analysis_hermes.md`.
- Claude Code reference: `analysis_claude-code.md`.
- GoClaw reference: `analysis_goclaw.md`.
- OpenClaw reference: `analysis_openclaw.md`.
- Local Claude Code ideas: `analysis_claude_code_ideas.md`.
- Prior autonomy gaps: `.compozy/tasks/autonomous/analysis/analysis_skills_tools_registry.md`.
- Security constraints: `.compozy/tasks/hermes/analysis/analysis_tools_security.md`.
