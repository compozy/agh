# Competitor Analysis: GoClaw Tool Registry

## Overview

GoClaw is the closest Go-native reference. It has a runtime `tools.Tool` interface with executable behavior, a registry that owns aliases, metadata, disabled state, rate limiting, grouping, deferred activation, and an executor/policy layer. It also adapts MCP tools into the same local interface.

The useful pattern for AGH is not GoClaw's exact `map[string]any` API. It is the separation between executable tool contract, registry metadata, run-scoped context, policy filtering, MCP adaptation, and dispatch hooks.

## Mechanisms / Patterns

GoClaw's core tool contract includes `Name`, `Description`, `Parameters`, and `Execute(context.Context, map[string]any) *Result`. This is weaker than the typed API AGH should build, but it proves the right direction: tools are executable handles, not only metadata records.

The registry owns tools, metadata, aliases, disabled state, rate limiting, result scrubbing, groups, and deferred activation. `ExecuteWithContext` injects run-scoped data through `context.Context` instead of mutating shared tool instances.

The policy engine filters tools through global, provider, agent, group, capability, subagent, and sandbox rules. Lazy/deferred activation rechecks policy before exposing or using tools.

MCP bridge tools implement the same interface. The MCP adapter converts remote tools into local handles, rechecks grants at runtime, and normalizes results.

Hooks are lifecycle gates, not tools. `pre_tool_use` can block tool execution, while post hooks can observe and mutate limited fields. This maps well to AGH's existing hook payloads.

Skills are exposed partly through searchable artifacts and marker/no-op tools such as skill search/use. AGH should be cautious here: "use skill" can be useful telemetry, but skill content loading should be a real catalog operation, not just a marker.

## Relevant Code Paths

- `.resources/goclaw/internal/tools/types.go:14-129`: core executable tool interface and result types.
- `.resources/goclaw/internal/tools/registry.go:18-459`: registry, aliases, groups, disabled state, rate limiting, scrubber, and deferred activation.
- `.resources/goclaw/internal/tools/executor.go`: central execution support.
- `.resources/goclaw/internal/tools/policy.go:13-520`: multi-layer tool policy and filtering.
- `.resources/goclaw/internal/tools/capability.go`: capability inference and policy integration.
- `.resources/goclaw/internal/tools/result.go`: result model.
- `.resources/goclaw/internal/mcp/bridge_tool.go:42-155`: MCP bridge implements local tool contract.
- `.resources/goclaw/internal/mcp/manager.go:318-515`: MCP manager and tool lifecycle.
- `.resources/goclaw/internal/mcp/grant_checker.go:46-129`: runtime grant checks.
- `.resources/goclaw/internal/mcp/mcp_tool_search.go:67-101`: MCP tool search support.
- `.resources/goclaw/internal/agent/loop_tool_filter.go:22-96`: agent loop filtering.
- `.resources/goclaw/internal/pipeline/tool_stage.go:51-152`: pipeline stage around tool execution.
- `.resources/goclaw/internal/hooks/types.go:19-44`: hook types.
- `.resources/goclaw/internal/hooks/dispatcher.go:153-318`: hook dispatch and mutation.
- `.resources/goclaw/migrations/000001...:478-499`, `000027...:230-245`: custom tool storage history.

## Transferable Patterns

AGH should promote `tools.Tool` from a record to a runtime contract, but with stronger types than GoClaw. Prefer `json.RawMessage` plus schema validation and typed `ToolResult` over unconstrained `map[string]any` crossing every boundary.

AGH should pass per-call/session/workspace/user information through a `ToolCallContext` or context-bound immutable values, not by mutating registry entries.

AGH should keep external adapters under the same registry: MCP tools, extension sidecar tools, and future bridge tools should all be executable through `Registry.Call`.

AGH should implement dynamic groups/toolsets as policy inputs, not as separate registries.

AGH should recheck grants at runtime even when discovery already filtered a tool.

AGH should treat hooks as gates around dispatch, not as an alternative dispatch surface.

AGH should persist metadata separately from executable code. Installed or extension-provided tool records can remain in resource storage while executable backends are resolved at runtime.

## Risks / Mismatches

GoClaw's `map[string]any` API is weak for AGH. AGH already has JSON schema resources and generated API contracts, so it should preserve schema validation and raw JSON boundaries.

Some GoClaw paths can bypass pre-tool hooks in parallel execution. AGH should enforce one central dispatch pipeline regardless of whether tools run concurrently.

Shared registry state can leak per-user or per-session MCP availability if not scoped carefully. AGH should compute context-specific views rather than storing "available for everyone" as a global truth.

Capability inference by tool name is brittle. AGH should use explicit metadata and namespaces.

Individual tool bodies should not own all security logic. The registry must own common gates: permission, availability, schema validation, risk class, hooks, result budget, redaction, telemetry, and concurrency.

## Open Questions

1. Should AGH's runtime registry be global with context-specific projections, or per-session snapshots derived from a global registry?
2. Should tool concurrency be enforced by a registry scheduler or by individual handlers returning metadata?
3. How should AGH persist disabled state and operator overrides for extension tools?
4. Should AGH include marker tools for skill use telemetry, or should skill view/install/load actions be real tools only?

## Evidence

- `.resources/goclaw/internal/tools/types.go:14-129`: executable Go tool contract.
- `.resources/goclaw/internal/tools/registry.go:18-459`: registry ownership of tools, aliases, groups, disabled state, and deferred activation.
- `.resources/goclaw/internal/tools/policy.go:13-520`: multi-layer policy.
- `.resources/goclaw/internal/mcp/bridge_tool.go:42-155`: MCP adapter as local tool.
- `.resources/goclaw/internal/mcp/manager.go:318-515`: MCP lifecycle.
- `.resources/goclaw/internal/mcp/grant_checker.go:46-129`: runtime grant checking.
- `.resources/goclaw/internal/agent/loop_tool_filter.go:22-96`: agent-specific filtering.
- `.resources/goclaw/internal/pipeline/tool_stage.go:51-152`: tool execution pipeline.
- `.resources/goclaw/internal/hooks/dispatcher.go:153-318`: hook-driven blocking/mutation.
- `.resources/goclaw/migrations/000001...:478-499`, `000027...:230-245`: persisted custom-tool metadata history.
