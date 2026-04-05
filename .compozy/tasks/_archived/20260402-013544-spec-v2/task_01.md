---
status: completed
title: Project Scaffolding & Core Types
type: ""
complexity: low
dependencies: []
---

# Task 1: Project Scaffolding & Core Types

## Overview
Set up the project directory structure, rename the CLI entry point from `cmd/agi` to `cmd/agh`, configure Cobra as the root command framework, add all required Go dependencies, and define all core Go types and interfaces that every subsequent task imports.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rename cmd/agi/ to cmd/agh/ with updated binary name
- MUST create the full directory tree per docs/spec-v2/12-development-sequence.md
- MUST set up spf13/cobra root command with version subcommand
- MUST add all Go dependencies via `go get` (never hand-edit go.mod)
- MUST define the AgentDriver interface with all 8 methods per docs/spec-v2/05-drivers.md
- MUST define all supporting types: StartOpts, AgentProcess, HookEvent, HookConfig, AgentHealth, ToolMapping, Message, RingBuffer struct signatures
- MUST define all config structs: Config, LimitsConfig, RuntimeConfig, BootstrapAgentConfig, DriverConfig, MetaConfig, DashboardConfig per docs/spec-v2/08-data-models.md
- MUST define all registry types: AgentInfo, WorkgroupInfo, RoleConfig per docs/spec-v2/08-data-models.md
- MUST define WebSocket message types: WsMessage, WsMessageType, PtyOutputPayload, TopologyUpdatePayload, TopologySnapshot, WorkgroupView, AgentView
- MUST include compile-time interface verification: var _ AgentDriver = (*Type)(nil) for future driver implementations
- MUST pass `make verify` (fmt + lint + test + build)
</requirements>

## Subtasks
- [x] 1.1 Rename cmd/agi to cmd/agh with Cobra root command and version subcommand
- [x] 1.2 Create the full internal/ directory tree (kernel, transport, state, registry, pty, dashboard, drivers/claude/codex/opencode/pi, cli, prompt, toon, config)
- [x] 1.3 Add all required Go dependencies via `go get` (nats-server, nats.go, modernc.org/sqlite, cobra, suture, gobreaker, xid, oklog/run, creack/pty, nhooyr.io/websocket, toon-go)
- [x] 1.4 Define AgentDriver interface and all supporting types in internal/kernel/types.go
- [x] 1.5 Define config structs, registry types, and WebSocket types
- [x] 1.6 Add compile-time interface checks and basic constructor/validation tests

## Implementation Details
Refer to docs/spec-v2/08-data-models.md for complete type definitions. Refer to docs/spec-v2/12-development-sequence.md for directory structure.

### Relevant Files
- `cmd/agi/main.go` — current entry point, rename to cmd/agh/main.go
- `go.mod` — add dependencies via go get
- `docs/spec-v2/08-data-models.md` — all type definitions
- `docs/spec-v2/12-development-sequence.md` — directory tree and build order
- `docs/spec-v2/05-drivers.md` — AgentDriver interface

### Dependent Files
- `internal/logger/logger.go` — keep as-is, used by kernel
- `internal/version/version.go` — keep as-is, used by CLI
- `magefile.go` — may need update if binary name changes

## Deliverables
- Renamed cmd/agh/ with working Cobra root command
- Full directory tree with placeholder .go files (package declarations)
- All core types defined in internal/kernel/types.go
- All config structs defined in internal/config/
- go.mod with all required dependencies
- Unit tests with 80%+ coverage **(REQUIRED)**
- `make verify` passes

## Tests
- Unit tests:
  - [x] Compile-time interface verification for AgentDriver
  - [x] Constructor tests for StartOpts, AgentProcess, HookEvent with valid/invalid inputs
  - [x] Config struct validation (required fields, valid ranges)
  - [x] XID generation produces valid 20-char IDs with correct prefixes
  - [x] Cobra root command executes without error
  - [x] Version subcommand outputs version string
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes (fmt + lint + test + build)
- `go build ./cmd/agh/...` produces a working binary
- All types from docs/spec-v2/08-data-models.md are defined
- No hand-edited go.mod entries
