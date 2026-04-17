---
status: completed
title: Shared E2E runtime harness and artifact plumbing
type: test
complexity: high
dependencies: []
---

# Task 01: Shared E2E runtime harness and artifact plumbing

## Overview

Create the shared runtime test harness that all daemon and browser E2E lanes will build on. This task establishes isolated daemon boot, seeded workspace/config helpers, public-surface clients, and mandatory artifact capture so later tasks can add scenarios without re-implementing infrastructure.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST create a shared `internal/testutil/e2e/` package that boots an isolated daemon runtime with isolated `AGH_HOME`, isolated workspace state, and reusable HTTP/UDS/CLI clients.
2. MUST provide artifact-manifest and artifact-capture helpers for transcript, events, network, automation, task, bridge, environment, and browser diagnostics as defined in the TechSpec "Data Models" and "Monitoring and Observability" sections.
3. MUST centralize seeded config and workspace setup so `internal/daemon`, `internal/api/httpapi`, and `internal/api/udsapi` integration suites can consume the same runtime fixture instead of duplicating boot logic.
4. MUST preserve the current real daemon startup path; the harness may orchestrate boot and read public surfaces, but it MUST NOT re-implement domain behavior outside the daemon.
5. SHOULD add small extensions to existing generic test utilities only when they remove duplication across the new E2E fixture surface.
</requirements>

## Subtasks
- [x] 1.1 Define the shared runtime harness API and artifact manifest contract for E2E suites.
- [x] 1.2 Add isolated daemon boot, seeded config, workspace creation, and public-client helpers under `internal/testutil/e2e/`.
- [x] 1.3 Add artifact capture helpers for required domain snapshots and failure diagnostics.
- [x] 1.4 Migrate one existing integration fixture path to prove the shared harness replaces duplicated setup cleanly.
- [x] 1.5 Add focused tests that lock the harness lifecycle and artifact-manifest behavior.

## Implementation Details

See TechSpec sections "Core Interfaces", "Data Models", "Development Sequencing", and "Monitoring and Observability". This task is the prerequisite for every later runtime or browser E2E task, so keep the API narrow: boot, seed, register clients, and collect artifacts.

### Relevant Files
- `internal/testutil/testutil.go` — existing shared test helpers that may absorb small common utilities.
- `internal/daemon/daemon_integration_test.go` — current composition-root integration suite that needs reusable runtime boot support.
- `internal/api/httpapi/httpapi_integration_test.go` — currently carries its own runtime fixture patterns that should converge on the shared harness.
- `internal/api/udsapi/udsapi_integration_test.go` — currently carries a parallel integration harness that should stop duplicating daemon boot concerns.
- `internal/config/agent.go` — agent-definition helpers and validation rules the harness must respect when seeding runtime state.
- `internal/config/provider.go` — provider defaults and resolution rules the seeded config must keep compatible with.

### Dependent Files
- `internal/testutil/e2e/runtime_harness.go` — new shared harness entrypoint for daemon boot and public-surface clients.
- `internal/testutil/e2e/artifacts.go` — new artifact-manifest and snapshot-capture helpers.
- `internal/testutil/e2e/config_seed.go` — new seeded config and workspace/runtime fixture helpers.
- `internal/api/httpapi/helpers_integration_test.go` — likely consumer of the shared harness once duplication is removed.
- `internal/api/udsapi/udsapi_integration_test.go` — likely consumer of the shared harness once duplication is removed.

### Related ADRs
- [ADR-001: Mock ACP Through a Temporary Agent Definition](adrs/adr-001.md) — The harness must support real daemon agent resolution rather than custom driver injection.
- [ADR-003: Run Cross-System Runtime E2E From the Composition Root](adrs/adr-003.md) — Shared boot helpers exist to support composition-root runtime scenarios.
- [ADR-004: Assert Through Domain-Specific Product Surfaces](adrs/adr-004.md) — Artifact capture must reflect domain-specific assertion surfaces.

## Deliverables
- Shared `internal/testutil/e2e/` runtime harness package with isolated daemon boot and public clients
- Stable artifact-manifest and artifact-capture helpers for E2E failures
- Refactored integration helper usage proving the shared harness can replace duplicated runtime setup
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for harness boot and artifact capture **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Runtime harness creates isolated home, database, and artifact directories for each test run
  - [x] Artifact manifest includes only the captured domain surfaces and writes stable paths for failures
  - [x] Seeded config generation preserves provider and agent validation invariants from the live config rules
- Integration tests:
  - [x] Shared harness boots a real daemon and returns working HTTP and UDS clients against the started runtime
  - [x] Shared harness can create a seeded workspace and read back a public daemon surface without package-local boot duplication
  - [x] Artifact capture persists transcript and event snapshots after a forced failing scenario
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- A reusable runtime harness exists under `internal/testutil/e2e/`
- At least one existing integration suite consumes the shared harness instead of local duplicate boot logic
- Required artifact capture is available for later runtime and browser E2E tasks
