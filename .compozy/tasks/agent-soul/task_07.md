---
status: completed
title: Implement Metadata-Only Session Health
type: backend
complexity: critical
dependencies:
  - task_06
---

# Task 07: Implement Metadata-Only Session Health

## Overview

Implement the runtime primitive for normal session health and presence. This task gives AGH a metadata-only way to know whether a session is idle, active, stale, hung, detached, attachable, and eligible for wake without prompting the model or renewing task leases.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_heartbeat.md`, ADR-009, and current session supervision code before implementation.
- REFERENCE TECHSPEC for session health states, fields, transitions, restart behavior, and route-ready read models.
- FOCUS ON WHAT must be true: health is metadata-only, session-owned, wake-eligibility input, and separate from `HEARTBEAT.md`.
- MINIMIZE CODE in notes; integrate with existing session supervision and manager boundaries.
- TESTS REQUIRED for active prompt activity, idle presence, stale detection, restart recovery, and no model/tool/task side effects.
- NO WORKAROUNDS: do not name public APIs `session heartbeat` and do not inject prompts to prove liveness.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST implement metadata-only session health/presence updates for normal sessions.
- MUST distinguish active prompt activity heartbeat from idle session presence/health.
- MUST expose internal read models with `state`, `health`, `last_activity_at`, `last_presence_at`, `active_prompt`, `attachable`, `eligible_for_wake`, and `ineligibility_reason`.
- MUST mark or recompute stale rows on daemon restart according to the TechSpec.
- MUST avoid prompt injection, model calls, tool calls, task creation, task lease renewal, ACP heavy events, and network greet updates.
</requirements>

## Subtasks
- [x] 7.1 Define session health state and reason enums with closed values.
- [x] 7.2 Integrate active prompt activity updates with existing session supervision.
- [x] 7.3 Add idle presence/health updates for attachable normal sessions.
- [x] 7.4 Add restart recovery and stale/hung/detached detection.
- [x] 7.5 Add internal read models consumed by routes, scheduler, Host API, and tools later.
- [x] 7.6 Add unit and integration tests proving metadata-only behavior and wake eligibility.

## Implementation Details

This task should reinforce or add the `session.health` primitive that Heartbeat wake decisions consume. It must not make `HEARTBEAT.md` responsible for liveness and must not rename task-run lease heartbeat concepts.

### Relevant Files
- `internal/session/prompt_activity.go` - existing active prompt activity/supervision primitive.
- `internal/session/liveness.go` - likely destination for session health/presence if added.
- `internal/session/manager.go` - session lifecycle and attachability state.
- `internal/session/manager_prompt.go` - active prompt transitions.
- `internal/session/query.go` - session read model inputs.
- `internal/store/globaldb/` - `session_health` store methods from task_06.
- `internal/api/core/conversions.go` - route-ready conversion later, if core types already exist.

### Dependent Files
- `internal/session/*_test.go` - health transitions, restart, stale, and side-effect coverage.
- `internal/store/globaldb/*_test.go` - session health persistence behavior.
- `.compozy/tasks/agent-soul/task_08.md` - uses health for policy status.
- `.compozy/tasks/agent-soul/task_09.md` - uses health for wake eligibility.
- `.compozy/tasks/agent-soul/task_11.md` - exposes health through HTTP and UDS.
- `.compozy/tasks/agent-soul/task_12.md` - exposes `agh session health/status/inspect`.

### Related ADRs
- [ADR-007: HEARTBEAT.md Is Advisory Wake Policy](adrs/adr-007.md) - keeps policy separate from liveness.
- [ADR-009: Separate Session Health From HEARTBEAT.md](adrs/adr-009.md) - defines this runtime primitive.
- [ADR-010: Managed Heartbeat and Session Health Surfaces](adrs/adr-010.md) - requires agent-operable health reads.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: session health read models must be usable by Host API, hooks, tools/resources, SDKs, scheduler, and future bridge integrations.
- Agent manageability: prepares data for `agh session health`, `agh session status`, `agh session inspect`, HTTP routes, UDS routes, and Host API reads.
- Config lifecycle: consumes existing `[session.supervision]` and `[agents.heartbeat]` bounds; do not redefine their keys here.

### Web/Docs Impact
- Web impact: generated session health types are handled in tasks 10 and 14; no UI page is required in MVP.
- Docs impact: task_15 must document the difference between task-run lease heartbeat, active prompt activity, idle session health, and `HEARTBEAT.md` policy.

## Deliverables
- Metadata-only session health/presence primitive.
- Internal read models with closed states and wake eligibility reasons.
- Restart recovery/stale detection behavior.
- Tests proving no prompt/model/tool/task side effects.
- Completion evidence that no public `agh session heartbeat` command or route was introduced.

## Tests
- Unit tests:
  - [x] Active prompt transitions update activity without renewing task leases.
  - [x] Idle attached sessions update presence/health without model calls.
  - [x] Detached, stale, hung, dead, and ineligible states produce deterministic reasons.
  - [x] Restart recovery marks stale rows or recomputes health according to the TechSpec.
  - [x] Health updates do not emit heavy ACP prompt events or network greet messages.
- Integration tests:
  - [x] A normal session can move idle -> active -> idle with persisted health evidence.
  - [x] Scheduler-facing eligibility reads return false for busy, stale, detached, or dead sessions.
  - [x] Task-run lease heartbeat remains unchanged during normal session health updates.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec_heartbeat.md` - session health primitive requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_agh_heartbeat.md` - local AGH liveness architecture.
- `.compozy/tasks/agent-soul/analysis/analysis_hermes_heartbeat.md` - Hermes liveness/activity separation.
- `.resources/hermes/run_agent.py:4518-4568` - activity/heartbeat status contrast.
- `.resources/hermes/run_agent.py:7271-7350` - run liveness and progress reporting contrast.
- `.resources/paperclip/server/src/services/run-liveness.ts:292-347` - liveness service precedent.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Normal sessions have queryable health and wake eligibility without periodic prompt turns.
- `HEARTBEAT.md` remains policy input only and does not implement liveness.
