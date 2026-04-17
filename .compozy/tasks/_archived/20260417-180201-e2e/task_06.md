---
status: completed
title: Runtime environment and sandbox scenarios
type: test
complexity: high
dependencies:
  - task_01
  - task_02
  - task_04
---

# Task 06: Runtime environment and sandbox scenarios

## Overview

Add the runtime E2E scenarios that prove environment-owned tool execution and sandbox restrictions through the real daemon and local environment provider. This task keeps PR-required coverage on deterministic local environment flows while explicitly leaving Daytona and other credentialed providers to the later nightly lane.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a runtime E2E scenario that runs a session in the configured environment and proves allowed tool execution plus blocked-operation behavior under sandbox rules.
2. MUST assert environment behavior through persisted environment metadata, tool-host side effects, and visible failure signals rather than transcript-only checks.
3. MUST use the local environment provider for PR-required runtime coverage and keep Daytona or other external providers out of the default lane.
4. MUST integrate environment diagnostics into the shared artifact model so failed sandbox scenarios leave actionable metadata behind.
5. SHOULD reuse existing provider-test suites and daemon environment integration patterns instead of inventing a second environment harness.
</requirements>

## Subtasks
- [x] 6.1 Add environment-aware runtime fixture setup for local provider scenarios.
- [x] 6.2 Implement the allowed tool execution runtime scenario under the real environment boundary.
- [x] 6.3 Implement the blocked-operation runtime scenario and capture the resulting restriction surfaces.
- [x] 6.4 Add environment metadata and tool-host side-effect artifacts to the shared failure manifest.
- [x] 6.5 Add focused regression coverage for local-provider cleanup and session-environment state persistence.

## Implementation Details

See TechSpec sections "PR-Required Runtime E2E", "Daemon-Only E2E In Current Product", and "Technical Dependencies". This task should remain local-provider focused; the credentialed Daytona lane belongs in the later nightly follow-up task.

### Relevant Files
- `internal/daemon/environment_reconcile_integration_test.go` — existing daemon/environment integration patterns that can inform the runtime E2E slice.
- `internal/environment/local/provider_test.go` — current local provider contract behavior relevant to PR-required runtime coverage.
- `internal/environment/providertest/suite.go` — reusable provider contract suite that should remain aligned with runtime E2E expectations.
- `internal/environment/registry.go` — environment registry behavior used by the shared daemon boot path.
- `internal/session/manager_start.go` — environment preparation and runtime handoff path that runtime E2E will exercise.
- `internal/environment/daytona/provider_integration_test.go` — explicit non-PR-required reference point that must remain outside the default lane.

### Dependent Files
- `internal/testutil/e2e/runtime_harness.go` — must support local environment seeding and environment-aware daemon boot.
- `internal/testutil/e2e/artifacts.go` — must capture session environment metadata and tool-host diagnostics.
- `internal/daemon/daemon_integration_test.go` — likely home for the new environment and sandbox runtime scenarios.
- `internal/api/httpapi/httpapi_integration_test.go` — later transport parity coverage may read environment-sensitive state through public HTTP surfaces.
- `Makefile` — later lane wiring must keep local-provider E2E in default runtime targets and external-provider flows out of them.

### Related ADRs
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) — Environment and sandbox proof belongs to the composition-root runtime lane.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Environment metadata and tool-host side effects are the primary assertion surfaces.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Local environment flows are PR-required; Daytona is later-tier and credentialed.

## Deliverables
- Runtime E2E for allowed local-provider tool execution
- Runtime E2E for blocked sandbox operation behavior
- Shared artifact support for environment metadata and tool-host diagnostics
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for local-provider environment and sandbox behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Environment artifact helpers persist session environment metadata and tool-host diagnostics consistently
  - [x] Local-provider runtime fixture seeding enables environment-backed sessions without leaking state across tests
  - [x] Assertion helpers distinguish allowed execution from blocked sandbox outcomes without transcript-only checks
- Integration tests:
  - [x] Local-provider session can execute an allowed tool action and persist environment metadata through the real daemon flow
  - [x] Blocked sandbox operation produces the expected runtime failure signal and diagnostic artifacts
  - [x] Environment metadata remains readable from public surfaces after session stop or failure
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- Runtime E2E proves local environment and sandbox behavior through the real daemon boundary
- Default PR-required E2E remains free of external-provider credential dependency
- Later nightly Daytona work can build on an already-proven environment artifact model
