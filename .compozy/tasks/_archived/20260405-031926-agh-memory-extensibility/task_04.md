---
status: completed
title: Memory Assembler & Daemon Wiring
type: ""
complexity: high
dependencies:
    - task_01
    - task_02
    - task_03
---

# Task 04: Memory Assembler & Daemon Wiring

## Overview

Implement the `PromptAssembler` in `internal/memory/` and wire the entire memory system into the daemon composition root. The assembler loads MEMORY.md indexes from both scopes and concatenates them with the agent's system prompt (frozen snapshot at session start). The daemon wiring initializes the memory Store at boot, creates the dream Service with a SessionSpawner callback, injects the assembler into the session Manager, and starts the periodic dream ticker goroutine. This is the integration hub that connects all memory subsystems.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC "System Architecture > Data Flow" for the full assembly + dream lifecycle
- REFERENCE TECHSPEC "Core Interfaces" for Assembler struct and PromptAssembler implementation
- REFERENCE `.old_project/internal/kernel/kernel.go` for how dream was wired in the old project
- REFERENCE `.resources/claude-code/_prompts/` for how Claude Code renders memory context into prompts
- REFERENCE `.resources/hermes/plugins/memory/` for Hermes's prompt injection ordering
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `Assembler` struct in `internal/memory/assembler.go` that implements `session.PromptAssembler`
- MUST load global MEMORY.md index via `Store.LoadIndex(ScopeGlobal)` during `Assemble()`
- MUST load workspace MEMORY.md index via `Store.LoadIndex(ScopeWorkspace)` using workspace path from args
- MUST render memory context as formatted sections with memory taxonomy instructions, CLI commands, and staleness policy
- MUST concatenate memory context with `AgentDef.Prompt` — memory context BEFORE agent prompt
- MUST return unmodified `AgentDef.Prompt` when both indexes are empty (no memory directories)
- MUST modify `Manager.Create()` to call `assembler.Assemble()` when assembler is non-nil, before `driver.Start()`
- MUST modify `Manager.Create()` to apply `approve-all` permissions when `CreateOpts.Type == SessionTypeDream`
- MUST initialize `memory.Store` in daemon boot with `HomePaths.MemoryDir` and call `EnsureDirs()`
- MUST create `memory.Assembler` and inject into session Manager via `WithPromptAssembler()`
- MUST create `dream.Service` with config-driven options from `MemoryConfig.Dream`
- MUST wire `SessionSpawner` callback in daemon that calls `Manager.Create(SessionTypeDream)` + `Manager.Prompt()` + `Manager.Stop()`
- MUST start periodic dream ticker goroutine with `time.Ticker` using `DreamConfig.CheckInterval`
- MUST use `context.Context` cancellation to stop ticker on daemon shutdown
- MUST track dream ticker goroutine with `sync.WaitGroup` for graceful shutdown
- MUST skip dream check when memory or dream is disabled in config
- MUST log dream gate results and consolidation outcomes via `slog`
</requirements>

## Subtasks
- [x] 4.1 Implement `Assembler` struct with `Assemble()` method and memory context rendering
- [x] 4.2 Modify `Manager.Create()` to call assembler and apply SessionType permission overrides
- [x] 4.3 Add memory Store initialization to daemon boot sequence
- [x] 4.4 Wire dream Service with SessionSpawner callback in daemon
- [x] 4.5 Add periodic dream ticker goroutine with context cancellation and WaitGroup tracking
- [x] 4.6 Add dream check trigger on session stop via Notifier fanout

## Implementation Details

New file:
- `internal/memory/assembler.go` — Assembler struct implementing PromptAssembler

Modified files:
- `internal/session/manager.go` — Call assembler in Create(), apply SessionType permissions
- `internal/daemon/daemon.go` — Boot sequence additions: Store init, Assembler injection, dream Service, ticker, spawner

