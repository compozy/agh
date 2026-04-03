---
status: completed
domain: Kernel
type: Feature Implementation
scope: Full
complexity: high
dependencies:
    - task_04
    - task_06
    - task_07
---

# Task 11: Hook System

## Overview
Implement the hook event system that captures agent tool usage, normalizes events from different driver formats into a common HookEvent struct, routes events to the workgroup master, and integrates with the NATS transport for event delivery.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST subscribe to agh.wg.{xid}.hook NATS subject per workgroup per docs/spec-v2/02-kernel.md
- MUST call driver.ParseHookEvent(rawPayload) to normalize events per docs/spec-v2/05-drivers.md
- MUST route normalized HookEvent to workgroup master via agh.wg.{wg}.agent.{master}
- MUST support the hook-event CLI command that publishes raw payloads from hook scripts
- MUST handle hook events from Claude/Codex (file-based hooks), OpenCode (SSE stream), and Pi (extensions)
- MUST normalize tool names to kernel canonical names per docs/spec-v2/05-drivers.md tool mapping table
- MUST log hook events to the events table with type "hook"
- MUST handle the case where master is not yet ready (queue until ready)
</requirements>

## Subtasks
- [x] 11.1 Implement NATS subscription per workgroup for hook events
- [x] 11.2 Implement hook event normalization pipeline (raw → driver.ParseHookEvent → HookEvent)
- [x] 11.3 Implement routing logic to deliver normalized events to workgroup master
- [x] 11.4 Implement hook event logging to SQLite events table
- [x] 11.5 Implement queuing for events when master is not yet ready
- [x] 11.6 Implement the agh hook-event CLI ingestion path

## Implementation Details
Refer to docs/spec-v2/05-drivers.md for the hook event flow diagram and per-driver hook mechanisms. Refer to docs/spec-v2/04-workgroups.md for hook routing rules.

### Relevant Files
- `docs/spec-v2/05-drivers.md` — hook event flow, per-driver mechanisms
- `docs/spec-v2/04-workgroups.md` — hook routing to master
- `docs/spec-v2/02-kernel.md` — NATS hook subject
- `docs/spec-v2/08-data-models.md` — HookEvent struct

### Dependent Files
- `internal/transport/nats.go` — NATS subscriptions
- `internal/pty/` — PTY manager for hook script execution context
- `internal/drivers/claude/` — ParseHookEvent implementation
- `internal/state/writer.go` — event logging
- `internal/registry/` — master agent lookup per workgroup

## Deliverables
- Hook subscription and routing logic (in kernel or dedicated hook package)
- Hook event normalization pipeline
- Hook-event CLI ingestion path
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for hook-to-master flow **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Hook published to agh.wg.{wg}.hook reaches subscription handler
  - [x] ParseHookEvent called on raw payload produces correct HookEvent
  - [x] Normalized event delivered to workgroup master
  - [x] Workers do not receive hook events (only master)
  - [x] Hook events in child workgroup do not reach parent unless escalated
  - [x] Hook event logged to events table with correct type and data
  - [x] Tool names normalized to kernel canonical names
- Integration tests:
  - [x] End-to-end: simulate hook script → agh hook-event CLI → NATS → normalization → master delivery
  - [x] Events queued when master not ready, delivered when master becomes ready
- Test coverage target: >=80%
- All tests must pass with -race flag

## Verification
- `go test -race -cover ./internal/kernel ./internal/cli ./internal/drivers/codex ./internal/drivers/opencode ./internal/drivers/pi`
  - `internal/kernel`: 81.1%
  - `internal/cli`: 81.7%
  - `internal/drivers/codex`: 84.6%
  - `internal/drivers/opencode`: 84.7%
  - `internal/drivers/pi`: 87.5%
- `make verify`

## Success Criteria
- All tests passing
- Test coverage >=80%
- `make verify` passes
- Hook events correctly routed to workgroup master regardless of driver
