# AGH Current State: Tool Registry Foundation

## Overview

AGH already has the cold side of a tool catalog, but not the runtime side.
`internal/tools` defines a canonical `tool` resource shape and the daemon projects tool records through the generic resources system. Extensions can publish static tool metadata from `extension.toml`. Sessions can also carry concrete permission atoms for tools in lineage metadata.

What is missing is the executable registry: a central service that can answer which tools exist for a specific agent/session, whether each tool is available now, whether the caller is allowed to use it, and how to dispatch the call through one uniform pipeline.

This matters because AGH's product premise is agent-first manageability. A tool registry is incomplete if it only helps internal Go code list metadata or if each ACP runtime owns a private tool universe that AGH cannot inspect, govern, or extend.

## Existing Mechanisms

### Tool resource metadata

`internal/tools/tool.go` defines `ToolSource` values for `builtin`, `mcp`, `extension`, and `dynamic`, plus a small `Tool` record:

- `Name`
- `Description`
- `InputSchema`
- `ReadOnly`
- `Source`

The only provider interface is:

```go
type ToolProvider interface {
    Tools(ctx context.Context) ([]Tool, error)
}
```

There is no `Call`, `Availability`, `CheckPermission`, `Aliases`, `Namespace`, `IsConcurrencySafe`, `IsDestructive`, `MaxResultBytes`, `Owner`, `Visibility`, or provenance-rich source metadata.

### Desired-state resources

`internal/tools/resource.go` defines `ToolResourceKind = "tool"` and validates tool records as JSON-object specs with a maximum size of 256 KiB. This is a good base for persisted inventory, desired-state reconciliation, and extension-published tool metadata.

The daemon already has a generic `resourceCatalog[T]` in `internal/daemon/tool_mcp_resources.go` and a `newToolProjector` that projects reconciled `tool` records into a daemon-local snapshot. This catalog is descriptive and revisioned, but it is not executable.

### Extension-published tools

`internal/extension/manifest.go` already lets extensions declare:

```go
type ResourcesConfig struct {
    Tools map[string]ToolConfig `toml:"tools,omitempty" json:"tools,omitempty"`
    MCPServers map[string]MCPServerConfig `toml:"mcp_servers,omitempty" json:"mcp_servers,omitempty"`
}
```

`ToolConfig` carries description, input schema, and read-only status. `ResolveManifestToolResources` converts these manifest entries into `toolspkg.Tool` records with `Source = ToolSourceExtension`. The daemon syncer publishes them into the resource graph with source keys like `extension/<name>/tool/<tool>`.

This is close to OpenClaw's manifest-first model, but AGH currently stops at metadata. A manifest-declared extension tool is not callable unless some separate ACP/MCP/runtime surface happens to expose it.

### MCP resources

The same extension manifest can declare MCP servers, and the daemon sync path resolves them into desired-state MCP server resources. Skills can also declare MCP sidecars. This gives AGH a strong candidate adapter for extension tools: manifest tools may be backed by an MCP server, an extension sidecar Host API endpoint, or native AGH built-ins, but all should normalize into one registry contract.

### Session permission atoms

`internal/store/session_lineage.go` defines `SessionPermissionPolicy` with concrete atoms:

- `Tools`
- `Skills`
- `MCPServers`
- `WorkspacePaths`
- `NetworkChannels`
- `SandboxProfiles`

`internal/session/spawn.go` validates child permissions as a subset of parent permissions. This is an important base for runtime tool policy because it is already persisted with session lineage and already participates in spawn delegation.

The current agent definition has a flat `Tools []string` field in `internal/config/agent.go`. It lacks allow/deny overlays, named toolsets, namespace patterns, visibility tiers, and risk classes.

### Hooks around tool calls

`internal/hooks/payloads.go` already defines `ToolPreCallPayload`, `ToolPostCallPayload`, `ToolPostErrorPayload`, `ToolCallPatch`, and `ToolResultPatch`. This is the right policy extension point for a centralized dispatch pipeline:

- pre-call hooks can deny or mutate input
- post-call hooks can redact or mutate output
- post-error hooks can classify or recover failures

The current gap is that AGH does not have a single dispatch pipeline that all AGH-owned tools must pass through.

### Skills registry contrast

`internal/skills.Registry` is much more mature than tools. It has global snapshots, workspace overlays, content loading, verification, install provenance, and `GlobalVersion()` for invalidation. Skills are injected as a static prompt catalog at session start, while tool resources are not exposed as a session-callable registry.

The Tool Registry should copy the skills registry's useful properties where they fit: global/workspace overlays, versioned snapshots, resource projection, progressive disclosure, and explicit content/schema loading.

## Gaps

