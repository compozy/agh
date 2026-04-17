---
status: completed
title: HTTP and UDS transport parity scenarios
type: test
complexity: medium
dependencies:
  - task_01
  - task_03
  - task_04
  - task_05
  - task_06
---

# Task 07: HTTP and UDS transport parity scenarios

## Overview

Add the focused transport-level scenarios that prove the daemon truth established in the runtime lane is visible through HTTP and UDS where the product depends on those transports. This task should stay narrow: approval, webhook ingress, CLI/UDS parity, and projection reads that need explicit transport coverage.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add explicit HTTP-focused coverage for approval-sensitive and webhook-ingress flows that remain transport-asymmetric today.
2. MUST add explicit UDS-focused coverage for operator and CLI-visible flows that need parity checks against the runtime truth scenarios.
3. MUST consume the shared runtime harness from `task_01` rather than preserving duplicated HTTP and UDS fixture logic.
4. MUST treat these tests as transport projections over daemon state, not as replacements for the composition-root runtime scenarios.
5. SHOULD document and preserve the current UDS approval behavior, which exposes the route but still returns `501 Not Implemented`.
</requirements>

## Subtasks
- [x] 7.1 Refactor HTTP and UDS integration suites to consume the shared runtime harness.
- [x] 7.2 Add HTTP coverage for approval-sensitive and webhook-ingress flows.
- [x] 7.3 Add UDS coverage for CLI-visible operator parity on in-scope runtime scenarios.
- [x] 7.4 Add transport-level assertions that explain parity gaps without re-testing protocol truth already covered in the daemon lane.
- [x] 7.5 Add focused regression checks for the current UDS approval asymmetry and documented projection behavior.

## Implementation Details

See TechSpec sections "Public Surfaces Exercised", "Technical Dependencies", and "Development Sequencing". This task follows the composition-root runtime work; it does not replace it.

### Relevant Files
- `internal/api/httpapi/httpapi_integration_test.go` — primary HTTP transport integration suite.
- `internal/api/httpapi/bridges_integration_test.go` — existing HTTP bridge transport coverage that may need shared harness convergence.
- `internal/api/httpapi/routes.go` — source of HTTP transport exposure for sessions, automation, bridges, and webhooks.
- `internal/api/udsapi/udsapi_integration_test.go` — primary UDS transport integration suite.
- `internal/api/udsapi/routes.go` — source of UDS transport exposure and current approval-route behavior.
- `internal/cli/client.go` — CLI transport surface that later runtime/browser lane wiring depends on for parity.

### Dependent Files
- `internal/testutil/e2e/runtime_harness.go` — must be consumed by both HTTP and UDS suites.
- `internal/api/httpapi/helpers_integration_test.go` — likely updated to stop duplicating runtime boot logic.
- `internal/api/udsapi/bridges_integration_test.go` — likely updated to consume the shared fixture surface.
- `Makefile` — later command wiring needs a stable transport-aware runtime target set.
- `magefile.go` — later Mage targets need a stable transport-aware runtime target set.

### Related ADRs
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) — Transport tests supplement the composition-root runtime lane rather than replacing it.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Transport assertions should still read meaningful product surfaces.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Approval and webhook ingress are important PR-required transport-specific proofs.

## Deliverables
- Shared-harness-based HTTP integration coverage for approval-sensitive and webhook-ingress flows
- Shared-harness-based UDS integration coverage for CLI/operator parity on in-scope flows
- Narrow parity assertions that explain transport differences without duplicating daemon-truth scenarios
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for HTTP/UDS transport parity and asymmetry behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Transport fixture helpers build HTTP and UDS clients from the shared runtime harness consistently
  - [x] Approval/parity assertion helpers report the documented UDS `501 Not Implemented` behavior clearly
  - [x] Webhook and projection helpers avoid duplicating daemon-truth assertions already covered elsewhere
- Integration tests:
  - [x] HTTP approval flow succeeds against a real daemon session when a permission request is pending
  - [x] UDS session approval route surfaces the current documented `501 Not Implemented` behavior without hiding the gap
  - [x] HTTP and UDS projection reads for an in-scope runtime scenario reflect the same persisted backend truth where parity is expected
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- HTTP and UDS suites both consume the shared runtime harness
- Approval, webhook ingress, and CLI-visible parity gaps are explicitly tested and documented
- Transport suites remain narrow supplements to the daemon runtime lane
