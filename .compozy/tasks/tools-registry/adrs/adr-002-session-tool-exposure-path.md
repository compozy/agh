# ADR-002: Session Tool Exposure Path

## Status

Accepted

## Context

The Tool Registry must be agent-manageable, not only an internal daemon API. AGH manages external ACP-compatible runtimes, so it cannot assume direct control over every provider's internal model API tool loop.

The registry still needs a model-visible path for session-callable tools such as `agh__tool_search`, `agh__skill_view`, `ext__linear__search`, and `mcp__github__create_issue`. Competitor research suggests MCP is the most portable first boundary:

- Claude Code and other runtimes already understand MCP tools.
- MCP keeps tool exposure protocol-based rather than driver-specific.
- MCP can be backed by the same registry dispatch path as CLI, HTTP, and UDS.
- Runtimes without MCP can still reach AGH through CLI/UDS fallback surfaces.

## Decision

The MVP will expose session-callable registry tools through an AGH-hosted local MCP server plus shared CLI, HTTP, and UDS contracts.

The daemon will own one registry contract and dispatch pipeline. Surfaces call into that same pipeline:

- hosted MCP server for model-visible AGH, extension-host, and MCP-backed tools in runtimes that support MCP,
- CLI commands for operator and agent fallback use,
- HTTP API for web/operator clients,
- UDS API for local trusted clients and internal AGH tools.

Direct driver/ACP injection can be added later as an optimization for runtimes that support it, but it is not the MVP exposure path.

## Consequences

Every session-callable tool must be representable as an MCP tool without losing policy, availability, hook, telemetry, source provenance, auth redaction, approval, and result-budget behavior.

The registry contract types must be shared below all surfaces rather than copied into each transport.

Session start should be able to attach the hosted AGH MCP server for agents whose runtime supports MCP. Agents/runtimes without MCP remain supported through CLI/UDS fallback.

The hosted MCP server is not an execution backend. It is an exposure transport. `tools/list` returns the effective session projection, and every `tools/call` re-enters `internal/tools.Registry.Call`, which resolves `native_go`, `extension_host`, or `mcp` handles and revalidates policy at dispatch time.

The TechSpec must define how the hosted MCP server is authorized, scoped to the session/workspace, and prevented from bypassing registry dispatch.

Live catalog deltas can be designed as a later driver capability. The MVP can refresh the hosted MCP server's tool list and expose search/list tools through the registry.

## Rejected Alternatives

### CLI/UDS only

This would be simpler and still agent-operable through terminal tools, but it would not provide native tool calls for runtimes with MCP support and would leave the main "last mile" gap partially open.

### Direct ACP/driver injection first

This could be cleaner for a single provider, but it is less portable and would force the first implementation into provider-specific behavior.

### HTTP/UDS only

This would build the management API but delay the session-visible tool surface, making the foundation less useful to autonomous agents.

## Evidence

- `.compozy/tasks/tools-registry/analysis/analysis_claude-code.md`: MCP tools are adapted into the same local tool contract and refreshed dynamically.
- `.compozy/tasks/tools-registry/analysis/analysis_openclaw.md`: MCP is a provider backend for plugin/bundle tools.
- `.compozy/tasks/tools-registry/analysis/analysis_agh_current_state.md`: AGH already resolves MCP sidecars and has CLI/HTTP/UDS-style management surfaces elsewhere.
- `.compozy/tasks/tools-registry/analysis/analysis_claude_code_ideas.md`: AGH should avoid assuming direct LLM API control while still exposing AGH-owned tools through provider-neutral surfaces.
