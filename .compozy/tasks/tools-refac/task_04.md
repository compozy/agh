---
status: completed
title: Memory, Observe, and Bridge Read Surfaces
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 04: Memory, Observe, and Bridge Read Surfaces

## Overview

Expose the remaining high-value runtime inspection domains that agents need for self-management: memory, observability, and bridge state. This task keeps the scope read-only and uses existing query surfaces so agents gain structured access without inventing new persistence or management logic.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, and ADR-002 before widening inspection surfaces
- REFERENCE TECHSPEC sections "Implementation Design", "Monitoring and Observability", and "Docs And Generated Surfaces"
- FOCUS ON WHAT: add memory, observe, and bridge reads only; writes belong to later tasks
- MINIMIZE CODE — reuse current managers and query surfaces rather than adding tool-specific storage paths
- TESTS REQUIRED — read surfaces must preserve redaction and existing visibility semantics
</critical>

<requirements>
1. MUST add read-only memory tools for list, read, search, and history-style inspection over the current memory system.
2. MUST add read-only observe tools for events, health/metrics-style summaries, and searchable inspection where the current runtime already supports it.
3. MUST add read-only bridge tools for list and status/health inspection.
4. MUST preserve redaction and visibility rules for sensitive content and bridge credentials.
</requirements>

## Subtasks

- [x] 4.1 Add built-in memory descriptors and handlers over the current memory query surfaces
- [x] 4.2 Add built-in observe descriptors and handlers over the current observe and health query surfaces
- [x] 4.3 Add built-in bridge descriptors and handlers over current bridge inspection surfaces
- [x] 4.4 Register the new read-only tool IDs and toolsets in the shipped registry
- [x] 4.5 Add tests for redaction, parity, and policy-filtered exposure

## Implementation Details

See TechSpec sections "Data Models", "Monitoring and Observability", and "Implementation Steps". The task should extend the built-in catalog only where the underlying runtime already has authoritative query behavior.

### Relevant Files

- `internal/api/core/memory.go` — current memory read DTOs and handlers
- `internal/observe/query.go` — current observability query layer
- `internal/observe/health.go` — current runtime health summaries
- `internal/api/core/bridges.go` — current bridge inspection and status behavior
- `internal/tools/builtin/toolsets.go` — built-in toolset registration for new read families

### Dependent Files

- `internal/cli/memory.go` — current memory management/read semantics
- `internal/cli/observe.go` — current observe CLI semantics

### Related ADRs

- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md)
- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts`; checked `web/src/systems/bridges/*` and existing bridge UI should continue to consume the same bridge DTOs even if tool projection shapes grow separately.
- `packages/site`: `packages/site/content/runtime/core/memory/*.mdx`, `packages/site/content/runtime/core/bridges/*.mdx`, `packages/site/content/runtime/cli-reference/memory/*`, `packages/site/content/runtime/cli-reference/observe/*`, and `packages/site/content/runtime/cli-reference/bridge/*`.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: affects built-in tool expansion for memory, observability, and bridges; no new extension or protocol surface is introduced.
- Agent manageability: gives agents structured inspection of runtime state they currently access through CLI or web-only diagnostics.
- Config lifecycle: no new config keys expected; tools must honor current memory scope, bridge visibility, and observe retention configuration.

## Deliverables

- Read-only memory built-ins
- Read-only observe built-ins
- Read-only bridge built-ins
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for redaction and parity **(REQUIRED)**

## Tests

- Unit tests:
  - [x] memory tools preserve current scope filtering and redact forbidden content
  - [x] observe tools do not expose secret-bearing payloads or raw tokens in event output
  - [x] bridge tools expose status/health without leaking credentials or secret config material
- Integration tests:
  - [x] tool-driven memory and observe reads match the current CLI or API views for the same runtime state
  - [x] bridge list/status tools agree with existing bridge inspection surfaces and policy filters
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Agents can inspect memory, runtime health/events, and bridge status through dedicated tools
- Redaction and visibility guarantees remain intact across tool, CLI, HTTP, and UDS paths
