---
status: completed
title: Implement Heartbeat Wake Service and Scheduler Integration
type: backend
complexity: critical
dependencies:
  - task_05
  - task_07
  - task_08
---

# Task 09: Implement Heartbeat Wake Service and Scheduler Integration

## Overview

Implement the runtime service that uses valid `HEARTBEAT.md` policy and session health to decide whether an eligible session should receive a synthetic reentry prompt. This task integrates with AGH's existing scheduler and synthetic prompt paths while preserving `ClaimNextRun` and task-run lease heartbeat as the only task ownership primitives.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_heartbeat.md`, ADR-007 through ADR-011, and current scheduler/session code before editing.
- REFERENCE TECHSPEC for wake eligibility, prompt-gate races, latest-snapshot selection, coalescing, cooldowns, and audit events.
- FOCUS ON WHAT must hold: wake policy can advise reentry only for already eligible sessions.
- MINIMIZE CODE in task notes; reuse existing scheduler and synthetic prompt primitives.
- TESTS REQUIRED for latest-valid-snapshot decisions, busy prompt gates, cooldowns, coalescing, audit events, and no task claims.
- NO WORKAROUNDS: do not create `task_runs`, claim tokens, independent heartbeat run loops, or network greet messages for wake policy.
</critical>

<requirements>
- MUST activate `agh-code-guidelines`, `golang-pro`, and `deadlock-finder-and-fixer` before editing runtime concurrency code.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST implement a wake service that evaluates latest valid Heartbeat policy at decision time.
- MUST require healthy/eligible session health before synthetic wake.
- MUST apply config-bound cooldown, coalescing, active-hours, max-wake, prompt-gate, and disabled-state rules.
- MUST inject synthetic reentry prompts through existing session prompt paths without claim tokens or task lease renewal.
- MUST write wake audit events with closed reason codes for skipped, coalesced, sent, failed, and ineligible decisions.
- MUST integrate scheduler/manual/harness reentry without turning Heartbeat into a queue or work ownership system.
</requirements>

## Subtasks
- [x] 9.1 Define wake decision inputs, outputs, reason enum, and audit event shape.
- [x] 9.2 Implement wake eligibility evaluation using latest policy, session health, config, and wake state.
- [x] 9.3 Integrate with the existing scheduler and synthetic prompt entrypoints.
- [x] 9.4 Handle prompt-gate races, active prompts, coalescing, cooldowns, and active-hours boundaries.
- [x] 9.5 Record wake state and audit events for every decision.
- [x] 9.6 Add concurrency, scheduler, prompt, and no-claim-token tests.

## Implementation Details

The wake service should operate as a policy consumer. It may ask an existing eligible session to reorient, but it must not create sessions, create task runs, claim tasks, renew leases, or bypass the session prompt gate.

### Relevant Files
- `internal/scheduler/scheduler.go` - scheduler tick and dispatch integration.
- `internal/daemon/scheduler_runtime.go` - daemon wiring for scheduler services.
- `internal/session/synthetic_prompt.go` - existing synthetic prompt bridge.
- `internal/session/manager_prompt.go` - prompt gate and active prompt state.
- `internal/daemon/harness_reentry_bridge.go` - harness reentry integration.
- `internal/heartbeat/` - wake service and policy/status inputs.
- `internal/store/globaldb/` - wake state and event persistence from task_06.
- `internal/task/lease_manager.go` - read only to preserve `ClaimNextRun` authority.
- `internal/network/` - read only to confirm AGH Network greet remains unaffected.

### Dependent Files
- `internal/heartbeat/*_test.go` - wake decision and audit behavior.
- `internal/scheduler/*_test.go` - scheduler integration and timing behavior.
- `internal/session/*_test.go` - synthetic prompt and prompt-gate race coverage.
- `internal/daemon/*_test.go` - daemon wiring and harness reentry coverage.
- `.compozy/tasks/agent-soul/task_10.md` - exposes wake status/audit contract data.
- `.compozy/tasks/agent-soul/task_17.md` - real-scenario QA proves end-to-end wake behavior.

