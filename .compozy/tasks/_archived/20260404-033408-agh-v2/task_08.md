---
status: completed
title: CLI Package
type: ""
complexity: medium
dependencies:
    - task_07
---

# Task 08: CLI Package

## Overview

Implement the `internal/cli` package — Cobra-based CLI commands that communicate with the daemon via UDS. Includes all v1 commands (daemon, session, agent, observe, whoami) and output formatters (human, json, toon).

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement all v1 commands:
  - `agh daemon start [--foreground]`, `agh daemon stop`, `agh daemon status`
  - `agh session new --agent <name> [--cwd <dir>] [--name <label>]`
  - `agh session list [--all]`, `agh session stop <id>`, `agh session status <id>`
  - `agh session resume <id>`, `agh session wait <id>`
  - `agh session prompt <id> "message"`
  - `agh session events <id> [--type <type>] [--last <n>] [--since <time>] [--follow]`
  - `agh session history <id>`
  - `agh agent list`, `agh agent info <name>`
  - `agh observe events [--session <s>] [--agent <a>] [--type <t>] [--follow]`
  - `agh observe health`
  - `agh whoami`
- MUST implement global flag: `-o human|json|toon` (default: human)
- MUST implement UDS client that connects to daemon socket
- MUST implement output formatters: human (styled table/text), json (raw), toon (LLM-friendly)
- MUST support `--follow` via SSE over UDS (long-lived connection)
- MUST support `--since` with RFC3339 and relative duration formats (5m, 1h, 24h)
- MUST read `AGH_SESSION_ID`, `AGH_AGENT`, `AGH_AGENT_NAME` env vars for agent identity (whoami)
- MUST implement daemon start with process detachment (background mode)
</requirements>

## Subtasks
- [x] 8.1 Implement root command and global flags (-o output format)
- [x] 8.2 Implement UDS client (HTTP client over unix socket)
- [x] 8.3 Implement daemon commands (start with detachment, stop, status)
- [x] 8.4 Implement session commands (new, list, stop, status, resume, wait, prompt, events, history)
- [x] 8.5 Implement agent commands (list, info)
- [x] 8.6 Implement observe commands (events with --follow, health)
- [x] 8.7 Implement whoami command (reads env vars)
- [x] 8.8 Implement output formatters: human, json, toon
- [x] 8.9 Implement --since time parsing (RFC3339 + relative durations)
- [x] 8.10 Implement --follow via SSE over UDS

## Implementation Details

Create the following files:
- `internal/cli/root.go` — Root command, global flags, output format
- `internal/cli/client.go` — UDS HTTP client
- `internal/cli/daemon.go` — Daemon start/stop/status commands
- `internal/cli/session.go` — Session commands
- `internal/cli/agent.go` — Agent commands
- `internal/cli/observe.go` — Observe commands
- `internal/cli/whoami.go` — Identity command
- `internal/cli/format.go` — Output formatters (human, json, toon)

### Relevant Files
- `.compozy/tasks/agh-v2/_techspec.md` — CLI v1 commands, API endpoints

### Old Project Reference
- `.old_project/internal/cli/root.go` — Cobra command tree setup, global flags
- `.old_project/internal/cli/daemon.go` — Daemon start with process detachment
- `.old_project/internal/cli/human/renderer.go` — Human-readable output formatting
- `.old_project/internal/cli/human/styles.go` — Terminal styling patterns
- `.old_project/internal/toon/renderer.go` — TOON format renderer
- `.old_project/internal/cli/lifecycle.go` — Session resume/wait command patterns
- `.old_project/internal/cli/observe.go` — Observe command structure, query patterns, output formatting

### Related ADRs
- [ADR-007: Background Sessions With CLI Prompt](../adrs/adr-007.md) — CLI command design
- [ADR-009: Agent-First Observability](../adrs/adr-009.md) — Output formats for agents

## Deliverables
- `internal/cli/` package with all v1 commands and formatters
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests against test daemon **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Output formatter: human format produces styled table
  - [x] Output formatter: json format produces valid JSON
  - [x] Output formatter: toon format produces TOON output
  - [x] Time parsing: RFC3339 timestamps parsed correctly
  - [x] Time parsing: relative durations (5m, 1h, 24h) converted correctly
  - [x] Whoami: reads correct env vars
  - [x] Session new: requires --agent flag
  - [x] Session events: --follow flag sets SSE mode
- Integration tests:
  - [x] Full CLI round-trip: daemon start → session new → prompt → events → session stop → daemon stop
  - [x] All output formats produce valid output for session list
  - [x] --follow mode: receives streaming events and exits on disconnect
- Test coverage target: >=80%
- All tests must pass with `-race` flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All v1 commands functional against a running daemon
- Output formats render correctly for all three modes
- `--follow` streams events in real-time
