---
status: completed
title: Coordination, Session, and Workspace Read Surfaces
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 03: Coordination, Session, and Workspace Read Surfaces

## Overview

Expand the shipped built-in read surface around coordination, sessions, and workspaces while preserving `agh__coordination` as the current network-facing toolset already present on this branch. The goal is to close the remaining read-only gaps that still force agents toward CLI paths for basic runtime inspection.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, and ADR-002 before extending built-in read surfaces
- REFERENCE TECHSPEC sections "MVP Boundary Statement", "Implementation Design", and "Agent Manageability Plan"
- FOCUS ON WHAT: add read surfaces only; do not mix in autonomy writes or config mutation
- MINIMIZE CODE — build on the shipped built-in registry, IDs, and toolset catalog already in `internal/tools/builtin`
- TESTS REQUIRED — every new read projection must prove CLI/HTTP/UDS parity where the underlying management surface already exists
</critical>

<requirements>
1. MUST expand `agh__coordination` to cover the current network read verbs needed by agents (`status`, `channels`, `inbox`) in addition to shipped discovery/send peers behavior.
2. MUST add read-only session inspection tools for list, status, history, events, and describe flows.
3. MUST add read-only workspace inspection tools for list, info, and describe flows.
4. MUST preserve operator/session projection rules from task_01 when these tools are unavailable or denied.
</requirements>

## Subtasks
- [x] 3.1 Extend built-in coordination descriptors and handlers for network inspection verbs
- [x] 3.2 Add session read-only built-ins over the current session query and history paths
- [x] 3.3 Add workspace read-only built-ins over the current workspace discovery paths
- [x] 3.4 Register the new tool IDs and toolset expansions without renaming shipped `agh__coordination`
- [x] 3.5 Add unit and transport-parity coverage for the new read surfaces

## Implementation Details

See TechSpec sections "Component Overview", "Bootstrap Task Tools", and "Implementation Steps". Use the shipped registry and built-in descriptor patterns instead of adding a parallel registration model.

### Relevant Files
- `internal/tools/builtin/network.go` — shipped coordination built-ins and extension point for more network reads
- `internal/tools/builtin/toolsets.go` — canonical built-in toolset expansion
- `internal/tools/builtin_ids.go` — built-in ID registry
- `internal/cli/session.go` — current session operator read surface to mirror semantically
- `internal/cli/workspace.go` — current workspace operator read surface to mirror semantically

### Dependent Files
- `internal/api/core/network.go` — current network-facing DTO and query behavior
- `internal/api/core/session_workspace.go` — shared session/workspace read helpers used by public surfaces

### Related ADRs
- [ADR-001: Agent Tool Surface Is Tool-First With Default Discovery](adrs/adr-001-agent-tool-surface.md)
- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)

### Web/Docs Impact
- `web/`: `web/src/generated/agh-openapi.d.ts` if public tool projection payloads change; no dedicated web consumer currently calls `/api/tools` or `/api/toolsets`, but network/session/workspace state already exists in `web/src/systems/network/*`, `web/src/systems/session/*`, and `web/src/systems/workspace/*` and should not drift from shared DTOs.
- `packages/site`: `packages/site/content/runtime/core/network/*.mdx`, `packages/site/content/runtime/core/sessions/*.mdx`, `packages/site/content/runtime/core/workspaces/*.mdx`, plus CLI reference pages under `runtime/cli-reference/network/`, `session/`, and `workspace/`.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: affects built-in toolset expansion and future extension-provided tools that may rely on the same session/workspace context.
- Agent manageability: closes CLI-only read gaps for coordination, sessions, and workspaces across tool, CLI, HTTP, and UDS surfaces.
- Config lifecycle: no new config keys; must honor existing workspace resolution and session visibility rules already defined in runtime config.

## Deliverables
- Expanded coordination read built-ins under the existing `agh__coordination` toolset
- New session and workspace read-only built-ins
- Deterministic projection and denial behavior for these read surfaces
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for transport and semantic parity **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `agh__coordination` expands to the new network inspection tool IDs deterministically
  - [x] session and workspace read built-ins preserve existing visibility and not-found semantics
  - [x] unavailable or denied projections show the correct operator-only reasons without leaking into session-visible catalogs
- Integration tests:
  - [x] tool-driven network status/channels/inbox matches existing management behavior for the same runtime state
  - [x] session list/status/history/events and workspace list/info/describe built-ins agree with CLI/HTTP/UDS data for the same caller scope
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Agents can inspect coordination, session, and workspace state without falling back to CLI-first runtime queries
- `agh__coordination` remains the shipped network-facing toolset name while covering the canonical read surface
