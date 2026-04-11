---
status: pending
title: Add per-instance channel observability and health reporting
type: backend
complexity: high
dependencies:
  - task_06
  - task_07
  - task_08
---

# Task 10: Add per-instance channel observability and health reporting

## Overview

Extend the observability surface so channel adapters are visible as operational units rather than opaque extension processes. This task adds per-instance health and delivery telemetry that operators can use to understand readiness, backlog, auth failures, and active route counts.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST extend observability and health reporting so channel state is tracked per `channel_instance`, not only per extension process.
2. MUST surface the minimum statuses defined in the TechSpec: `disabled`, `starting`, `ready`, `degraded`, `auth_required`, and `error`.
3. MUST track or expose delivery backlog, route counts, and auth/delivery failure signals needed for operational diagnosis.
4. SHOULD keep channel observability additive to the existing health surface so active session and agent reporting remain intact.
</requirements>

## Subtasks
- [ ] 10.1 Extend observe health and telemetry models with channel-instance metrics and status data
- [ ] 10.2 Add channel backlog, route-count, and failure-signal reporting from the channel runtime
- [ ] 10.3 Expose per-instance health details through the existing health and channel APIs where appropriate
- [ ] 10.4 Add unit and integration tests for health aggregation and state propagation

## Implementation Details

Follow the TechSpec sections "Monitoring and Observability", "Operational visibility", and "Health surface". This task should focus on observability and health behavior only; it should not add a web UI or a reference adapter.

### Relevant Files
- `internal/observe/health.go` — Current health snapshot must expand to include channel-instance visibility
- `internal/observe/observer.go` — Existing observability aggregation logic is the natural place to attach channel signals
- `internal/observe/query.go` — Query surfaces may need to expose channel-oriented telemetry details
- `internal/api/contract/contract.go` — Health payloads and channel detail payloads may need additive transport fields
- `internal/api/httpapi/routes.go` — Existing health endpoints and channel endpoints will expose the new observability data

### Dependent Files
- `internal/daemon/boot.go` — Observability depends on the daemon wiring channel runtime state into the observer
- `internal/api/httpapi/handlers_test.go` — Health and channel transport tests should expand for the new observability payloads
- `internal/extension/manager.go` — Channel instance state transitions emitted from extension lifecycle code should feed these metrics

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Requires the daemon-owned substrate to be operationally visible
- [ADR-007: Negotiated Channel Delivery Stream for Real-Time Outbound Messaging](adrs/adr-007.md) — Delivery backlog and recovery state should be observable at the channel-instance level

## Deliverables
- Additive observability and health support for per-instance channel status and metrics
- Transport-visible channel health details for operational debugging
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for health aggregation and state propagation **(REQUIRED)**

## Tests
- Unit tests:
  - [ ] Channel health aggregation reports the expected status counts across `starting`, `ready`, `degraded`, and `auth_required` instances
  - [ ] Backlog metrics increase and decrease with queued delivery work without affecting existing active-session counts
  - [ ] Route-count reporting reflects the number of active routes owned by a channel instance
  - [ ] Auth failures and terminal delivery failures are surfaced on the owning channel instance instead of only the extension process
- Integration tests:
  - [ ] `/api/observe/health` includes additive channel metrics while preserving existing daemon and session health fields
  - [ ] Updating a channel instance from `ready` to `auth_required` is visible through the health surface and the channel detail API
  - [ ] A queued delivery scenario reports backlog for the affected channel instance and clears it once delivery completes
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Operators can inspect channel health and backlog per instance instead of inferring it from generic extension status
- Existing daemon health reporting remains intact while gaining additive channel visibility
