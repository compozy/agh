---
status: pending
domain: CLI
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
  - task_04
---

# Task 05: CLI Memory Commands

## Overview

Implement the `agh memory` CLI command group with 5 subcommands (list, read, write, delete, consolidate), daemon client methods for the memory HTTP API, and dual human/TOON output renderers. This gives both human users and AI agents a complete interface for managing persistent memories.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement `agh memory list [--scope global|workspace]` — list memory headers with name, type, age, description
- MUST implement `agh memory read <filename> [--scope global|workspace]` — read full memory file content
- MUST implement `agh memory write <filename> --type <type> --description <desc> [--scope global|workspace]` — write memory with content from stdin or `--content` flag
- MUST implement `agh memory delete <filename> [--scope global|workspace]` — delete memory file
- MUST implement `agh memory consolidate` — manually trigger dream consolidation (bypasses time/session gates)
- Scope defaults MUST follow memory type: `user`/`feedback` → global, `project`/`reference` → workspace
- MUST add daemon client methods: `ListMemories()`, `ReadMemory()`, `WriteMemory()`, `DeleteMemory()`, `TriggerConsolidation()`
- MUST implement human renderers for memory list (styled table) and memory content (key-value section)
- MUST implement TOON output for all commands
- MUST follow existing `daemonCommandDeps` dependency injection pattern
- MUST use `writeOutput(cmd, humanFn, toonFn)` for dual output
- MUST register `memory` command in root command tree
</requirements>

## Subtasks

- [x] 5.1 Add daemon client methods for all 5 memory API endpoints
- [x] 5.2 Implement `newMemoryCommand()` parent command with 5 subcommands
- [x] 5.3 Implement `agh memory list` with scope flag and dual output
- [x] 5.4 Implement `agh memory read` with filename arg, scope flag, and dual output
- [x] 5.5 Implement `agh memory write` with filename, type, description, content/stdin, scope
- [x] 5.6 Implement `agh memory delete` with filename, scope, and confirmation
- [x] 5.7 Implement `agh memory consolidate` with status output
- [x] 5.8 Add human renderers (`RenderMemoryList`, `RenderMemoryContent`) to `internal/cli/human/`
- [x] 5.9 Register `memory` command in `root.go`
- [x] 5.10 Write CLI tests with mock daemon client

## Implementation Details

Create and modify files:

- `internal/cli/memory.go` (new) — `newMemoryCommand()` with all 5 subcommands
- `internal/cli/memory_test.go` (new) — CLI tests with mock daemon client
- `internal/cli/root.go` — Register `cmd.AddCommand(newMemoryCommand(deps))`
- `internal/cli/daemon.go` — Add memory methods to `udsHTTPDaemonClient` and `daemonAPIClient` interface
- `internal/cli/human/renderer.go` — Add `RenderMemoryList()` and `RenderMemoryContent()`

### Relevant Files

- `internal/cli/state.go:18-133` — Reference for subcommand group pattern (`newStateCommand` → `newStateReadCommand` / `newStateAppendCommand`)
- `internal/cli/daemon.go` — `daemonAPIClient` interface and `udsHTTPDaemonClient` implementation
- `internal/cli/output.go:84-116` — `writeOutput()` dual output pattern
- `internal/cli/human/renderer.go` — Existing human renderers (`RenderBlackboard`, `RenderEvents`, etc.)
- `internal/cli/human/styles.go` — Lipgloss style constants
- `internal/cli/root.go` — Root command tree where `memory` command is registered
- `internal/toon/` — TOON renderer for agent-friendly output

### Dependent Files

- None — this is the leaf task

### Related ADRs

- [ADR-001: Dual Storage](../adrs/adr-001.md) — Scope flag behavior and defaults
- [ADR-005: Index-Only Prompt Injection](../adrs/adr-005.md) — Agents use `agh memory read` for full content

## Deliverables

- `agh memory` command group with 5 functional subcommands
- Daemon client methods for all memory API endpoints
- Human renderers for memory list (table) and content (key-value)
- TOON output for all commands
- Registration in root command tree
- CLI tests with mock daemon client and 80%+ coverage **(REQUIRED)**

## Tests

- Unit tests:
  - [x] `agh memory list` with default scope returns all memories
  - [x] `agh memory list --scope global` returns only global memories
  - [x] `agh memory list --scope workspace` returns only workspace memories
  - [x] `agh memory read <filename>` returns memory content
  - [x] `agh memory read` with missing filename returns usage error
  - [x] `agh memory write <filename> --type user --description "desc" --content "body"` creates memory
  - [x] `agh memory write` with missing required flags returns usage error
  - [x] `agh memory write` with invalid type returns validation error
  - [x] `agh memory write` scope defaults to global for user/feedback types
  - [x] `agh memory write` scope defaults to workspace for project/reference types
  - [x] `agh memory delete <filename>` deletes memory
  - [x] `agh memory delete` with missing filename returns usage error
  - [x] `agh memory consolidate` triggers consolidation and returns status
  - [x] Human output: `RenderMemoryList` produces styled table with name, type, age, description columns
  - [x] Human output: `RenderMemoryContent` produces key-value section with metadata + content
  - [x] TOON output: all commands produce valid TOON format
  - [x] Daemon client methods call correct HTTP endpoints with correct parameters
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria

- All tests passing
- Test coverage >=80%
- `make verify` passes
- All 5 subcommands functional with both human and TOON output
- Scope defaults work correctly based on memory type
- Commands follow existing CLI patterns (dependency injection, dual output, error handling)
