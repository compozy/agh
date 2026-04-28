---
status: completed
title: Combined-flow and credentialed nightly E2E follow-up
type: test
complexity: critical
dependencies:
  - task_13
---

# Task 14: Combined-flow and credentialed nightly E2E follow-up

## Overview

Add the later-tier E2E scenarios that combine multiple runtime domains and introduce credentialed or externally dependent coverage. This task intentionally sits after the base runtime, browser, and command-lane work so the nightly lane can build on stable foundations instead of becoming the first place system complexity appears.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add later-tier E2E scenarios for combined flows such as automation to network reply, bridge ingress to environment tool to bridge delivery, and automation task delegation to resumed session behavior.
2. MUST add credentialed or externally dependent E2E coverage for Daytona-backed environment flows and other real-provider integrations only in the nightly or opt-in lane.
3. MUST keep these scenarios out of the default PR-required runtime and browser targets introduced by `task_13`.
4. MUST capture rich artifacts for multi-domain failures, including runtime artifacts and browser traces where combined browser-visible flows are involved.
5. SHOULD keep nightly scenarios aligned with the same shared harnesses rather than creating a separate one-off test stack.
</requirements>

## Subtasks
- [x] 14.1 Add combined multi-domain runtime scenarios on top of the completed base harnesses and runtime slices.
- [x] 14.2 Add browser-observed combined flows where the shipped UI is part of the operator path.
- [x] 14.3 Add credentialed Daytona or external-provider scenarios to the nightly or opt-in lane only.
- [x] 14.4 Expand artifact capture and failure diagnostics for multi-domain and external-provider scenarios.
- [x] 14.5 Add focused nightly-lane regression checks that prove these scenarios stay out of default PR-required execution.

## Implementation Details

See TechSpec sections "Nightly Or Credentialed E2E", "Combined-Flow Follow-Up", and "Known Risks". This task is explicitly later-tier work; do not let it leak back into the default runtime or browser lane.

### Relevant Files
- `internal/sandbox/daytona/provider_integration_test.go` — credentialed environment path that belongs to the nightly or opt-in lane.
- `internal/sandbox/daytona/ssh_validation_test.go` — validation reference for Daytona transport behavior in external-provider coverage.
- `internal/daemon/daemon_integration_test.go` — base location for combined runtime flows that cross automation, network, task, bridge, and environment boundaries.
- `web/e2e/` — browser-visible combined flows should extend the existing Playwright harness rather than introducing a second browser stack.
- `Makefile` — nightly lane wiring introduced in `task_13` must pick up these scenarios without polluting default targets.
- `magefile.go` — nightly and opt-in command orchestration must reflect the same later-tier scope.

### Dependent Files
- `internal/testutil/e2e/artifacts.go` — may need richer multi-domain artifact capture for combined-flow failures.
- `internal/testutil/e2e/runtime_harness.go` — may need opt-in credential and environment switches for nightly scenarios.
- `web/e2e/fixtures/runtime.ts` — may need combined browser/runtime fixture support for nightly browser-observed flows.
- `internal/sandbox/daytona/VALIDATION.md` — likely reference or update point when live credentialed scenarios are operationalized.
- `package.json` — root script wrappers may need an explicit nightly or opt-in command surface.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — Combined flows may span both lanes, but the split still holds.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Multi-domain assertions must stay meaningful and artifact-rich.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — This task directly implements the later-tier, nightly, and credentialed scope.

## Deliverables
- Combined multi-domain runtime and browser-observed E2E scenarios on top of the completed base harnesses
- Credentialed Daytona and external-provider E2E scenarios isolated to nightly or opt-in execution
- Expanded artifact capture for multi-domain and external-provider failures
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for combined-flow and nightly-lane behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Nightly-lane fixture switches and artifact helpers isolate credentialed configuration from default PR-required execution
  - [x] Combined-flow artifact capture preserves enough cross-domain context to diagnose failures without rerunning immediately
  - [x] Lane-selection helpers prevent nightly scenarios from leaking into default runtime or browser commands
- Integration tests:
  - [x] Automation-to-network or task-delegation combined runtime flow completes with the expected cross-domain visible state
  - [x] Bridge-to-environment combined flow produces the expected bridge-visible and environment-visible outcomes
  - [x] Credentialed Daytona or external-provider scenario runs only behind the nightly or opt-in lane and leaves usable diagnostics
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Combined-flow E2E exists without destabilizing the default PR-required lanes
- Credentialed Daytona and external-provider coverage is available only in nightly or opt-in execution
- Multi-domain failures produce enough artifacts to be diagnosable from one failed run
