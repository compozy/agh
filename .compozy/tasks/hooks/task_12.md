---
status: pending
title: Hook observability storage and HTTP introspection
type: backend
complexity: medium
dependencies:
  - task_09
---

# Task 12: Hook observability storage and HTTP introspection

## Overview

Implement `HookRunRecord` persistence in the observability store and three HTTP introspection endpoints: catalog (resolved hooks with ordering), runs (execution history with patch audit), and events (taxonomy with eligibility). This provides the debugging and forensic capabilities the platform needs.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST persist `HookRunRecord` to the observability store with all fields: hook name, event, source, mode, duration, outcome, dispatch depth, `PatchApplied`
- MUST populate `PatchApplied` for security-relevant families: `permission.*`, `prompt.*`, `tool.*`, `input.*`
- MUST omit `PatchApplied` for other families unless debug mode is enabled
- MUST implement `GET /api/hooks/catalog?workspace=:id&agent=:name` — returns resolved hooks with source attribution and pipeline order
- MUST implement `GET /api/hooks/runs?session=:id&event=:event` — returns execution history including patch diffs
- MUST implement `GET /api/hooks/events` — returns taxonomy with sync eligibility and payload/patch schema names
- MUST integrate telemetry emission into the dispatch pipeline (called after each hook execution)
- MUST add metrics: dispatch count, latency, queue depth, drop count, permission escalation blocks, depth violations
</requirements>

## Subtasks
- [ ] 12.1 Add `HookRunRecord` schema to observability store
- [ ] 12.2 Implement telemetry emitter called by pipeline after each hook execution
- [ ] 12.3 Implement `GET /api/hooks/catalog` endpoint
- [ ] 12.4 Implement `GET /api/hooks/runs` endpoint with patch audit data
- [ ] 12.5 Implement `GET /api/hooks/events` endpoint
- [ ] 12.6 Add structured log events and metrics per TechSpec monitoring section
- [ ] 12.7 Write tests for storage, endpoints, and telemetry

## Implementation Details

Create/modify files:
- `internal/observe/` — Add HookRunRecord to store schema, add WriteHookRecord method
- `internal/api/httpapi/` — Add hook introspection endpoint handlers
- `internal/hooks/telemetry.go` — Telemetry emitter integrated into pipeline

Reference TechSpec "Monitoring and Observability" and "API Endpoints" sections.

### Relevant Files
- `internal/observe/observer.go` — Observer struct and Registry interface
- `internal/store/sessiondb/` — Per-session event store (for run records)
- `internal/api/httpapi/` — HTTP handler patterns
- `internal/api/contract/` — Shared contract types for API responses
- `internal/hooks/pipeline.go` (task_04) — Pipeline calls telemetry after each hook
- `internal/hooks/types.go` (task_01) — HookRunRecord struct

### Dependent Files
- `internal/observe/` — New methods for hook telemetry
- `internal/api/httpapi/` — New endpoint handlers
- `internal/store/sessiondb/` — New table/schema for run records

### Related ADRs
- [ADR-010: Persist Patch Audit Trail for Security-Relevant Families](../adrs/adr-010.md) — Patch audit storage

## Deliverables
- HookRunRecord persistence in observability store
- Three HTTP introspection endpoints
- Telemetry emitter in dispatch pipeline
- Structured logs and metrics
- Unit and integration tests with 80%+ coverage **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] HookRunRecord persisted with all fields for security-relevant family (permission.request)
  - [ ] HookRunRecord PatchApplied is null for non-security family (session.post_create) in normal mode
  - [ ] HookRunRecord PatchApplied is populated for non-security family when debug mode enabled
  - [ ] Telemetry emitter records duration and outcome correctly
  - [ ] Permission escalation block generates `hook.dispatch.permission_escalation_blocked` log
- Integration tests:
  - [ ] `GET /api/hooks/catalog` returns hooks sorted by pipeline order with source attribution
  - [ ] `GET /api/hooks/catalog?workspace=X` filters to workspace-scoped hooks
  - [ ] `GET /api/hooks/runs?session=X` returns execution history with patch diffs
  - [ ] `GET /api/hooks/events` returns all 27 events with correct sync eligibility
  - [ ] Dispatch → store → query cycle: hook fires, record persisted, endpoint returns it
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Patch audit data available for forensic analysis via runs endpoint
- Catalog endpoint shows resolved pipeline order for debugging
- All structured log events from TechSpec monitoring section are emitted
