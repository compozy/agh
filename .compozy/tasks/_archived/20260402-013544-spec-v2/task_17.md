---
status: completed
title: CLI Commands — Session & Runtime
type: ""
complexity: medium
dependencies:
    - task_04
    - task_05
    - task_06
    - task_07
    - task_14
    - task_15
    - task_16
---

# Task 17: CLI Commands — Session & Runtime

## Overview
Implement the Cobra CLI subcommands for within-session runtime operations (spawn, kill, ps, whoami, attach, dashboard). Each command is a thin HTTP client that connects to the kernel via HTTP over UDS (~/.agh/daemon.sock), sends a request, and renders the response. Session context is resolved via COLLAB_SESSION env var or --session flag.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST implement agh spawn with --role, --name, --workgroup, --session flags
- MUST implement agh kill with agent-id argument
- MUST implement agh ps with --verbose flag, TOON output (list agents in session)
- MUST implement agh whoami with TOON output
- MUST implement agh attach with agent-id or --name, read-only PTY streaming to terminal
- MUST implement agh dashboard (print dashboard URL and status)
- MUST use HTTP over UDS connection (~/.agh/daemon.sock) for all kernel communication
- MUST resolve session context via COLLAB_SESSION env var or --session flag
- MUST read AGI_AGENT, AGI_SESSION, AGI_WORKGROUP, AGI_SOCKET env vars per docs/spec-v2/06-cli.md
- NOTE: Daemon-level commands (agh init, agh start, agh stop) belong in task_16
</requirements>

## Subtasks
- [x] 17.1 Implement CLI HTTP-over-UDS connection helper (connect to ~/.agh/daemon.sock, send HTTP request, wait for response)
- [x] 17.2 Implement runtime commands: spawn, kill, ps, whoami
- [x] 17.3 Implement attach command with PTY output streaming to terminal
- [x] 17.4 Implement dashboard command (URL display)
- [x] 17.5 Set up session context resolution (COLLAB_SESSION env var or --session flag)
- [x] 17.6 Set up environment variable reading for agent identity

## Implementation Details
Refer to docs/spec-v2/06-cli.md for all command specs, flags, arguments, and output formats.

### Relevant Files
- `docs/spec-v2/06-cli.md` — complete CLI reference
- `docs/spec-v2/01-architecture.md` — CLI-to-kernel communication flow

### Dependent Files
- `internal/transport/uds.go` — HTTP over UDS connection (~/.agh/daemon.sock)
- `internal/pty/` — PTY output streaming for agh attach
- `internal/toon/` — TOON rendering for output

## Deliverables
- internal/cli/root.go — Cobra root command with UDS connection and session context resolution
- internal/cli/runtime.go — spawn, kill, ps, whoami commands
- internal/cli/attach.go — attach command with PTY streaming
- internal/cli/dashboard.go — dashboard command
- Unit tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Spawn command builds correct HTTP request with role, name, workgroup, session (sent via UDS)
  - [x] Kill command validates agent-id argument
  - [x] Ps renders TOON output with correct fields
  - [x] Whoami reads env vars and displays agent identity
  - [x] Dashboard command outputs URL and status
  - [x] Session context resolved from COLLAB_SESSION env var
  - [x] Session context resolved from --session flag (overrides env var)
  - [x] Environment variables correctly parsed (AGI_AGENT, AGI_SESSION, etc.)
- Integration tests:
  - [x] Spawn via CLI creates agent in kernel registry
- Test coverage target: >=80%
- All tests must pass with -race flag

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- All commands match docs/spec-v2/06-cli.md spec
