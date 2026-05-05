---
status: pending
title: Runtime Routing, Delivery Wrappers, and Task Ingress
type: backend
complexity: critical
dependencies:
  - task_03
  - task_05
---

# Task 06: Runtime Routing, Delivery Wrappers, and Task Ingress

## Overview

Wire the store-backed conversation model into the runtime manager, router, delivery wrapper, and network task-ingress path. This task makes persisted conversation state authoritative before any public API or UI consumes it.

<critical>
- ALWAYS READ `_techspec.md`, all ADRs, `internal/CLAUDE.md`, and tasks 02-05 before editing.
- ACTIVATE `nats`, `agh-code-guidelines`, `golang-pro`, `agh-test-conventions`, `testing-anti-patterns`, and `agh-cleanup-failure-paths`.
- REFERENCE TECHSPEC for router lifecycle, work handoff, prompt metadata, and task-ingress boundaries.
- FOCUS ON runtime orchestration; hooks/metrics are task_07 and public contracts are task_08.
- TESTS REQUIRED for public threads, direct rooms, work handoff, prompt wrappers, and task metadata.
- NO WORKAROUNDS: do not leave old peer-room delivery paths or `interaction_id` shims.
</critical>

<requirements>
- MUST route conversation-bearing messages by `channel + surface + thread_id|direct_id + to`.
- MUST call the store conversation repository before delivery side effects that depend on durable state.
- MUST update delivery wrappers with `surface`, matching container ID, `work_id`, `reply_to`, `trace_id`, `causation_id`, and trust metadata.
- MUST update `PromptNetworkMeta` or equivalent structured prompt metadata to carry the same fields.
- MUST implement public thread to direct-room handoff semantics as a new `work_id` linked by `reply_to`, `trace_id`, and `causation_id`.
- MUST write `task_runs.metadata_json.network_work_id` for network-created task runs without making `network_work` claimable or queue-like.
- MUST preserve task-run claim, lease, heartbeat, complete, fail, release, and cancel ownership under `task_runs`.
</requirements>

## Subtasks

- [ ] 6.1 Replace runtime message persistence calls with the conversation repository.
- [ ] 6.2 Update router dispatch for thread and direct surfaces.
- [ ] 6.3 Update delivery wrappers and structured prompt metadata.
- [ ] 6.4 Update network task ingress to attach `network_work_id` metadata only.
- [ ] 6.5 Add integration tests for thread delivery, direct delivery, handoff, summarize-back linkage, and task metadata.

## Implementation Details

The store write must commit before prompt delivery and before later hook dispatch. If routing fails after commit, logs/audit should identify the delivery result without rolling back the durable conversation.

### Relevant Files

- `internal/network/manager.go` - runtime composition and store calls.
- `internal/network/router.go` - dispatch semantics.
- `internal/network/delivery.go` - prompt wrapper rendering.
- `internal/network/tasks.go` - network task ingress and task metadata.
- `internal/network/audit.go` - delivery/audit relationship until task_07 finalizes observability.
- `internal/acp/types.go` - prompt metadata if required by wrapper structures.
- `internal/session/manager_prompt.go` - prompt augmentation path if it carries network metadata.
- `internal/daemon/boot.go` - dependency wiring.

### Dependent Files

- `internal/hooks/events.go` - task_07 observes committed transitions.
- `internal/api/core/network.go` - task_08 exposes runtime/store state.
- `internal/skills/bundled/skills/agh-network/SKILL.md` - task_12 documents agent behavior.

### Related ADRs

- [ADR-001: Separate Public Threads from Direct Rooms](adrs/adr-001.md) - runtime conversation surfaces.
- [ADR-002: Rename interaction_id to work_id and narrow it to lifecycle-bearing work](adrs/adr-002.md) - task metadata boundary.
- [ADR-003: Make direct a conversation surface, not a message kind](adrs/adr-003.md) - router semantics.

## Extensibility / Agent Manageability / Config Lifecycle

- Extensibility: runtime write results should be structured so hooks and Host API can observe them later.
- Agent manageability: task-ingress correlation must be visible through later CLI/API/native surfaces.
- Config lifecycle: no new config keys; existing network enablement still gates runtime behavior.

### Web/Docs Impact

- Web impact: runtime payloads must support later web route isolation and direct-room visibility.
- Docs impact: task_16 documents handoff and task-ingress semantics.

## Deliverables

- Runtime router/delivery integration with store-backed conversation containers.
- Prompt wrapper and structured metadata changes.
- Task ingress using `network_work_id` metadata only.
- Integration tests for public thread, direct room, handoff, and task metadata flows.

## Tests

- Unit tests:
  - [ ] Router dispatches `surface:"thread"` by thread container.
  - [ ] Router dispatches `surface:"direct"` by direct-room membership.
  - [ ] Delivery wrapper includes exact container/work/correlation metadata.
  - [ ] Prompt metadata contains no legacy interaction fields.
- Integration tests:
  - [ ] Public thread creation and message delivery persist and deliver expected wrappers.
  - [ ] Direct-room resolve plus direct message delivery persists and delivers expected wrappers.
  - [ ] Public-to-direct handoff creates a new work ID and links by reply/trace/causation.
  - [ ] Summarize-back posts a public `say` without leaking restricted direct-room messages.
  - [ ] Network-created task runs carry `metadata_json.network_work_id` and remain claimable only through task-run machinery.
- Test coverage target: >=80% for touched runtime packages.
- All tests must pass.

## Success Criteria

- Runtime uses conversation containers end-to-end for routing and delivery.
- Work metadata correlates network task ingress without becoming a queue.
- Prompt wrappers give agents enough structure to respond in the correct thread or direct room by default.
