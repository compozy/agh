---
status: pending
title: Observer-backed dashboard read models
type: backend
complexity: medium
dependencies:
  - task_02
---

# Task 05: Observer-backed dashboard read models

## Overview

Build the aggregate dashboard view for tasks on top of the existing observer layer. This task keeps queue depth, health, totals, and active-run summaries on the read side, where they belong, instead of bloating task-manager point reads with dashboard-specific aggregation.

<critical>
- ALWAYS READ `_techspec.md`, ADRs, and `analysis_dashboard.md` before shaping aggregate reads
- REFERENCE TECHSPEC sections "Data Models", "Observer-backed read-model endpoints", and "Monitoring and Observability"
- FOCUS ON "WHAT" — define dashboard aggregates and freshness expectations, not HTTP registration
- MINIMIZE CODE — reuse the observer’s current summary, metrics, and health computations wherever they already exist
- TESTS REQUIRED — totals, health, queue depth, and active-run cards need deterministic coverage
- GREENFIELD: dashboard e aggregate health ficam no read side; nao infle `GET /api/tasks` para simular cards de observabilidade
</critical>

<requirements>
- MUST add a dashboard read model that combines task summary, metrics, health, queue depth, and recent active-run information for the Paper dashboard
- MUST reuse existing observer computations where possible instead of duplicating summary logic in a second service
- MUST keep dashboard freshness and stale-read behavior explicit enough for UI loading and warning states
- MUST expose aggregate fields shaped for cards and charts rather than forcing the frontend to derive them from raw summary buckets
- SHOULD keep lower-level metrics reusable if the dashboard later shares pieces with other operator views
</requirements>

## Subtasks
- [ ] 5.1 Define the observer-backed dashboard view type and aggregation boundaries
- [ ] 5.2 Reuse summary, metrics, and health queries to assemble dashboard-ready output
- [ ] 5.3 Add queue-depth and active-run shaping suitable for the Paper cards
- [ ] 5.4 Add observer tests for totals, freshness, and edge-case aggregation

## Implementation Details

See TechSpec sections "Data Models", "Observer-backed read-model endpoints", and ADR-004. The goal is a dashboard-specific read model that is still grounded in observer-owned calculations, not a client-side reconstruction of summary buckets.

### Relevant Files
- `internal/observe/tasks.go` — existing task summary, metrics, and health queries that should back the dashboard view
- `internal/observe/tasks_test.go` — core observer summary/metrics test coverage
- `internal/observe/tasks_integration_test.go` — real-store observer validation for aggregate task reads
- `internal/observe/health.go` — useful reference for how aggregate health payloads are currently shaped
- `.compozy/tasks/tasks-ui/analysis/analysis_dashboard.md` — identifies the dashboard transport and shaping gaps to close

### Dependent Files
- `internal/api/core/interfaces.go` — task_08 will depend on a stable dashboard read surface
- `internal/api/contract/contract.go` — task_07 may need new observe-task aggregate payload wrappers
- `internal/api/spec/spec.go` — task_07 will document the dashboard endpoint
- `web/src/systems/tasks/hooks/use-task-dashboard.ts` — task_13 and task_16 will consume this aggregate read

### Related ADRs
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Assigns dashboard aggregation to the observer layer instead of task-manager point reads

## Deliverables
- Dashboard read model assembled from observer-owned task aggregates
- Queue depth, health, totals, and active-run dashboard shaping
- Unit tests with >=80% coverage for dashboard aggregation logic **(REQUIRED)**
- Integration tests proving dashboard output against durable state **(REQUIRED)**
- Clear separation between point reads and aggregate dashboard data

## Tests
- Unit tests:
  - [ ] Dashboard totals and card values are derived consistently from summary and metrics buckets across ready, running, blocked, failed, and completed task mixes
  - [ ] Queue-depth, oldest-queue age, and backlog-warning calculations surface the expected values for both normal and degraded states
  - [ ] Active-run cards pick the correct recent runs and expose stable status or health summaries when multiple runs are in flight
  - [ ] Dashboard shaping handles zero-data, partially populated, and stale-projection states without leaking raw observer internals
  - [ ] Scope or workspace filtering preserves totals consistency and does not cross-contaminate aggregate buckets
- Integration tests:
  - [ ] Observer integration tests return dashboard payloads aligned with persisted task and run state across multiple status buckets
  - [ ] Queue depth, health warnings, and oldest-item metrics remain correct after persisted task transitions and observer refreshes
  - [ ] Active-run summaries stay correct when tasks span multiple statuses, owners, and network channels in real stored data
  - [ ] Dashboard reads are satisfied from the aggregate endpoint without requiring client-side joins across lower-level observer outputs
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified observer task files
- The dashboard has a dedicated observer-backed read model ready for transport and UI work
- The frontend no longer needs to derive dashboard cards from low-level raw task queries
