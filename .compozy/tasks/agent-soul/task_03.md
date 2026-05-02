---
status: completed
title: Implement Managed Soul Authoring Service
type: backend
complexity: high
dependencies:
  - task_01
  - task_02
---

# Task 03: Implement Managed Soul Authoring Service

## Overview

Implement the internal service that validates and mutates `SOUL.md` through managed AGH paths. This task makes write/delete/history/rollback safe and deterministic before the service is exposed through CLI, HTTP, UDS, or extensions.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, and ADR-006 before changing authoring behavior.
- REFERENCE TECHSPEC for mutation semantics, CAS fields, atomic write rules, and diagnostics.
- FOCUS ON WHAT must be guaranteed: managed writes, validation, `expected_digest`, revision history, rollback, and no side-channel prompt mutation.
- MINIMIZE CODE in task notes; implement through existing file and store abstractions.
- TESTS REQUIRED for CAS conflicts, invalid content, rollback, delete, atomicity, and revision rows.
- NO WORKAROUNDS: direct unmanaged writes are not a valid authoring path for AGH-managed surfaces.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST implement an internal `SoulAuthoringService` or equivalent service boundary.
- MUST validate every write/delete/rollback through the resolver from task_01.
- MUST require body-level `expected_digest` for mutating managed APIs; do not implement HTTP `If-Match` as the primary contract.
- MUST write files atomically and append a revision row for every successful mutation.
- MUST return deterministic redacted errors for CAS conflicts, invalid content, permission/path errors, and missing agents.
</requirements>

## Subtasks
- [x] 3.1 Define the authoring service interface and request/response models used by later transports.
- [x] 3.2 Implement validate, write, delete, history, and rollback operations.
- [x] 3.3 Add atomic file writes and managed-path safety checks.
- [x] 3.4 Persist revision rows and resolve new snapshot state after successful mutations.
- [x] 3.5 Add exhaustive service tests for success, conflicts, invalid content, rollback, and delete.
- [x] 3.6 Confirm service calls do not refresh active sessions or alter task-run ownership.

## Implementation Details

Keep the authoring service transport-agnostic. Later tasks should adapt CLI, HTTP, UDS, and Host API routes to this service instead of reimplementing validation or file writes.

### Relevant Files
- `internal/soul/authoring.go` - likely service boundary for managed mutations.
- `internal/soul/resolver.go` - validation and digest source from task_01.
- `internal/store/globaldb/` - revision and snapshot persistence from task_02.
- `internal/fileutil/` or equivalent - atomic writes, path safety, and durable file replacement.
- `internal/diagnostics/` - shared closed diagnostics and redaction.

### Dependent Files
- `internal/soul/*_test.go` - service behavior and edge-case coverage.
- `internal/store/globaldb/*_test.go` - revision writes and rollback readbacks.
- `.compozy/tasks/agent-soul/task_04.md` - explicit refresh/session semantics consume the resolved state.
- `.compozy/tasks/agent-soul/task_11.md` - transport handlers adapt to this service.
- `.compozy/tasks/agent-soul/task_12.md` - CLI commands call this service through the shared core route/client path.

### Related ADRs
- [ADR-001: Optional Scoped SOUL.md Persona Artifact](adrs/adr-001.md) - keeps authored identity scoped.
- [ADR-002: Soul Prompt and Read Model Exposure](adrs/adr-002.md) - defines read-model data consumers.
- [ADR-006: Managed Soul Authoring in v1](adrs/adr-006.md) - defines managed write/delete/history/rollback semantics.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: this service is the only mutation authority that future Host API actions, tools, and SDK methods may call.
- Agent manageability: service models must be ready for structured CLI/HTTP/UDS errors, but no route is exposed here.
- Config lifecycle: honors `[agents.soul]` limits and enabled flags from task_01; no new keys.

### Web/Docs Impact
- Web impact: no UI editor in MVP; generated type impact starts in task_10 and web consumers in task_14.
- Docs impact: task_15 must document managed authoring semantics, CAS, and rollback from this service.

## Deliverables
- Transport-agnostic managed Soul authoring service.
- Atomic write/delete/history/rollback behavior with CAS and revision persistence.
- Redacted deterministic service errors.
- Tests for managed authoring paths and failure modes.
- Completion evidence that active sessions and task leases are not mutated by authoring alone.

## Tests
- Unit tests:
  - [x] Valid write creates/updates `SOUL.md`, appends a revision, and returns the new digest.
  - [x] Invalid content fails closed and does not modify the file or append a success revision.
  - [x] Stale `expected_digest` fails with a deterministic conflict error.
  - [x] Delete appends a revision and removes only the managed file.
  - [x] Rollback restores the selected revision through the same validation path.
  - [x] Path traversal, symlink, and unsupported agent paths are rejected.
- Integration tests:
  - [x] Authoring service persists revision history in the global DB and survives reopen.
  - [x] Repeated writes preserve digest ordering and do not refresh an active session implicitly.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec_soul.md` - managed authoring service requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw.md` - authored file DX precedent.
- `.compozy/tasks/agent-soul/analysis/analysis_paperclip.md` - companion instruction file precedent.
- `.resources/openclaw/src/agents/bootstrap-files.ts:194-288` - generated/managed authored files precedent.
- `.resources/paperclip/server/src/onboarding-assets/ceo/SOUL.md:1-33` - authored soul example.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Every managed Soul mutation is validated, atomic, auditable, and CAS-protected.
- Later transports can expose the service without duplicating write logic.
