---
status: completed
title: Runtime bridge ingress and extension subprocess scenarios
type: test
complexity: high
dependencies:
  - task_01
  - task_02
  - task_04
---

# Task 05: Runtime bridge ingress and extension subprocess scenarios

## Overview

Add the runtime E2E scenarios that prove bridge ingress, agent processing, bridge delivery, and extension Host API behavior through real subprocess boundaries. This task reuses the existing extension bridge harness where it helps, but keeps daemon-visible runtime truth in the composition-root test lane.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a runtime E2E scenario that ingests a real bridge event and proves route creation or reuse, session reuse or creation, agent processing, and delivery progression.
2. MUST add a runtime E2E scenario that uses a real extension subprocess and proves one bridge-related or automation/task-related Host API flow through the extension boundary.
3. MUST reuse `internal/extensiontest/` and existing extension integration harnesses where they reduce duplication, but MUST NOT move daemon scenario logic out of the composition-root runtime lane.
4. MUST assert bridge behavior through bridge-specific surfaces such as health snapshots, delivery state, route state, secret bindings where relevant, and provider-side effect logs or markers.
5. SHOULD keep real subprocess delivery and Host API behavior in the same task because they share the same extension/bridge harness boundary.
</requirements>

## Subtasks
- [x] 5.1 Extend runtime fixtures with bridge-enabled and extension-enabled daemon setup.
- [x] 5.2 Implement bridge ingress and downstream delivery progression coverage in `internal/daemon`.
- [x] 5.3 Reuse and extend `internal/extensiontest/` for a real extension subprocess Host API flow.
- [x] 5.4 Add bridge and provider-side effect artifacts to the shared failure manifest.
- [x] 5.5 Add focused regression coverage for session reuse and delivery side effects.

## Implementation Details

See TechSpec sections "PR-Required Runtime E2E", "Integration Points", and "Combined-Flow Follow-Up". This task should reuse the marker- and conformance-based bridge harnesses where possible, but the top-level pass/fail signal still belongs to the real daemon scenario.

### Relevant Files
- `internal/daemon/daemon_integration_test.go` — composition-root location for bridge ingress and delivery scenarios.
- `internal/extensiontest/bridge_adapter_harness.go` — reusable bridge adapter harness surface.
- `internal/extensiontest/bridge_adapter_harness_integration_test.go` — existing integration coverage that can inform runtime subprocess setup.
- `internal/extension/bridge_delivery_integration_test.go` — real bridge-delivery subprocess behavior already covered at package level.
- `internal/extension/host_api_integration_test.go` — existing Host API integration behavior that informs extension-boundary assertions.
- `internal/extension/manager_integration_test.go` — real extension subprocess manager behavior relevant to runtime boot.

### Dependent Files
- `internal/testutil/e2e/runtime_harness.go` — must support bridge-enabled and extension-enabled daemon fixtures.
- `internal/testutil/e2e/artifacts.go` — must add bridge health, route, delivery, and provider-call artifact capture.
- `internal/extensiontest/bridge_conformance_matrix.go` — may absorb reusable helper coverage without becoming the source of truth.
- `web/e2e/bridges.spec.ts` — later browser bridges flow depends on the runtime bridge truth established here.
- `internal/api/httpapi/bridges_integration_test.go` — later transport parity work can read the same bridge state through HTTP.

### Related ADRs
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) — Bridge ingress and subprocess delivery must still be proven from the real runtime graph.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Bridge health, routes, and provider-side effects are the primary assertion surfaces.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Bridge flows are PR-required on shipped surfaces, while heavier external-provider flows remain later-tier.

## Deliverables
- Composition-root runtime E2E for bridge ingress, agent handling, and delivery progression
- Reused and extended extension subprocess harness coverage for real Host API behavior
- Artifact capture for bridge health, route state, delivery state, and provider-side effects
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for bridge ingress and extension subprocess behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Bridge artifact helpers persist route, health, and provider-call snapshots in stable locations
  - [x] Runtime fixture seeding can register bridge-enabled and extension-enabled daemon state without hidden assumptions
  - [x] Assertion helpers differentiate session reuse from new-session bridge handling
- Integration tests:
  - [x] Bridge ingress triggers real route reuse or creation and records downstream delivery progression
  - [x] Extension subprocess flow reaches the real Host API boundary and produces a visible bridge or automation/task side effect
  - [x] Provider-side effect logs or markers align with the daemon-observed bridge state for the same scenario
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Runtime E2E proves bridge ingress and extension subprocess flows through real daemon boundaries
- Existing extension harnesses are reused without absorbing daemon-truth responsibilities
- Browser bridge operator tests can rely on this runtime lane for backend truth
