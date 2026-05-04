---
status: completed
title: Hook Management Tool Family
type: backend
complexity: high
dependencies:
  - task_01
  - task_05
---

# Task 06: Hook Management Tool Family

## Overview

Open hook inspection and mutation to agents where AGH is already the authoritative owner of the declaration, while preserving immutability for source-owned hook definitions. This task builds on the config mutation foundation from task_05 and keeps hook normalization, validation, and permission behavior on the existing runtime path.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, and ADR-006 before widening hook mutation
- REFERENCE TECHSPEC sections "Hooks", "Config Lifecycle", and "Post-Implementation Residual Checks"
- FOCUS ON WHAT: expose hook lifecycle where AGH owns the mutable overlay; do not create tool-only hook storage
- MINIMIZE CODE — reuse current hook normalization, introspection, and permission behavior
- TESTS REQUIRED — source-owned and secret-bearing mutations must fail deterministically
</critical>

<requirements>
1. MUST expose hook list/info/events/runs read surfaces to agents through tools.
2. MUST expose create/update/delete/enable/disable only for config-backed or overlay-backed hook declarations AGH owns.
3. MUST keep source-owned or extension-owned declarations structurally immutable through the tool surface.
4. MUST require approval for mutating hook operations and preserve current hook validation and permission rules.
</requirements>

## Subtasks

- [x] 6.1 Add hook inspection tools over current hook introspection and event surfaces
- [x] 6.2 Add hook mutation tools over the current config-backed or overlay-backed declaration lifecycle
- [x] 6.3 Enforce source ownership and secret-field denial rules for hook writes
- [x] 6.4 Wire approval and policy checks into hook mutation
- [x] 6.5 Add unit and integration coverage for read, mutate, and deny paths

## Implementation Details

See TechSpec sections "Hooks", "Config Lifecycle", and "Implementation Steps". This task should not invent a second hook model; it should project the existing normalized declaration lifecycle into the canonical tool surface.

### Relevant Files

- `internal/tools/builtin/hooks.go` — shipped hook-related built-in entry points
- `internal/config/hooks.go` — config-backed hook declaration model
- `internal/hooks/normalize.go` — canonical normalization and validation path
- `internal/hooks/introspection.go` — current read/introspection behavior
- `internal/cli/hooks.go` — current operator hook management semantics

### Dependent Files

- `internal/daemon/native_config_hook_tools.go` — daemon glue shared with config-backed mutation tooling
- `internal/hooks/permission.go` — current hook permission and execution boundaries

### Related ADRs

- [ADR-002: Tool Policy Is Recomputed Per Call With Separate Operator And Session Projections](adrs/adr-002-dynamic-tool-policy-and-projections.md)
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts`; checked `web/src/systems/*` and there is no dedicated hooks UI consumer that needs a new route in this task.
- `packages/site`: `packages/site/content/runtime/core/hooks/*.mdx` and CLI reference pages under `runtime/cli-reference/hooks/`.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: directly affects the hooks surface and must preserve source ownership and extension-provided hook immutability.
- Agent manageability: adds structured hook inspection and mutation instead of CLI-only or file-edit flows for AGH-owned declarations.
- Config lifecycle: hook mutation rides on the same config-backed overlay lifecycle established in task_05.

## Deliverables

- Hook inspection tools
- Hook mutation tools for AGH-owned declarations
- Deterministic immutability and secret-field denial behavior
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for hook lifecycle parity **(REQUIRED)**

## Tests

- Unit tests:
- [x] hook read tools expose current introspection data without leaking secret fields
- [x] source-owned or extension-owned hook declarations reject mutation with deterministic reason codes
- [x] config-backed hook writes reuse normalization and permission rules
- Integration tests:
- [x] tool-driven hook read and mutation flows match current operator behavior for the same runtime state
- [x] approval and denial semantics stay consistent across tool, CLI, HTTP, and UDS management surfaces
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Agents can inspect and manage AGH-owned hook declarations through tools
- Source-owned hooks remain immutable and secret-bearing fields never cross the tool surface
