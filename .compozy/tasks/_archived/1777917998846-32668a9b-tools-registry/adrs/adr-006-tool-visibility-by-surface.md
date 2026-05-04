# ADR-006: Tool Visibility by Surface

## Status

Accepted

## Context

The Tool Registry will track more states than "exists" or "does not exist." A tool may be registered but disabled, unauthorized, unavailable, unhealthy, missing configuration, missing an MCP backend, blocked by session policy, denied by ACP approval mode, or conflicted by name.

Different consumers need different views:

- operators need diagnostics and reason codes to fix configuration and extension problems;
- agents need a low-noise callable surface that does not invite impossible calls;
- dispatch still needs to revalidate because discovery visibility is not a security boundary.

## Decision

Operator surfaces show unavailable and unauthorized tools with reason codes. Session-visible and model-visible surfaces expose only tools that are visible and callable for the effective session context.

Operator surfaces include:

- CLI,
- HTTP API,
- Web UI,
- privileged UDS/operator views.

Session-visible/model-visible surfaces include:

- the AGH-hosted MCP tool list exposed to an agent session,
- any future direct ACP/driver tool injection,
- non-privileged session-scoped UDS catalog views.

The registry must compute both:

- `OperatorToolView`: includes all registered tools plus state, reason codes, source/provenance, policy diagnostics, conflict diagnostics, and availability details.
- `SessionToolView`: includes only tools that pass effective visibility and execution preconditions for that session.

Dispatch must revalidate the full effective execution decision even when a tool was present in `SessionToolView`.

## Consequences

Agents are not shown tools that they cannot call in the current session. This avoids prompt/tool-call noise and reduces attempts to invoke unavailable tools.

Operators can still debug why a tool is not appearing to an agent, including whether the cause is ACP approval mode, session lineage, agent policy, source grants, extension health, MCP health, missing config, or a conflict.

CLI/HTTP endpoints need an explicit scope or view mode. For example:

- operator default: include unavailable tools and reasons;
- session-scoped query: return the same filtered view that the hosted MCP server would expose.

The hosted MCP server must use `SessionToolView`, not raw registry contents.

## Rejected Alternatives

### Everyone sees unavailable tools

This improves agent planning transparency but increases noise and risks inducing models to call tools that the daemon will reject.

### Hide unavailable tools everywhere

This is clean for fail-closed execution, but it makes operator troubleshooting poor and hides extension/MCP/config problems.

### Configurable per surface in MVP

This offers maximum flexibility but creates a larger behavior matrix before the registry foundation is stable.

## Evidence

- `.compozy/tasks/tools-registry/analysis/analysis_hermes.md`: availability filtering is useful for model-visible definitions.
- `.compozy/tasks/tools-registry/analysis/analysis_claude-code.md`: request-time tool pools are context-specific and filtered before model exposure.
- `.compozy/tasks/tools-registry/analysis/analysis_openclaw.md`: lifecycle and policy states need diagnostics while agent projections should be policy-filtered.
- `.compozy/tasks/tools-registry/analysis/synthesis.md`: discovery can hide unavailable/unauthorized tools from agents while operator surfaces show reasons, but dispatch still rechecks.
