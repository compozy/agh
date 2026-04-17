---
status: pending
title: Host API parity for task read and aggregate surfaces
type: backend
complexity: medium
dependencies:
  - task_07
---

# Task 11: Host API parity for task read and aggregate surfaces

## Overview

Extend the extension Host API so embedded consumers can read the same richer task and aggregate surfaces introduced for the web and transport APIs. This task keeps the extension-facing task vocabulary aligned with the product surface instead of freezing it at today’s narrow CRUD and run commands.

<critical>
- ALWAYS READ `_techspec.md`, `task_07.md`, and the current Host API task surface before adding new methods
- REFERENCE TECHSPEC sections "Impact Analysis", "API Endpoints", and "Technical Considerations"
- FOCUS ON "WHAT" — extend Host API parity for task reads and aggregates, not a second custom task protocol
- MINIMIZE CODE — reuse the existing task-manager actor resolution and host API method registration patterns
- TESTS REQUIRED — host API method registration, request decoding, and payload shaping need coverage
- GREENFIELD: extensoes nao podem ficar presas no payload antigo de tasks se o produto ja expuser reads mais ricos e agregados
</critical>

<requirements>
- MUST extend the Host API with task-native point reads that are relevant to extension consumers, including richer task detail and run detail views where appropriate
- MUST expose the new aggregate read surfaces that extension consumers can benefit from, especially dashboard and inbox-style task views
- MUST keep Host API method names and request shapes aligned with the existing `tasks/*` naming pattern
- MUST reuse the same actor-context and error-mapping rules already applied to the current Host API task methods
- SHOULD avoid exposing frontend-specific concepts; Host API additions should stay task-domain or aggregate-read oriented
</requirements>

## Subtasks
- [ ] 11.1 Extend Host API contract/request shapes for the richer task reads and aggregate surfaces
- [ ] 11.2 Register the new `tasks/*` methods on the Host API dispatcher
- [ ] 11.3 Implement handler logic for the new task point reads and aggregates
- [ ] 11.4 Add Host API unit and integration tests for the expanded task surface

## Implementation Details

See TechSpec "Impact Analysis" and ADR-003/ADR-004. The Host API should remain an alternate entrypoint to the same task semantics, not a parallel protocol with its own subset of capabilities.

### Relevant Files
- `internal/extension/host_api_tasks.go` — current Host API task handlers and natural place for new task read methods
- `internal/extension/host_api.go` — Host API method registration table
- `internal/extension/contract/host_api.go` — shared Host API request/response contract definitions
- `internal/extension/host_api_test.go` — unit coverage for Host API task behavior
- `internal/extension/host_api_integration_test.go` — integration coverage for Host API method execution

### Dependent Files
- `internal/api/contract/tasks.go` — task_07 defines the richer payload vocabulary that Host API parity should align with
- `internal/task/manager.go` — Host API point reads depend on the richer manager-owned views
- `internal/observe/tasks.go` — Host API aggregate reads depend on the observer-backed dashboard and inbox models

### Related ADRs
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Host API should be able to consume task-native reads rather than only basic task summaries
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Host API parity should include the aggregate task reads that extension consumers can benefit from

## Deliverables
- Extended Host API methods for richer task reads and aggregate task surfaces
- Updated method registration and request/response contracts **(REQUIRED)**
- Unit tests with >=80% coverage for new Host API task methods **(REQUIRED)**
- Integration tests proving the new Host API task reads execute correctly **(REQUIRED)**
- Host API task parity that reflects the product’s richer task surface

## Tests
- Unit tests:
  - [ ] Host API dispatch registers the new task detail, run-detail, dashboard, inbox, and live-read methods
  - [ ] Request decoding and error mapping behave consistently with existing `tasks/*` methods for identifiers, filters, and missing resources
  - [ ] New Host API task reads return the expected richer payloads without requiring extension-side JSON reshaping
  - [ ] Aggregate method payloads remain aligned with observer-backed reads rather than list-derived fallbacks
- Integration tests:
  - [ ] Host API execution can fetch the richer task-detail and run-detail payloads against real task state and persisted runs
  - [ ] Host API aggregate methods return dashboard and inbox data aligned with the observer-backed task reads
  - [ ] Unknown method or missing-task calls return the expected host-facing error responses instead of transport-specific failures
  - [ ] Host API task surfaces remain aligned with the contract layer after code generation or interface updates
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80% for modified Host API task files
- Extension consumers can access richer task read and aggregate surfaces through the Host API
- The Host API task surface no longer lags behind the documented product-level task capabilities