### Related ADRs
- [ADR-007: HEARTBEAT.md Is Advisory Wake Policy](adrs/adr-007.md) - defines wake policy scope.
- [ADR-008: Heartbeat Snapshots and Wake Audit](adrs/adr-008.md) - defines audit/state behavior.
- [ADR-009: Separate Session Health From HEARTBEAT.md](adrs/adr-009.md) - defines health dependency.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - defines wake limits.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: emit stable wake events and hook payload data for extension observers without giving extensions queue ownership.
- Agent manageability: prepare status/audit data for CLI, HTTP, UDS, Host API, and tools; route exposure is later.
- Config lifecycle: consume `[agents.heartbeat]` wake limits and effective digest; do not allow Markdown policy to redefine config.

### Web/Docs Impact
- Web impact: generated contract consumers are handled in tasks 10 and 14; no interactive UI is required in MVP.
- Docs impact: task_15 must document wake behavior, skip reasons, cooldowns, prompt-gate behavior, and the no-queue boundary.

## Deliverables
- Heartbeat wake service with scheduler/manual/harness integration.
- Synthetic reentry prompts for eligible sessions through existing prompt paths.
- Wake state and audit events for sent, skipped, coalesced, failed, and ineligible decisions.
- Tests for concurrency, prompt gate races, cooldowns, latest snapshot selection, and no task ownership changes.
- Completion evidence that no task runs, claim tokens, session creations, network greet messages, or lease renewals were introduced by wake policy.

## Tests
- Unit tests:
  - [x] Wake decision uses the latest valid policy snapshot at decision time.
  - [x] Invalid, disabled, stale, detached, active, or ineligible sessions are skipped with closed reasons.
  - [x] Cooldown, coalescing, active-hours, and max-wake rules produce deterministic results.
  - [x] Prompt-gate race between eligibility check and prompt dispatch is handled without duplicate wakes.
  - [x] Wake prompt text contains no claim token, task lease token, or queue semantics.
  - [x] Every decision writes the correct audit event and updates wake state only when appropriate.
- Integration tests:
  - [x] Scheduler tick can wake an eligible idle session through the synthetic prompt path.
  - [x] Manual/harness reentry uses the same service and audit path.
  - [x] Task-run lease heartbeat and `ClaimNextRun` behavior remain unchanged while wake policy runs.
- Test coverage target: >=80%.
- All tests must pass.

## Completion Evidence
- `go test ./internal/heartbeat -cover -count=1` passed with 80.3% statement coverage.
- `go test -race ./internal/heartbeat ./internal/session ./internal/daemon ./internal/scheduler -run 'TestManagedWakeServiceDecision|TestManagedWakeServiceClosedSkipsAndValidation|TestPromptSyntheticHeartbeatWakeOptions|TestSchedulerHeartbeatWakeIntegration|TestHarnessHeartbeatWakeIntegration|TestRunOnceDispatchesSelectedTargetsAsBatch|TestRunOnce' -count=1` passed.
- `make verify` passed before marking the task complete.
- Self-review search found no new Heartbeat task-run creation, claim-token handling, lease renewal, session creation, or AGH Network greet integration.

## References
- `_techspec_heartbeat.md` - wake service and scheduler integration requirements.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - OpenClaw scheduled wake/coalescing patterns.
- `.compozy/tasks/agent-soul/analysis/analysis_paperclip_heartbeat.md` - Paperclip wake run/coalescing contrast.
- `.resources/openclaw/src/infra/heartbeat-wake.ts:42-208` - wake decision and prompt contribution precedent.
- `.resources/openclaw/src/infra/heartbeat-runner.ts:610-725` - scheduler/coalescing precedent.
- `.resources/paperclip/packages/db/src/schema/agent_wakeup_requests.ts:5-40` - queue pattern to avoid.
- `.resources/paperclip/packages/db/src/schema/heartbeat_runs.ts:6-82` - independent run table pattern to avoid.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- Eligible sessions can be reoriented by Heartbeat policy without creating work, claiming tasks, renewing leases, or changing network presence.
- Wake behavior is auditable, bounded, and driven by session health plus config-bound policy.
