---
status: completed
domain: CLI
type: Feature Implementation
scope: Full
complexity: medium
dependencies:
    - task_14
    - task_15
---

# Task 16: Daemon & Session CLI

## Overview
Implement the Cobra CLI commands for daemon management (agh start, agh status, agh stop) and session management (agh session start, agh session list, agh session stop, agh session status, agh session resume). The daemon commands control the kernel process lifecycle, while session commands interact with the running kernel via UDS to create and manage isolated sessions.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE docs/plans/2026-03-30-multi-session-design.md for CLI interface spec
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement agh start: acquire daemon.lock at ~/.agh/daemon.lock, write daemon.json at ~/.agh/daemon.json, create daemon.sock at ~/.agh/daemon.sock, boot the kernel daemon process per docs/plans/2026-03-30-multi-session-design.md Daemon Commands section
- MUST implement agh status: show kernel state + session summary (active sessions, total agents, uptime)
- MUST implement agh stop: stop kernel + all sessions gracefully with cascading shutdown, release daemon.lock, remove daemon.json, remove daemon.sock
- MUST implement agh session start with goal argument, optional --name flag, and optional --dir flag (workspace override) per docs/plans/2026-03-30-multi-session-design.md Session Commands section
- MUST implement agh session list: list active sessions (name, state, agent count, uptime)
- MUST implement agh session list --all: list active + historical sessions (include stopped sessions from disk)
- MUST implement agh session stop with session name argument
- MUST implement agh session status with session name argument: show session state, agents, workgroups
- MUST implement agh session resume with session name argument: recreate session from persisted state
- MUST handle error "kernel not running. Run 'agh start' first." when kernel is not running
- MUST handle error "max sessions reached (N/N). Stop a session first." when limit exceeded
- MUST implement daemon discovery: check ~/.agh/daemon.json exists, verify flock on daemon.lock, connect to daemon.sock
- MUST use HTTP over UDS connection (~/.agh/daemon.sock) for all daemon/session communication (Gin client)
- MUST render output in TOON format where appropriate
</requirements>

## Subtasks
- [x] 16.1 Implement agh start command (acquire daemon.lock, write daemon.json, create daemon.sock at ~/.agh/, boot kernel daemon, block until signal or stop)
- [x] 16.2 Implement agh status command (kernel state, active sessions, total agents, uptime)
- [x] 16.3 Implement agh stop command (graceful shutdown, release daemon.lock, remove daemon.json + daemon.sock)
- [x] 16.4 Implement agh session start command with goal argument, --name flag, and --dir flag (CWD capture)
- [x] 16.5 Implement agh session list command with --all flag for historical sessions
- [x] 16.6 Implement agh session stop command with session name argument
- [x] 16.7 Implement agh session status command (session state, agents, workgroups)
- [x] 16.8 Implement agh session resume command (restore from persisted state)
- [x] 16.9 Implement error handling for "kernel not running" and "max sessions reached"
- [x] 16.10 Implement daemon discovery: check daemon.json, verify flock, connect via HTTP over UDS

## Implementation Details
Refer to docs/plans/2026-03-30-multi-session-design.md for CLI interface, error handling examples, and session naming rules.

### Relevant Files
- `docs/plans/2026-03-30-multi-session-design.md` — daemon/session CLI interface
- `docs/spec-v2/06-cli.md` — existing CLI reference (for output format consistency)

### Dependent Files
- `internal/kernel/kernel.go` — kernel boot/shutdown (from task_14)
- `internal/session/manager.go` — SessionManager (from task_15)
- `internal/transport/uds.go` — UDS connection for CLI-to-kernel communication
- `internal/toon/` — TOON rendering for output

## Deliverables
- internal/cli/daemon.go — agh start, agh status, agh stop commands
- internal/cli/session.go — agh session start, list, stop, status, resume commands
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for daemon and session commands **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Start command acquires daemon.lock, writes daemon.json, creates daemon.sock at ~/.agh/, boots kernel and blocks
  - [x] Status command outputs kernel state, session count, total agents, uptime
  - [x] Stop command sends shutdown signal, releases daemon.lock, removes daemon.json + daemon.sock
  - [x] Session start with goal creates session with auto-slug name, captures CWD as workspace
  - [x] Session start with --name flag uses provided name
  - [x] Session start with --dir flag uses provided directory as workspace
  - [x] Session start when kernel not running returns correct error message
  - [x] Session start when max sessions reached returns correct error message
  - [x] Session list shows active sessions with name, state, agent count
  - [x] Session list --all includes stopped sessions from disk
  - [x] Session stop sends stop command for specific session
  - [x] Session status shows session state, agents, workgroups
  - [x] Session resume sends resume command with session name
  - [x] Daemon discovery: daemon.json exists + flock verified = daemon running
  - [x] Daemon discovery: missing daemon.json = daemon not running
  - [x] All CLI communication uses HTTP over UDS (~/.agh/daemon.sock)
- Integration tests:
  - [x] Start -> session start -> session list shows session -> session stop -> session list empty -> stop
  - [x] Start -> session start "Build API" creates session named "build-api"
  - [x] Start -> session start --name api "Build" creates session named "api"
  - [x] Start -> session start --dir /some/path "Build" captures correct workspace
  - [x] Session start without running kernel returns "kernel not running" error
  - [x] Multiple sessions created and listed correctly
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Daemon commands correctly manage kernel lifecycle (daemon.lock, daemon.json, daemon.sock at ~/.agh/)
- Session commands correctly manage session lifecycle via HTTP over UDS (~/.agh/daemon.sock)
- Daemon discovery correctly detects running/stopped daemon
- Error messages match docs/plans/2026-03-30-multi-session-design.md examples
- All output rendered in TOON format where applicable
