---
status: completed
title: Implement channel registry and policy-driven routing
type: backend
complexity: high
dependencies:
  - task_01
---

# Task 02: Implement channel registry and policy-driven routing

## Overview

Implement the daemon-owned channel registry and routing logic that make channel identity and session continuity first-class runtime concerns. This task is responsible for instance lifecycle state, canonical routing-key construction, and ownership of the `routing key -> session` mapping.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST implement a daemon-owned registry in `internal/channels/` for creating, reading, updating, and resolving `ChannelInstance` and `ChannelRoute` records backed by the global DB layer from task 01.
2. MUST build canonical routing keys from the fixed base `scope + workspace_id? + channel_instance_id` plus the policy-controlled `peer`, `thread`, and `group` dimensions described in the TechSpec "RoutingKey" model.
3. MUST keep the authoritative `routing key -> session_id` mapping in the daemon rather than extension-local storage, including route reuse and route upsert behavior.
4. SHOULD provide explicit state transitions and validation for disabled, starting, ready, degraded, auth-required, and error channel instance states.
5. SHOULD define a per-platform dimension mapping contract that documents which platform concept maps to `peer_id`, `thread_id`, and `group_id`, so that cross-platform routing queries produce semantically consistent results.
</requirements>

## Subtasks
- [x] 2.1 Implement the registry APIs for channel instance lifecycle and route persistence
- [x] 2.2 Add routing-key builders and policy-aware route resolution helpers
- [x] 2.3 Add state-transition validation for instance status changes and route ownership updates
- [x] 2.4 Document per-platform dimension mapping contract for `peer_id`, `thread_id`, and `group_id`
- [x] 2.5 Add unit and integration tests for route reuse, scope isolation, and registry behavior

## Implementation Details

Follow the TechSpec sections "Core Interfaces", "RoutingKey", "ChannelRoute", and "Technical Considerations". This task should stop at core registry and routing behavior; it should not expose transports or stream session output yet.

### Relevant Files
- `internal/session/interfaces.go` — The registry will later coordinate with session lifecycle, so the existing session manager boundary matters here
- `internal/store/globaldb/global_db.go` — Registry methods will sit on top of the persistence layer introduced in task 01
- `internal/store/globaldb/global_db_workspace.go` — Existing workspace scoping patterns are the closest reference for scope-aware data access
- `internal/store/globaldb/global_db_test.go` — Existing database regression coverage is the right place to extend route and instance persistence tests

### Dependent Files
- `internal/extension/host_api.go` — Inbound message ingestion later resolves instances and routes through this registry
- `internal/daemon/boot.go` — Daemon composition will later inject the registry into channel runtime services
- `internal/api/httpapi/routes.go` — Transport surfaces later expose channel instances and route inspection based on registry behavior

### Related ADRs
- [ADR-005: Hybrid Channel Substrate with Extension-Based Platform Adapters](adrs/adr-005.md) — Confirms registry ownership belongs in the daemon substrate
- [ADR-006: Core-Owned Channel Registry, Scoped Instances, and Policy-Driven Routing](adrs/adr-006.md) — Defines the routing-key model and canonical route ownership this task must implement

## Deliverables
- A daemon-owned channel registry with scoped instance lifecycle and route ownership
- Policy-driven routing-key construction and route resolution helpers
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for route reuse and scope-aware isolation **(REQUIRED)**

## Tests
- Unit tests:
  - [x] A routing policy that enables only `peer` includes peer identity in the routing key and omits `thread` and `group`
  - [x] A routing policy that enables `peer` and `thread` generates distinct keys for the same peer across different threads
  - [x] Instance state transitions reject invalid moves such as reporting `ready` from a disabled instance without an enable path
  - [x] Route resolution for the same canonical key reuses the stored session ID instead of creating a second route record
- Integration tests:
  - [x] A global channel instance and a workspace-scoped instance with the same peer values resolve to different routing keys and route records
  - [x] Updating a route for an existing routing key replaces the stored session ownership without creating duplicate rows
  - [x] Listing routes for one channel instance returns only that instance's route set
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The daemon has a canonical registry and route-ownership layer for channels
- Session continuity can be keyed off daemon-built routing identity rather than extension-local storage
