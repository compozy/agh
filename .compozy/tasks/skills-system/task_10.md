---
status: pending
title: Daemon boot integration
type: backend
complexity: high
dependencies:
  - task_03
  - task_04
  - task_05
  - task_06
  - task_08
  - task_09
---

# Task 10: Daemon boot integration

## Overview

Wire the skills registry, watcher, catalog provider, and composed assembler into the daemon's boot and shutdown sequences. This is the integration task that connects all skills components into the running daemon, replacing the current memory-gated assembler with the unconditional composed assembler.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST refactor `daemon.boot()` to construct `ComposedAssembler` unconditionally (outside Memory.Enabled branch)
- MUST create skills `Registry` when `cfg.Skills.Enabled` is true
- MUST call `registry.LoadAll()` during boot
- MUST start `Watcher` goroutine during boot (tracked with context cancellation)
- MUST create `CatalogProvider` from registry
- MUST add memory and skills providers to `ComposedAssembler` based on their enabled state
- MUST pass `ComposedAssembler` as `PromptAssembler` to `SessionManagerDeps`
- MUST stop Watcher on shutdown (before session manager shutdown)
- MUST support: memory=on+skills=on, memory=on+skills=off, memory=off+skills=on, both=off
- MUST add skills watcher to cleanup chain for graceful shutdown
</requirements>

## Subtasks
- [ ] 10.1 Refactor boot() to extract PromptAssembler construction from Memory.Enabled branch
- [ ] 10.2 Add skills Registry creation and LoadAll in boot()
- [ ] 10.3 Add Watcher goroutine lifecycle management (start in boot, stop in shutdown)
- [ ] 10.4 Create ComposedAssembler with conditional providers
- [ ] 10.5 Update RuntimeDeps or SessionManagerDeps if needed
- [ ] 10.6 Write integration tests for all feature-flag combinations

## Implementation Details

See TechSpec "Daemon Boot Sequence" section for the exact integration point and code sketch. The critical change is at `daemon.go:587-602` where the current code only sets `promptAssembler` inside `cfg.Memory.Enabled`.

### Relevant Files
- `internal/daemon/daemon.go` — boot() at line 540, current memory-gated assembler at line 593-602
- `internal/skills/registry.go` — Registry to create (task_03)
- `internal/skills/catalog.go` — CatalogProvider to create (task_04)
- `internal/skills/bundled/embed.go` — BundledFS for RegistryConfig (task_05)
- `internal/skills/watcher.go` — Watcher to start/stop (task_06)
- `daemon/composed_assembler.go` — ComposedAssembler to wire (task_09)

### Dependent Files
- `internal/session/manager.go` — Receives PromptAssembler via deps (unchanged interface)

### Related ADRs
- [ADR-003: Composed PromptAssembler](../adrs/adr-003.md) — Unconditional assembler construction

## Deliverables
- Modified `internal/daemon/daemon.go` with skills integration in boot/shutdown
- Integration tests for daemon boot with skills
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for feature-flag combinations **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Boot with skills.enabled=true + memory.enabled=true: both providers wired
  - [ ] Boot with skills.enabled=true + memory.enabled=false: only skills provider wired
  - [ ] Boot with skills.enabled=false + memory.enabled=true: only memory provider wired
  - [ ] Boot with both disabled: ComposedAssembler with zero providers (base prompt only)
  - [ ] Watcher started on boot and stopped on shutdown
  - [ ] Shutdown stops watcher before session manager
- Integration tests:
  - [ ] Daemon boots with bundled skills loaded into registry
  - [ ] Session prompt contains skill catalog when skills enabled
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (full pipeline: fmt, lint, test, build)
- Daemon boots and shuts down cleanly with skills enabled
- No goroutine leaks
