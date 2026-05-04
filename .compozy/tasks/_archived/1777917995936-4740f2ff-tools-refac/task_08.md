---
status: completed
title: Extension Lifecycle Tool Family
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 08: Extension Lifecycle Tool Family

## Overview

Expose the managed extension lifecycle through dedicated tools so agents can search, install, update, remove, enable, and disable extensions through structured AGH surfaces. This task must reuse the shipped extension manager and marketplace/install flows already present on this branch, with source policy and approval doing the containment.

<critical>
- ALWAYS READ `_techspec.md`, ADR-004, and ADR-006 before widening extension mutation
- REFERENCE TECHSPEC sections "Extension Manifests And Runtime Extension Points", "Config Lifecycle", and "Post-Implementation Residual Checks"
- FOCUS ON WHAT: project the existing extension lifecycle into tools; do not create a second extension installation pipeline
- MINIMIZE CODE — reuse current extension manager, registry, install, and marketplace flows
- TESTS REQUIRED — install/update/remove flows must prove trust-source, approval, and rollback behavior
</critical>

<requirements>
1. MUST expose extension search, install, update, remove, enable, and disable through built-in tools.
2. MUST reuse the current extension manager and managed-install lifecycle, including marketplace and local-source distinctions.
3. MUST enforce trust-source checks, approval requirements, and deterministic denials for forbidden installs or updates.
4. MUST preserve current reconciliation behavior between installed extensions and the shipped registry/tool surfaces.
</requirements>

## Subtasks

- [x] 8.1 Add extension discovery and status tools over the current extension registry and manager
- [x] 8.2 Add install/update/remove tools over the current managed-install and marketplace flows
- [x] 8.3 Add enable/disable tools over the existing runtime extension lifecycle
- [x] 8.4 Enforce trust-source policy, approval, and rollback behavior for mutating extension operations
- [x] 8.5 Add unit and integration coverage for lifecycle parity and failure paths

## Implementation Details

See TechSpec sections "Extension Manifests And Runtime Extension Points", "Agent Manageability Plan", and "Implementation Steps". The canonical surface should converge on the existing extension manager and reconciliation flow, not layer an alternate tool-only lifecycle on top.

### Relevant Files

- `internal/extension/manager.go` — authoritative extension lifecycle and runtime coordination
- `internal/extension/registry.go` — installed-extension registry and metadata persistence
- `internal/extension/install_managed.go` — managed install pipeline that tools must reuse
- `internal/cli/extension.go` — current extension management semantics
- `internal/cli/extension_marketplace.go` — current marketplace install/update/remove behavior

### Dependent Files

- `internal/daemon/extensions.go` — daemon extension service wiring that tool calls must converge on
- `internal/extension/tool_reconciliation.go` — current reconciliation with tool surfaces that must stay consistent

### Related ADRs

- [ADR-004: MCP Auth Exposes Agent Status Only; Login And Logout Stay On Management Surfaces](adrs/adr-004-mcp-auth-status-tool.md)
- [ADR-006: Mutable AGH Management Surfaces Are Tool-Callable By Default](adrs/adr-006-agent-manageable-mutation-default.md)

### Web/Docs Impact

- `web/`: `web/src/generated/agh-openapi.d.ts`; checked `web/src/systems/*` and there is no dedicated extension management UI system on this branch, so no new web route is expected here.
- `packages/site`: `packages/site/content/runtime/core/extensions/*.mdx` and CLI reference pages under `runtime/cli-reference/extension/`.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: directly affects extension manifests, installation sources, reconciliation, and future bridge/tool/resource publication from extensions.
- Agent manageability: removes the remaining CLI-only gap for extension lifecycle management.
- Config lifecycle: must respect existing extension marketplace config, install roots, and source metadata without introducing compatibility shims.

## Deliverables

- Extension discovery and lifecycle tool family
- Trust-source and approval-integrated install/update/remove behavior
- Reconciliation-safe enable/disable semantics
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for extension lifecycle parity **(REQUIRED)**

## Tests

- Unit tests:
  - [x] install/update/remove tools reuse current manager and registry validation paths
  - [x] disallowed or untrusted install sources fail with deterministic reasons
  - [x] enable/disable tools preserve current reconciliation and runtime activation semantics
- Integration tests:
  - [x] tool-driven extension search/install/update/remove/enable/disable matches current operator behavior for the same source and runtime state
  - [x] failed installs or updates roll back registry or on-disk state the same way the current manager does
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- Agents can manage the full extension lifecycle through dedicated tools
- Trust-source boundaries, approvals, and reconciliation behavior stay aligned with the existing authoritative extension manager