Reference TechSpec "System Architecture > Data Flow" steps 1 and 3 for the assembly and dream flows. The daemon boot sequence additions should follow the existing pattern (after observer creation, before readiness signal).

### Relevant Files
- `.old_project/internal/kernel/kernel.go` — Dream service initialization at boot and periodic ticker pattern
- `.old_project/internal/prompt/context.go` — Memory context rendering (`RenderMemoryContext`)
- `.resources/claude-code/_prompts/` — Claude Code's prompt assembly and memory section format
- `.resources/hermes/plugins/memory/` — Hermes prompt injection with capacity display
- `internal/memory/store.go` (task_01) — Store.LoadIndex() for index loading
- `internal/memory/dream.go` (task_03) — Service.ShouldRun() and Service.Run()
- `internal/session/manager.go:259-370` — Current Create() method to modify
- `internal/session/manager.go:63-82` — Manager struct to add assembler field
- `internal/daemon/daemon.go:211-321` — New() constructor where wiring happens
- `internal/daemon/daemon.go:323-347` — Run() method where ticker starts

### Dependent Files
- `internal/httpapi/memory.go` (task_05) — Will need Store reference from RuntimeDeps
- `internal/udsapi/memory.go` (task_05) — Will need Store reference from RuntimeDeps

### Related ADRs
- [ADR-002: PromptAssembler Interface in session/](adrs/adr-002.md) — Assembler implements this interface
- [ADR-003: Frozen Snapshot Memory Injection](adrs/adr-003.md) — Assembly happens once at session start

## Deliverables
- `internal/memory/assembler.go` with PromptAssembler implementation
- Modified `internal/session/manager.go` with assembly call and SessionType permissions
- Modified `internal/daemon/daemon.go` with full memory system wiring
- Unit tests for assembler with 80%+ coverage **(REQUIRED)**
- Integration tests for daemon wiring **(REQUIRED)**
- All existing tests continue to pass **(REQUIRED)**

## Tests
- Unit tests (assembler):
  - [x] `Assemble` with global index only returns prompt with global memory section
  - [x] `Assemble` with workspace index only returns prompt with workspace memory section
  - [x] `Assemble` with both indexes returns prompt with both sections
  - [x] `Assemble` with empty indexes returns unmodified agent prompt
  - [x] `Assemble` includes memory taxonomy instructions in rendered context
  - [x] `Assemble` includes `agh memory` CLI command reference in context
  - [x] `Assemble` includes staleness policy in context
  - [x] Memory context is placed BEFORE agent prompt in assembled string
- Unit tests (session manager):
  - [x] `Create` with non-nil assembler calls `Assemble()` with correct agent and workspace
  - [x] `Create` with nil assembler passes raw agent prompt to driver
  - [x] `Create` with `SessionTypeDream` applies approve-all permissions regardless of config
  - [x] `Create` with `SessionTypeUser` uses config-defined permissions (unchanged behavior)
- Unit tests (daemon config):
  - [x] Dream ticker does NOT start when `memory.enabled = false`
  - [x] Dream ticker does NOT start when `memory.dream.enabled = false`
  - [x] Assembler is NOT injected when `memory.enabled = false`
- Integration tests (daemon):
  - [x] Daemon boot initializes memory Store and calls EnsureDirs
  - [x] Daemon boot creates Assembler and injects into session Manager
  - [x] Dream ticker fires at configured interval (use short interval in test)
  - [x] Dream ticker stops on context cancellation (graceful shutdown)
  - [x] SessionSpawner callback creates dream session with correct SessionType
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing (new + existing)
- Test coverage >=80% on new code
- `make verify` passes
- Dream ticker starts on daemon boot and stops on shutdown
- Assembler correctly enriches agent prompts with memory context

## Verification
- `go test ./internal/acp ./internal/session ./internal/daemon`
- `go test -tags integration ./internal/daemon`
- `go test -race -cover ./internal/memory ./internal/session ./internal/daemon ./internal/acp`
- `make verify`
