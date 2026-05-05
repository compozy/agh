---
status: pending
title: Network Hooks, Status Counters, and Observability
type: backend
complexity: critical
dependencies:
  - task_05
  - task_06
---

# Task 07: Network Hooks, Status Counters, and Observability

## Overview

Add post-commit network observation surfaces after conversation writes are durable. This task implements the `network` hook family, status counters, structured logs, audit fields, and low-cardinality metrics without giving hooks authority over routing or persistence.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and the hooks-related memory before editing.
- ACTIVATE `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`, and `deadlock-finder-and-fixer` if concurrency or async dispatch changes.
- REFERENCE TECHSPEC for hook delivery semantics and metric cardinality.
- FOCUS ON post-commit observation; do not introduce sync hooks, deny hooks, replay logs, or table tailers.
- TESTS REQUIRED for hook catalog, payloads, post-commit dispatch, failure isolation, redaction, and cardinality.
- NO WORKAROUNDS: hooks must not compensate for missing transaction boundaries.
</critical>

<requirements>
- MUST add hook events `network.thread.opened`, `network.direct_room.opened`, `network.message.persisted`, `network.work.opened`, `network.work.transitioned`, and `network.work.closed`.
- MUST dispatch hooks after durable commit at the state-transition call site.
- MUST make network hooks best-effort, fire-and-forget, and non-replayed in MVP.
- MUST log hook failures without rolling back committed network writes.
- MUST include stable dedupe fields in payloads: event, message ID, work ID, trace ID where present.
- MUST expose aggregate status counters for open threads, open direct rooms, open work items, message totals, work transition totals, delivery queue depth, and direct resolve totals.
- MUST keep high-cardinality IDs out of metric labels while preserving them in structured logs and audit rows.
</requirements>

## Subtasks

- [ ] 7.1 Add network hook event catalog, payload types, matcher support, and introspection.
- [ ] 7.2 Wire post-commit hook dispatch from committed conversation write results.
- [ ] 7.3 Update daemon hook bridge and failure logging.
- [ ] 7.4 Update status counters and metrics with approved labels.
- [ ] 7.5 Add observability, redaction, and failure-isolation tests.

## Implementation Details

Network hooks observe already-committed state. A crash after commit but before hook dispatch may lose the notification; hook consumers dedupe by stable payload identifiers.

### Relevant Files

- `internal/hooks/events.go` - hook event constants.
- `internal/hooks/payloads.go` - network payloads.
- `internal/hooks/types.go` - hook family typing if needed.
- `internal/hooks/matcher.go` - matcher support.
- `internal/hooks/dispatch.go` - async dispatch behavior.
- `internal/hooks/introspection.go` - hook discovery.
- `internal/daemon/hooks_bridge.go` - daemon wiring.
- `internal/network/manager.go` - dispatch call sites after commit.
- `internal/network/stats.go` - status counters and metrics.
- `internal/network/audit.go` - structured audit/log field updates.

### Dependent Files

- `internal/api/core/network.go` - task_08 exposes status payloads.
- `internal/extension/host_api.go` - task_11 may expose hook-related capabilities indirectly.
- `packages/site/content/runtime/core/network/*` - task_16 documents observation semantics.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - hook payload container fields.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - work events.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: hooks are extension-visible observation points, not mutation points.
- Agent manageability: status counters support CLI/API/native-tool status later.
- Config lifecycle: no new config keys; no retention/unread/notification controls.

### Web/Docs Impact

- Web impact: status payloads expose aggregate counts only; the settings UI scope is defined by `_design.md` §10 row A8 and §7.4 — implementers must not invent controls beyond that surface.
- Docs impact: task_16 must document best-effort hook semantics and no replay log in MVP.

## Deliverables

- Network hook family and payloads.
- Post-commit async hook dispatch.
- Status counters, structured log fields, audit fields, and low-cardinality metrics.
- Tests for redaction, cardinality, dispatch timing, and failure isolation.

## Tests

- Unit tests:
  - [ ] Hook catalog and introspection include all network events.
  - [ ] Hook matchers normalize and match network payload fields.
  - [ ] Hook payloads include container/work/correlation fields and exclude raw tokens.
  - [ ] Metric labels exclude `thread_id`, `direct_id`, `work_id`, `message_id`, `trace_id`, and `causation_id`.
- Integration tests:
  - [ ] Hooks dispatch only after durable commit.
  - [ ] Hook failure does not roll back persisted messages or work state.
  - [ ] Duplicate notifications are dedupe-able by stable payload identifiers.
  - [ ] Runtime status reports aggregate thread/direct/work counters.
- Test coverage target: >=80% for touched packages.
- All tests must pass.

## Success Criteria

- Extensions can observe network conversation activity without controlling it.
- Observability is useful and safe under cardinality and redaction constraints.
- Runtime status reflects conversation containers without introducing unsupported configuration knobs.
