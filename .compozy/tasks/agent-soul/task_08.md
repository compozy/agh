---
status: completed
title: Implement Managed Heartbeat Authoring and Status Services
type: backend
complexity: high
dependencies:
  - task_05
  - task_06
  - task_07
---

# Task 08: Implement Managed Heartbeat Authoring and Status Services

## Overview

Implement the internal services for managed `HEARTBEAT.md` validation, mutation, history, rollback, and status reads. This task creates transport-agnostic behavior for Heartbeat policy management while consuming session health as runtime data rather than authored content.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_heartbeat.md`, and ADR-007 through ADR-011 before implementation.
- REFERENCE TECHSPEC for CAS, service contracts, status payloads, diagnostics, and no-refresh semantics.
- FOCUS ON WHAT must be guaranteed: managed authoring, policy validation, revision history, status, and separation from session health.
- MINIMIZE CODE in task notes; expose no HTTP/UDS/CLI route in this task.
- TESTS REQUIRED for CAS conflicts, invalid policy, rollback, delete, status, and session-health composition.
- NO WORKAROUNDS: writes must not wake sessions, refresh sessions, create tasks, or mutate scheduler state.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST implement a transport-agnostic Heartbeat authoring service for validate, write, delete, history, and rollback.
- MUST implement status/inspect service behavior that combines latest valid policy, diagnostics, config digest, wake state, and session health where requested.
- MUST require body-level `expected_digest` for mutating managed APIs; do not use HTTP `If-Match` as the primary contract.
- MUST append revision rows for successful mutations and persist resolved snapshots.
- MUST return closed, redacted errors for invalid content, stale digest, missing agent, path errors, and unsupported health states.
- MUST avoid session refresh or wake scheduling as a side effect of authoring.
</requirements>

## Subtasks
- [x] 8.1 Define Heartbeat authoring and status service interfaces and DTO-adjacent models.
- [x] 8.2 Implement validate, write, delete, history, and rollback with atomic file operations.
- [x] 8.3 Implement inspect/status reads combining policy, diagnostics, config provenance, wake state, and session health.
- [x] 8.4 Persist revisions and latest valid snapshots through task_06 storage.
- [x] 8.5 Add service tests for all mutation, status, and failure paths.
- [x] 8.6 Confirm authoring does not schedule wakes, prompt sessions, create tasks, or renew leases.

## Implementation Details

Keep the service boundary transport-neutral so HTTP, UDS, CLI, Host API, tools, and SDKs all share the same behavior. Status reads may include session health, but authored policy must never become the source of liveness.

### Relevant Files
- `internal/heartbeat/authoring.go` - likely service boundary for managed mutations.
- `internal/heartbeat/status.go` - likely status/inspect composition boundary.
- `internal/heartbeat/resolver.go` - validation and digest source from task_05.
- `internal/store/globaldb/` - snapshots, revisions, session health, and wake audit from task_06.
- `internal/session/` - session health read model from task_07.
- `internal/fileutil/` or equivalent - atomic writes and path safety.
- `internal/diagnostics/` - closed diagnostics and redaction.

### Dependent Files
- `internal/heartbeat/*_test.go` - service mutation, status, and diagnostics coverage.
- `internal/store/globaldb/*_test.go` - revision and snapshot persistence tests.
- `internal/session/*_test.go` - health composition edge cases if needed.
- `.compozy/tasks/agent-soul/task_09.md` - scheduler wake service consumes latest valid policy/status data.
- `.compozy/tasks/agent-soul/task_11.md` - transports adapt to these services.
- `.compozy/tasks/agent-soul/task_12.md` - CLI commands consume these services through routes/clients.

### Related ADRs
- [ADR-007: HEARTBEAT.md Is Advisory Wake Policy](adrs/adr-007.md) - defines what authoring may control.
- [ADR-008: Heartbeat Snapshots and Wake Audit](adrs/adr-008.md) - defines revision and status persistence.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - defines manageability requirements.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - defines config-bound status.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: this service is the only Heartbeat mutation authority for Host API actions, tools/resources, bridge SDKs, and future bundles.
- Agent manageability: service models must be stable for CLI JSON, HTTP, UDS, and Host API consumers.
- Config lifecycle: service must report effective `[agents.heartbeat]` config digest and reject policy outside configured bounds.

### Web/Docs Impact
- Web impact: generated type impact starts in task_10; no UI editor is included in MVP.
- Docs impact: task_15 must document managed Heartbeat authoring, CAS, status, and separation from session health.

## Deliverables
- Transport-agnostic managed Heartbeat authoring service.
- Heartbeat status/inspect service combining policy, wake audit/state, and session health safely.
- Atomic write/delete/history/rollback behavior with CAS and revision persistence.
- Tests for mutation, status, diagnostics, and no side effects.
- Completion evidence that no wake, prompt, task, or lease side effects occur during authoring.

## Tests
- Unit tests:
  - [x] Valid write creates/updates `HEARTBEAT.md`, appends a revision, and returns the new digest.
  - [x] Invalid policy fails closed without file mutation or successful revision rows.
  - [x] Stale `expected_digest` returns a deterministic conflict.
  - [x] Delete and rollback use the same validation and revision paths as write.
  - [x] Status reports policy digest, config digest, diagnostics, wake state, and session health without leaking raw data.
  - [x] Authoring does not enqueue wake events or update session health.
- Integration tests:
  - [x] Heartbeat authoring persists and reloads revisions across DB reopen.
  - [x] Status composition handles missing policy, invalid policy, stale session health, and disabled config states.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec_heartbeat.md` - managed authoring and status requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - authored Heartbeat policy precedent.
- `.compozy/tasks/agent-soul/analysis/analysis_paperclip_heartbeat.md` - wake/run status contrast.
- `.resources/openclaw/docs/gateway/heartbeat.md:14-17` - advisory wake policy precedent.
- `.resources/paperclip/server/src/onboarding-assets/ceo/HEARTBEAT.md:1-85` - rich policy file precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Heartbeat policy can be managed safely through one internal service.
- Status combines runtime health and authored policy without confusing their authorities.
