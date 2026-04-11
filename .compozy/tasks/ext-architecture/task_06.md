---
status: completed
title: Extension Manager (lifecycle orchestrator)
type: backend
complexity: high
dependencies:
  - task_02
  - task_04
  - task_05
---

# Task 06: Extension Manager (lifecycle orchestrator)

## Overview

Create the Extension Manager that orchestrates the 6-phase extension loading pipeline: DISCOVER → PARSE → VALIDATE → REGISTER → INITIALIZE → ACTIVATE. The Manager owns extension subprocess lifecycle, wires extensions into the existing hook declaration system, performs capability-negotiated handshakes, and handles crash recovery with exponential backoff. This is the critical-path component that ties together the manifest parser, capability checker, registry, and subprocess primitives.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `internal/extension/manager.go` with `Manager` struct following AGH's functional options pattern
- MUST implement `NewManager(registry *Registry, opts ...Option) *Manager`
- MUST implement `Start(ctx) error` that executes the 6-phase loading pipeline for every enabled extension
- MUST implement `Stop(ctx) error` that gracefully shuts down all subprocess extensions per protocol spec section 8
- MUST implement each of the 6 phases independently with clear error propagation
- MUST register extension resources (skills, agents, hooks, MCP configs) into the existing AGH registries without duplicating them
- MUST expose `HookDeclarations(ctx) ([]hooks.HookDecl, error)` for wiring into the existing `hooks.DeclarationProvider` pattern
- MUST launch subprocess extensions via `internal/subprocess/` package
- MUST perform capability-negotiated initialize handshake per protocol spec section 4
- MUST handle subprocess crash recovery with exponential backoff (1s, 2s, 4s, 8s, max 60s) and disable extension after 5 consecutive failures
- MUST handle subprocess hang via health check timeout (SIGTERM → wait 10s → SIGKILL)
- MUST expose extension health status via `ExtensionStatus` struct consumable by observer/health endpoint
</requirements>

## Subtasks
- [x] 6.1 Create `Manager` struct with functional options and dependencies (registry, capability checker, subprocess package)
- [x] 6.2 Implement 6-phase loading pipeline with per-phase error isolation
- [x] 6.3 Implement resource registration wiring into existing skills, agent def, hook declaration systems
- [x] 6.4 Implement subprocess launch with handshake using `internal/subprocess/` primitives
- [x] 6.5 Implement crash recovery with exponential backoff and failure threshold
- [x] 6.6 Implement `HookDeclarations()` provider for wiring into `hooks.Rebuild()`
- [x] 6.7 Write unit and integration tests covering pipeline phases, recovery, and shutdown

## Implementation Details

New file `internal/extension/manager.go` and `internal/extension/manager_test.go`. This is the largest single component of the extension architecture and pulls together tasks 02-05.

See TechSpec "Extension Loading Pipeline" section for the 6 phases. See TechSpec "Core Interfaces" for the `Manager` struct shape. See `_protocol.md` sections 3 and 4 for lifecycle and handshake rules.

Resource registration must NOT duplicate existing registries — the Manager calls into `skills.Registry`, appends to the hook declaration provider chain, and registers agent definitions through the existing config pattern.

### Relevant Files
- `internal/extension/registry.go` — Persistent extension state (task 05)
- `internal/extension/manifest.go` — Manifest parsing (task 03)
- `internal/extension/capability.go` — Capability enforcement (task 04)
- `internal/subprocess/process.go` — Subprocess lifecycle primitives (task 02)
- `internal/hooks/hooks.go` — `DeclarationProvider` pattern that Manager plugs into
- `internal/skills/registry.go` — Skills registry for resource registration
- `internal/daemon/hooks_bridge.go` — Existing pattern for wiring declaration providers

### Dependent Files
- `internal/daemon/boot.go` — Will initialize Manager in new boot phase (task 08)
- `internal/extension/host_api.go` — Will use Manager to look up extensions and enforce capabilities (task 07)
- `internal/cli/extension.go` — Will use Manager for install/enable/disable operations (task 09)

### Related ADRs
- [ADR-001: Two-Tier Extension Model](adrs/adr-001.md) — Manager owns L3 subprocess tier
- [ADR-005: Extension Three-Dimensional Package Model](adrs/adr-005.md) — Manager implements resource/capability/action loading phases

## Deliverables
- New `internal/extension/manager.go` with `Manager` struct, functional options, 6-phase pipeline
- Extension hook declaration provider for wiring into existing hooks system
- Crash recovery with exponential backoff
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for full extension lifecycle **(REQUIRED)**

## Tests
- Unit tests:
  - [x] `Start()` runs all 6 phases for each enabled extension
  - [x] `Start()` skips disabled extensions from registry
  - [x] DISCOVER phase finds manifests in configured extension directories
  - [x] PARSE phase returns error when manifest is invalid, continues other extensions
  - [x] VALIDATE phase rejects extensions with incompatible `min_agh_version`
  - [x] REGISTER phase adds resources to skills and hooks registries
  - [x] INITIALIZE phase launches subprocess and performs handshake
  - [x] ACTIVATE phase marks extension live and available for Host API
  - [x] Subprocess crash triggers restart with backoff
  - [x] 5 consecutive crashes disables extension and logs error
  - [x] `Stop()` sends shutdown to all subprocesses then waits with timeout
  - [x] `Stop()` escalates to SIGKILL after shutdown timeout
  - [x] `HookDeclarations()` returns declarations from all loaded extensions
  - [x] Failed extension in one phase does not block other extensions
- Integration tests:
  - [x] End-to-end: load test extension → handshake → receive Host API call → shutdown
  - [x] Restart recovery: kill subprocess → verify restart with correct backoff timing
  - [x] Resource registration: install extension with skills → verify skills appear in registry
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Full 6-phase pipeline validated end-to-end
- `make verify` passes
