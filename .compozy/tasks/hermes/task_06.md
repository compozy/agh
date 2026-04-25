---
status: completed
title: Tool Process Registry and Interrupts
type: backend
complexity: critical
dependencies:
  - task_01
---

# Task 06: Tool Process Registry and Interrupts

## Overview

Introduce a shared runtime process registry and scoped interrupt model for long-running tool processes. This task records subprocess ownership durably, validates recovered PIDs with start-time evidence, reconciles state on boot, and routes interrupts to the correct thread, tool, or subprocess without broad session-level termination.

<critical>
- ALWAYS READ `_techspec.md`, ADR-001, ADR-004, and task_01 outputs before changing subprocess ownership
- DO NOT put the registry inside `session.Manager` or `environment.ToolHost`; use a shared runtime package
- DO NOT kill by PID alone; validate ownership and process start time before signaling recovered processes
- DO NOT introduce fire-and-forget goroutines for process watching
- Interrupts must be scoped and auditable, not a blanket session reset
</critical>

<requirements>
- MUST add a shared process registry package for tool subprocess records
- MUST persist registry checkpoints on process start, update, and completion
- MUST validate PID and start-time evidence during boot reconciliation
- MUST expose scoped interrupt APIs for thread, tool call, and process-level cancellation
- MUST integrate registry and interrupts with ACP, environment tools, hooks, extensions, and subprocess helpers
- MUST analyze and implement required `web/` and `packages/site` follow-up changes caused by this task
</requirements>

## Subtasks
- [x] 6.1 Design `internal/toolruntime` registry and interrupt interfaces consumed by existing subprocess owners
- [x] 6.2 Add durable process records with checkpoint-on-write and boot reconciliation
- [x] 6.3 Implement PID/start-time validation and stale-record cleanup
- [x] 6.4 Wire scoped interrupts into ACP tool host, environment tools, hooks, extensions, and subprocess helpers
- [x] 6.5 Add tests for ownership validation, restart reconciliation, scoped cancellation, and stale PID safety
- [x] 6.6 Analyze and implement any required follow-up changes in `web/` and `packages/site`, including documentation, typed clients, settings pages, examples, stories, and tests where applicable

## Implementation Details

The registry should be a small runtime package imported by owners that launch tool subprocesses. It should not know about daemon composition, but it must expose enough state for daemon boot reconciliation and interrupt routing. Keep process IDs, command metadata, owner IDs, start times, and completion state bounded and redacted.

### Relevant Files
- `internal/toolruntime/` - new shared process registry and interrupt package
- `internal/acp/launcher.go` - ACP subprocess launch integration
- `internal/acp/launcher_tool_host.go` - ACP tool-host process ownership
- `internal/acp/handlers.go` - interrupt and tool-call handling
- `internal/environment/types.go` - environment tool process ownership
- `internal/hooks/executor_subprocess.go` - hook subprocess integration
- `internal/extension/host_api.go` - extension host process integration
- `internal/subprocess/` - shared subprocess helpers
- `internal/procutil/` - PID and start-time validation helpers

### Dependent Files
- `internal/toolruntime/*_test.go` - registry, checkpoint, and interrupt tests
- `internal/acp/*_test.go` - ACP integration tests for process ownership and interrupt routing
- `internal/hooks/*_test.go` - hook subprocess cancellation tests
- `internal/extension/*_test.go` - extension process ownership tests if impacted
- `web/src/systems/session/` - interrupt UI or typed event updates if surfaced
- `packages/site/` - docs for interrupts and long-running tool process behavior
- `.compozy/tasks/hermes/task_10.md` - QA plan must include restart and stale PID scenarios

### Related ADRs
- [ADR-001: Hermes Hardening Tracks](adrs/adr-001-hermes-hardening-tracks.md) - identifies process registry and interrupts as selected hardening work
- [ADR-004: Shared Process Registry and Interrupt Runtime](adrs/adr-004-shared-process-registry-and-interrupt-runtime.md) - defines package placement and scoped interrupt decisions

## Deliverables
- Shared process registry and scoped interrupt package
- Durable checkpoint-on-write process records
- Boot reconciliation with PID/start-time validation
- Integrated interrupt routing for ACP, tools, hooks, and extensions
- Tests proving scoped cancellation and stale PID safety
- Documented `web/` and `packages/site` impact assessment with required changes applied or explicitly marked not applicable

## Tests
- Unit tests:
  - [x] Registry writes records on start/update/completion
  - [x] Recovered PID validation rejects mismatched start-time evidence
  - [x] Scoped interrupts signal only the requested owner and process scope
  - [x] Stale records are reconciled without killing unrelated host processes
- Integration tests:
  - [x] A long-running tool process can be interrupted without resetting the full session
  - [x] Boot reconciliation recovers active owned processes and retires stale records
  - [x] ACP, hook, and extension subprocess owners use the same registry contract
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- Long-running tool processes are durably tracked and recoverable
- Interrupts are scoped, observable, and safe across process reuse
- No owner keeps a private incompatible subprocess registry for selected paths
- Affected backend, web, and docs tests pass