1. No central runtime registry that owns executable tool handles.
2. No agent-facing discovery API for tools.
3. No native AGH tools such as `agh.tool.search`, `agh.skill.view`, or `agh.network.send`.
4. No availability model for env vars, binaries, MCP health, extension health, workspace scope, or policy state.
5. No central permission pipeline for AGH-owned tools.
6. No namespace or structured provenance model, so duplicate names would be ambiguous.
7. No toolsets or bundles comparable to skills/capabilities.
8. No direct extension execution boundary for manifest-declared tools.
9. No consistent way to expose tools over CLI, HTTP, UDS, and session-visible agent surfaces.
10. No usage telemetry by tool or skill.
11. No result-size budget, persistence policy, redaction, or output mapping at registry level.
12. The `dynamic` source enum exists but has no producer.

## Relevant Code Paths

- `internal/tools/tool.go:14-136`: tool source enum, metadata-only `Tool`, and list-only `ToolProvider`.
- `internal/tools/resource.go:13-61`: `ToolResourceKind` codec and JSON schema validation.
- `internal/daemon/tool_mcp_resources.go:20-122`: generic daemon `resourceCatalog` and `newToolProjector`.
- `internal/daemon/tool_mcp_resources.go:620-640`: extension manifest tools are projected into desired-state resources.
- `internal/extension/manifest.go:55-62`: extension resources can include tools and MCP servers.
- `internal/extension/manifest.go:154-160`: extension `ToolConfig` is static metadata only.
- `internal/extension/resource_publication.go:13-31`: manifest tool declarations become `toolspkg.Tool` records.
- `internal/config/agent.go:14-23`: `AgentDef.Tools []string` is flat.
- `internal/store/session_lineage.go:31-39`: session lineage has concrete `Tools` permission atoms.
- `internal/session/interfaces.go:244-251`: `AgentDriver` has no catalog-delta or AGH tool injection extension.
- `internal/hooks/payloads.go:520-568`: tool pre/post/error payloads already exist.
- `internal/skills/registry.go:100-103`: skills expose a global version suitable for catalog delta detection.
- `.compozy/tasks/autonomous/analysis/analysis_skills_tools_registry.md`: prior autonomy gap analysis with G1-G12 and P1-P9 proposals.
- `.compozy/tasks/hermes/analysis/analysis_tools_security.md`: security gaps relevant once AGH exposes agent-callable tools.

## Design Constraints for the TechSpec

The Tool Registry should be a foundation, not a pile of built-in commands. It should define the contracts, policy path, extension boundary, and surfaces first, then add a small bootstrap set of native AGH tools to prove the system.

AGH should avoid copying in-process plugin patterns from Python/TypeScript systems. Third-party executable tools should cross a process/protocol boundary: MCP, extension sidecar Host API, subprocess adapter, or future bridge SDK. Built-in Go tools can register in-process because they are part of the daemon binary.

The cold `tool` resource should remain valuable as catalog metadata and desired state, but executable dispatch must be modeled separately. A manifest-declared tool can be installed and discoverable while still being unavailable until its backend is healthy and authorized.

Availability and authorization must both be rechecked at dispatch time. Hiding unavailable tools from discovery is useful, but it is not a security boundary.

## Open Questions

1. Should extension tools be executable in the MVP, or should MVP only make them discoverable with an explicit unavailable state?
2. If executable, should extension tools be allowed only through MCP/sidecar boundaries, or should trusted bundled extensions get in-process handlers?
3. Should the first AGH-native tool surface be injected into ACP sessions, exposed as an MCP server hosted by AGH, or exposed only through CLI/UDS/HTTP for drivers to call indirectly?
4. Should `internal/catalog` coordinate tools and skills, or should `internal/tools` own runtime tools while a thinner catalog/search service composes skills and tools?
5. Which visibility tiers are needed for MVP: internal, CLI/HTTP, agent-visible, model-visible, deferred-discoverable, extension-private?

## Evidence

- `internal/tools/tool.go:91-136`: current `Tool` and `ToolProvider` are descriptive and list-only.
- `internal/tools/resource.go:13-61`: tool resource codec validates metadata records.
- `internal/extension/manifest.go:55-62`: extension manifests can publish `resources.tools`.
- `internal/extension/resource_publication.go:13-31`: extension manifest tools become static tool resources.
- `internal/daemon/tool_mcp_resources.go:620-640`: daemon sync publishes extension tools and MCP servers into resource desired state.
- `internal/store/session_lineage.go:31-39`: session permission policy already includes `Tools`.
- `internal/hooks/payloads.go:520-568`: tool lifecycle hooks are already typed.
- `.compozy/tasks/autonomous/analysis/analysis_skills_tools_registry.md:1-220`: prior gap analysis identifies no runtime tool registry, no discovery API, no availability, and no agent-facing skill/tool call surface.
- `.compozy/tasks/hermes/analysis/analysis_tools_security.md:1-140`: security analysis warns that URL-capable, command-capable, MCP, and skill-install surfaces require stronger guardrails before broad exposure.
