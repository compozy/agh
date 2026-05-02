---
status: completed
title: Add Heartbeat Config and Policy Resolver Foundation
type: backend
complexity: high
dependencies: []
---

# Task 05: Add Heartbeat Config and Policy Resolver Foundation

## Overview

Create the backend foundation for optional `HEARTBEAT.md` wake-policy files. This task defines config authority, parsing, validation, digesting, active-hours preferences, and diagnostics while explicitly keeping liveness, lease renewal, queueing, and scheduler ownership outside authored Markdown.

<critical>
- ALWAYS READ `_techspec.md`, `_techspec_soul.md`, `_techspec_heartbeat.md`, and ADR-007 through ADR-011 before implementation.
- REFERENCE TECHSPEC for `HEARTBEAT.md` scope, config authority, allowed fields, and failure behavior.
- FOCUS ON WHAT must exist: config, parser, policy resolver, digest, diagnostics, and config-bound preferences.
- MINIMIZE CODE in task notes; do not implement wake scheduling in this task.
- TESTS REQUIRED for config bounds, active hours, invalid authority claims, deterministic digests, and redaction.
- NO WORKAROUNDS: authored heartbeat policy must not become liveness, a task queue, a lease, or a separate run table.
</critical>

<requirements>
- MUST activate `agh-code-guidelines` and `golang-pro` before editing production Go.
- MUST activate `agh-test-conventions` and `testing-anti-patterns` before writing tests.
- MUST add `[agents.heartbeat]` config keys, defaults, validation, merge/overlay behavior, and examples.
- MUST parse `HEARTBEAT.md` as optional authored wake/reentry policy with strict frontmatter plus body rules from the TechSpec.
- MUST reject fields that redefine `[session.supervision]`, scheduler cadence, network greet, task lease heartbeat, `ClaimNextRun`, queues, ownership, or independent run loops.
- MUST resolve active-hours and cadence preferences only within `[agents.heartbeat]` bounds.
- MUST produce deterministic `heartbeat_digest`, config digest/provenance, compact prompt contribution, status data, and redacted diagnostics.
</requirements>

## Subtasks
- [x] 5.1 Add `[agents.heartbeat]` config structs, defaults, validation, and config tests.
- [x] 5.2 Implement `HEARTBEAT.md` parser and strict allowed-field validation.
- [x] 5.3 Implement policy resolution for prompt contribution, active-hours preferences, and config-bound cadence hints.
- [x] 5.4 Add deterministic digest and redacted diagnostics.
- [x] 5.5 Add tests for invalid authority claims, config bounds, missing files, and oversized content.
- [x] 5.6 Confirm no session health, scheduler wake, task lease, or network greet behavior changes in this task.

## Implementation Details

Keep the policy resolver isolated so later tasks can use the same resolved representation for authoring, status, scheduler decisions, prompt contributions, and hooks. The resolver must make the boundary explicit: `HEARTBEAT.md` is policy input, not runtime liveness.

### Relevant Files
- `internal/config/config.go` - add and validate `[agents.heartbeat]` config.
- `internal/config/agent.go` - connect agent-scoped defaults if needed.
- `internal/frontmatter/frontmatter.go` - reuse strict Markdown frontmatter parsing.
- `internal/heartbeat/` - likely destination for resolver, policy, digest, diagnostics, and validation.
- `internal/diagnostics/` - redacted error and warning reporting.

### Dependent Files
- `internal/config/*_test.go` - heartbeat config defaults and validation.
- `internal/heartbeat/*_test.go` - policy parsing, digest, diagnostics, and bounds.
- `.compozy/tasks/agent-soul/task_06.md` - persists resolved Heartbeat policy and session health state.
- `.compozy/tasks/agent-soul/task_08.md` - exposes managed Heartbeat authoring through the service.
- `.compozy/tasks/agent-soul/task_09.md` - consumes policy for wake decisions.

### Related ADRs
- [ADR-007: HEARTBEAT.md Is Advisory Wake Policy](adrs/adr-007.md) - defines the artifact boundary.
- [ADR-009: Separate Session Health From HEARTBEAT.md](adrs/adr-009.md) - prevents liveness confusion.
- [ADR-011: Config Authority for Cadence and Wake Limits](adrs/adr-011.md) - defines config precedence.

## Extensibility / Agent Manageability / Config Lifecycle
- Extensibility: produce resolver types that later Host API, hooks, tools/resources, SDKs, and bundles can read without claiming runtime authority.
- Agent manageability: no external verbs in this task; service and route exposure are deferred to tasks 08, 10, 11, and 12.
- Config lifecycle: add `[agents.heartbeat]` keys, defaults, validation, examples, and tests; documentation deferred to task_15.

### Web/Docs Impact
- Web impact: no generated types or UI changes in this task.
- Docs impact: task_15 must document the config authority and the distinction between policy and liveness.

## Deliverables
- `[agents.heartbeat]` config model with defaults, bounds, and validation.
- `HEARTBEAT.md` parser and policy resolver.
- Deterministic policy digest and config digest/provenance.
- Redacted diagnostics for invalid wake-policy content.
- Tests proving authored policy cannot override liveness, task leases, scheduler ownership, or network greet.

## Tests
- Unit tests:
  - [x] Valid `HEARTBEAT.md` resolves to a deterministic policy and digest.
  - [x] Active-hours and cadence preferences are clamped or rejected according to config bounds.
  - [x] Attempts to declare queues, leases, liveness, `ClaimNextRun`, or network presence fail closed.
  - [x] Missing optional `HEARTBEAT.md` resolves according to enabled/default config.
  - [x] Oversized content and unsupported frontmatter produce redacted diagnostics.
- Integration tests:
  - [x] Agent fixture loading can resolve heartbeat policy without mutating scheduler, sessions, task runs, or network state.
  - [x] Config overlays produce deterministic effective policy digests.
- Test coverage target: >=80%.
- All tests must pass.

## References
- `_techspec_heartbeat.md` - normative Heartbeat policy behavior.
- `.compozy/tasks/agent-soul/analysis/analysis_openclaw_heartbeat.md` - OpenClaw authored wake policy and runner findings.
- `.compozy/tasks/agent-soul/analysis/analysis_hermes_heartbeat.md` - Hermes durable task-run heartbeat contrast.
- `.resources/openclaw/docs/gateway/heartbeat.md:41-59` - `HEARTBEAT.md` policy/checklist precedent.
- `.resources/openclaw/docs/reference/templates/HEARTBEAT.md:8-12` - authored template precedent.
- `.resources/paperclip/server/src/onboarding-assets/ceo/HEARTBEAT.md:1-85` - rich wake policy example.

## Success Criteria
- All tests passing.
- Test coverage >=80%.
- `HEARTBEAT.md` is represented as config-bound wake/reentry policy, not session liveness or task ownership.
- Later Heartbeat tasks can consume one resolver without duplicating validation or digest logic.
