---
status: completed
title: Agent Task Lease API And CLI Verbs
type: backend
complexity: high
dependencies:
  - task_05
  - task_08
---

# Task 09: Agent Task Lease API And CLI Verbs

## Overview
Expose task-run claiming and lease maintenance to agents through first-class UDS endpoints and CLI verbs. Agents should claim, receive coordination channel metadata, heartbeat, complete, fail, and release work without operator-only commands or private daemon calls.

<critical>
- ALWAYS READ `_techspec.md`, ADR-002, ADR-003, ADR-010, ADR-011, and ADR-012 before adding agent task verbs
- DO NOT EXPOSE RAW CLAIM TOKENS THROUGH LIST/GET/LOG OUTPUT - only the immediate claim response may contain the raw token
- AGENT TASK VERBS MUST INFER CALLER IDENTITY from task_05 and must preserve operator task commands
- TESTS REQUIRED - CLI, UDS handler, token redaction, permission denial, and restart/reconnect flows must be covered
- NO WORKAROUNDS - do not shell out to existing commands or bypass the task service to avoid API design
</critical>

<requirements>
- MUST add agent-facing UDS endpoints for next claim, heartbeat, release, complete, and fail.
- MUST add CLI verbs under existing task command structure for `next`, `heartbeat`, `complete`, `fail`, and `release` with stable JSON output.
- MUST include `coordination_channel_id` and channel display metadata in claim responses for channel-bound coordinated runs.
- MUST require `task.create` or relevant task lease permissions where the TechSpec requires them.
- MUST infer claimer session/workspace/capabilities from caller identity and situation context.
- MUST preserve existing operator `agh task ...` flows and manual start/approval commands.
- MUST update generated contracts and web types if any public API payloads are exposed beyond UDS.
</requirements>

## Subtasks
- [x] 9.1 Add agent task UDS routes and handlers for claim-next and lease mutations.
- [x] 9.2 Add CLI client methods and `agh task next|heartbeat|complete|fail|release` verbs with JSON output.
- [x] 9.3 Wire caller identity, permissions, capability criteria, and structured audit actor fields.
- [x] 9.4 Enforce claim-token redaction in command errors, logs, and non-claim responses.
- [x] 9.5 Add handler/CLI tests for success, coordination channel metadata, stale token, denied caller, malformed payload, and no-work-found cases.
- [x] 9.6 Run generated contract/web checks if HTTP/OpenAPI-facing DTOs change.

## Implementation Details
The CLI should be thin over the UDS API. Do not make the CLI a second implementation of claim rules. Command output must be machine-readable for agents and stable enough for prompts and scripts.

Claim-next should return enough context for an agent to execute the task: run ID, task identity, instructions/metadata needed by the current task model, coordination channel metadata, lease deadline, raw claim token, and redacted audit metadata. The channel metadata tells the worker where to communicate; it does not grant ownership or replace task API transitions.

### Relevant Files
- `internal/api/udsapi/routes.go` - route registration.
- `internal/api/udsapi/task*.go` - task handler precedent and new agent endpoints.
- `internal/api/core/interfaces.go` - task service interface additions consumed by API handlers.
- `internal/cli/task.go` - existing operator task commands and new agent verbs.
- `internal/cli/client.go` - UDS client methods and output mapping.
- `internal/task/manager.go` - claim/lease service from task_08.
- `internal/task/actors.go` - actor/caller audit mapping.
- `.resources/paperclip/cli/src/commands/heartbeat-run.ts` - reference for heartbeat CLI semantics.
- `.resources/claude-code/tasks.ts` - reference for task command ergonomics and machine-readable task state.

### Dependent Files
- `internal/daemon/task_runtime.go` - task_10/task_11 use enqueue and coordinator trigger behavior.
- `packages/site/content/runtime/cli-reference/` - task_16 documents the new commands.
- `.compozy/tasks/autonomous/qa/test-cases/` - task_17 plans CLI verification.

### Related ADRs
- [ADR-002: Agent-Facing CLI Before Built-In MCP Tools](adrs/adr-002.md) - command naming and agent-first surface.
- [ADR-003: Task Run Claim Lease Model](adrs/adr-003.md) - token and lease rules.
- [ADR-010: Manual Operator Control Remains First-Class](adrs/adr-010.md) - existing operator task flows remain.
- [ADR-011: Generated Contract And Runtime Docs Co-Ship](adrs/adr-011.md) - generated type discipline.
- [ADR-012: Task-Run Coordination Channels](adrs/adr-012.md) - claim response channel metadata.

## Deliverables
- Agent UDS endpoints for task claim and lease lifecycle.
- Claim responses that include coordination channel metadata for coordinated runs.
- CLI verbs for claim, heartbeat, complete, fail, and release.
- Permission, identity, and audit wiring for agent task access.
- Unit tests with 80%+ coverage for handlers and command helpers **(REQUIRED)**.
- UDS/CLI integration tests for real lease lifecycle flows **(REQUIRED)**.

## Tests
- Unit tests:
  - [x] Handler rejects unauthenticated callers and callers without task lease permission.
  - [x] CLI payload parsing validates token/run IDs and produces stable JSON errors.
  - [x] Claim response includes raw token once; heartbeat/complete/fail/release responses do not echo it.
  - [x] Claim response includes `coordination_channel_id` for coordinated runs and omits it cleanly for non-coordinated runs.
  - [x] No-work-found returns a structured non-error or documented exit code suitable for agent loops.
  - [x] Existing operator task commands continue to parse and call the operator service paths.
- Integration tests:
  - [x] A session claims a queued run, heartbeats it, completes it, and cannot complete it again with the old token.
  - [x] A worker can use the claim response channel ID for `agh ch send --kind status` while completion still requires `agh task complete`.
  - [x] A stale token cannot release or fail a run after lease recovery.
  - [x] CLI commands work over UDS using agent identity environment from task_05.
  - [x] Manual user-started and agent-created task runs both flow through the same API.
  - [x] Generated OpenAPI/web typecheck/web tests pass if public DTOs change.
- Test coverage target: >=80%.
- All tests must pass.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Agents can use first-class task commands instead of private daemon internals.
- Manual task and operator commands remain intact.
