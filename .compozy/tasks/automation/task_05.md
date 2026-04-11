---
status: completed
title: Build trigger engine with normalized ingress and webhook auth
type: backend
complexity: critical
dependencies:
  - task_03
---

# Task 05: Build trigger engine with normalized ingress and webhook auth

## Overview

Implement the event-driven side of automation around normalized activation envelopes instead of ad hoc callbacks or map-shaped payload plumbing. This task is responsible for exact-match trigger evaluation, strict prompt templating, internal ingress from the existing observer/hooks boundary, and authenticated webhook normalization.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST normalize all trigger input into `ActivationEnvelope` values before matching so session, memory, webhook, hook, and extension-originated events share one matching path.
2. MUST consume internal lifecycle signals from the existing observer and hooks boundary instead of introducing direct subscription APIs inside `session` or `memory/consolidation`.
3. MUST enforce scope-aware exact-match filtering and strict `text/template` execution with `missingkey=error` for trigger prompts.
4. MUST validate webhook timestamp freshness and HMAC signatures before any trigger matching or dispatch occurs.
</requirements>

## Subtasks
- [x] 5.1 Implement trigger matching against normalized activation envelopes
- [x] 5.2 Add strict filter and prompt-template evaluation behavior
- [x] 5.3 Add internal ingress adapters for observer/hooks-backed session, memory, and hook-completion events
- [x] 5.4 Add webhook endpoint parsing, slug-plus-id resolution, timestamp validation, and HMAC verification helpers
- [x] 5.5 Add tests proving authenticated webhook and internal event normalization behavior

## Implementation Details

Follow the TechSpec sections "ActivationEnvelope", "Session Notifier Integration", "Memory Consolidation Integration", "Webhook HTTP Integration", and "Testing Approach". This task should stop at the trigger-engine boundary; route registration and transport handlers belong to later API work.

### Relevant Files
- `internal/daemon/hooks_bridge.go` — Existing lifecycle and hook bridge behavior defines the canonical internal event boundary automation must consume
- `internal/memory/consolidation/runtime.go` — Dream consolidation completion is one of the built-in trigger sources and will need normalization
- `internal/hooks/events.go` — Hook event names and lifecycle taxonomy need to stay aligned with trigger event kinds
- `internal/observe/` — Trigger ingress should fit the existing observer-facing event model rather than inventing a new bus

### Dependent Files
- `internal/api/httpapi/routes.go` — Later webhook routes will hand HTTP deliveries into this trigger engine
- `internal/extension/host_api.go` — Extension-fired `ext.*` trigger events will eventually route through the same matching path
- `internal/automation/manager.go` — The manager will compose and start this engine in task 06

### Related ADRs
- [ADR-001: Built-In Daemon Component with Extension Integration Points](adrs/adr-001.md) — Confirms automation stays built-in while remaining extensible
- [ADR-002: Unified Automation Model — Schedules and Triggers](adrs/adr-002.md) — Requires triggers to share execution governance with schedules

## Deliverables
- Trigger engine with normalized matching, strict filters, and strict templating
- Internal ingress adapters and authenticated webhook normalization helpers
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for internal and webhook-triggered activation flows **(REQUIRED)**

## Tests
- Unit tests:
  - [x] A trigger with `filter = {"data.agent_name": "researcher"}` matches only envelopes with the exact field value
  - [x] A trigger prompt template referencing a missing envelope field fails validation under `missingkey=error`
  - [x] Webhook endpoint parsing resolves the stable `webhook_id` from a `slug--wbh_*` endpoint value
  - [x] Webhook HMAC validation rejects an invalid signature before trigger lookup returns a dispatchable activation
  - [x] Webhook validation rejects a stale timestamp outside the accepted freshness window
- Integration tests:
  - [x] A `session.stopped` event flowing through the existing observer/hooks boundary produces one matched trigger activation without a direct session subscription
  - [x] A `memory.consolidated` event is normalized and can dispatch a matching trigger
  - [x] A valid webhook request is normalized into an activation envelope and dispatched exactly once
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Trigger matching uses one normalized ingress model across internal and external sources
- Webhook authentication is mandatory and enforced before dispatch
