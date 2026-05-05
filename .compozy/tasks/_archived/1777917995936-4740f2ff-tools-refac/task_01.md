---
status: completed
title: Dynamic Policy Input Resolver and Default Discovery Overlay
type: backend
complexity: critical
dependencies: []
---

# Task 01: Dynamic Policy Input Resolver and Default Discovery Overlay

## Overview

Extend the shipped registry so every `list/search/get/call` projection resolves effective policy from current runtime state instead of mostly boot-time config inputs. This task also establishes the default discovery overlay for `agh__bootstrap` and `agh__catalog`, which the rest of the canonical surface assumes.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, and ADR-002 before changing registry evaluation paths
- REFERENCE TECHSPEC sections "Implementation Design", "Agent Manageability Plan", and "Build Order" instead of copying policy rules here
- FOCUS ON WHAT: this task resolves runtime policy inputs and default discovery; it does not implement the later tool families
- MINIMIZE CODE in the task body; reuse the shipped `tools.Scope`, `Registry`, and `PolicyEvaluator` contracts
- TESTS REQUIRED — projection and dispatch must prove the same runtime policy inputs are authoritative
</critical>

<requirements>
1. MUST resolve effective `PolicyInputs` per call from current agent definition, session lineage, source policy, availability state, and hook outputs.
2. MUST apply `agh__bootstrap` and `agh__catalog` as default discovery toolsets unless effective policy narrows or denies them.
3. MUST preserve separate operator and session/model projections while keeping `Registry.Call` authoritative for execution-time revalidation.
4. MUST keep caches, if any, strictly invalidatable accelerators rather than a second source of truth.
5. MUST emit deterministic reason codes for unavailable, denied, hook-blocked, or source-health-blocked tools.
</requirements>

## Subtasks
- [x] 1.1 Add a runtime policy-input resolver that reuses the shipped registry evaluation path
- [x] 1.2 Thread agent, session, lineage, source, and hook state into `list/search/get/call`
- [x] 1.3 Apply the default discovery overlay for `agh__bootstrap` and `agh__catalog`
- [x] 1.4 Preserve separate operator diagnostics and session-callable projections
- [x] 1.5 Add unit and integration coverage for projection parity, invalidation, and dispatch revalidation

## Implementation Details

See TechSpec sections "Core Interfaces", "Data Models", "Agent Manageability Plan", and "Implementation Steps". This task is foundational for every later tool family and should land before any new built-in expansion.

### Relevant Files
- `internal/tools/policy.go` — shipped evaluator contracts and policy input model
- `internal/tools/registry.go` — projection and call entry points that must keep one evaluation path
- `internal/daemon/native_tools.go` — current runtime wiring for registry policy inputs
- `internal/config/tools.go` — agent/toolset config grammar already parsed on this branch
- `internal/session/manager.go` — current source of session and lineage state

### Dependent Files
- `internal/api/core/tools.go` — operator-visible diagnostics and projection rendering
- `internal/daemon/native_tools_test.go` — integration coverage for daemon-wired policy behavior

### Related ADRs
- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md)
- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)

### Web/Docs Impact
- `web/`: `web/src/generated/agh-openapi.d.ts` if tool or toolset projection payloads change; checked `web/src/systems/*/adapters` and there is no dedicated direct consumer for `/api/tools` or `/api/toolsets` beyond generated types.
- `packages/site`: `packages/site/content/runtime/core/configuration/agent-md.mdx`, `packages/site/content/runtime/core/agents/definitions.mdx`, and generated CLI docs for `agh tool*` / `agh toolsets*` if default discovery semantics or diagnostics change.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: affects registry toolsets, source availability, hook-deny behavior, and future built-in or extension-provided tools that rely on the same evaluation path.
- Agent manageability: affects `agh tool list`, `agh tool search`, `agh tool info`, `agh tool invoke`, `/api/tools`, `/api/toolsets`, and hosted MCP/session-visible tool projections.
- Config lifecycle: affects interpretation of agent `tools`, `toolsets`, and `deny_tools` declarations in config and `AGENT.md`, but should not add new top-level config keys.

## Deliverables
- Runtime policy-input resolver reused by projection and dispatch
- Default discovery overlay for `agh__bootstrap` and `agh__catalog`
- Deterministic operator/session projection behavior with invalidation rules
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for projection and dispatch parity **(REQUIRED)**

## Tests
- Unit tests:
  - [x] effective policy expands default discovery only when no stronger deny or narrower allow applies
  - [x] operator projections retain deny/unavailable reasons while session projections expose only callable tools
  - [x] cached projections invalidate when agent, session, source-health, or hook inputs change
- Integration tests:
  - [x] daemon-wired `list/search/get/call` uses the same runtime-resolved inputs for projection and execution
  - [x] `agh tool list` / `/api/tools` / hosted MCP surface reflect current session policy instead of boot-time-only inputs
- Test coverage target: >=80%
- All tests must pass

## Verification Evidence

- `go test -cover ./internal/tools` => 80.7% statements
- `go test ./internal/tools ./internal/daemon`
- `make verify` => passed after implementation changes
- Local commit: `a4601294 feat: add dynamic tool policy resolver`
- Post-commit `make verify` => passed; Go lint `0 issues.`, Go tests `DONE 7008 tests in 10.381s`, package boundaries `OK: all package boundaries respected`

## Success Criteria
- All tests passing
- Test coverage >=80%
- Runtime policy decisions reflect current agent, session, source, and hook state on every call path
- Default discovery for `agh__bootstrap` and `agh__catalog` is visible and auditable without duplicating config into every agent definition

## References
- `.compozy/tasks/tools-refac/analysis/competitor-tool-surface-notes.md`
- `.resources/claude-code/tools.ts`
- `.resources/claude-code/services/tools/toolExecution.ts`
- `.resources/claude-code/utils/permissions/permissions.ts`
- `.resources/hermes/tools/registry.py`
- `.resources/openclaw/src/agents/tool-policy-pipeline.ts`
