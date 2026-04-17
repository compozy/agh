---
status: completed
title: E2E commands, Mage targets, and tiered lane wiring
type: infra
complexity: high
dependencies:
  - task_07
  - task_09
  - task_10
  - task_11
  - task_12
---

# Task 13: E2E commands, Mage targets, and tiered lane wiring

## Overview

Wire the completed runtime and browser scenarios into explicit project entrypoints so E2E execution matches the tiered strategy in the TechSpec. This task defines the repo-local command surface for `runtime`, `web`, combined, and nightly lanes without collapsing everything back into the existing broad `test-integration` target.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add explicit repo-local entrypoints for `test-e2e-runtime`, `test-e2e-web`, `test-e2e`, and `test-e2e-nightly` that reflect the tiered execution model from the TechSpec.
2. MUST add matching Mage targets and package-script wrappers where appropriate so local and automation invocations are consistent.
3. MUST keep default PR-required E2E separate from the broad existing `test-integration` sweep and from credentialed nightly scenarios.
4. MUST document or encode the daemon-served browser execution mode and runtime/browser split in the new command wiring.
5. SHOULD avoid introducing CI assumptions that depend on in-repo workflow files if the current checkout does not manage CI definitions there.
</requirements>

## Subtasks
- [x] 13.1 Add Makefile targets for runtime, web, combined, and nightly E2E lanes.
- [x] 13.2 Add matching Mage targets that preserve the same lane semantics.
- [x] 13.3 Add root and `web` script wrappers needed for local invocation and tooling integration.
- [x] 13.4 Ensure lane definitions exclude credentialed nightly coverage from default PR-required targets.
- [x] 13.5 Add focused command-level regression checks that prove each target runs the intended slice.

## Implementation Details

See TechSpec sections "Development Sequencing", "Technical Dependencies", and "Known Risks". The main risk here is accidentally hiding the new E2E lanes inside the existing integration umbrella, which would undermine the whole tiering strategy.

### Relevant Files
- `Makefile` — current top-level entrypoints stop at `test-integration` and need explicit E2E lane targets.
- `magefile.go` — current Mage targets stop at `TestIntegration` and need explicit runtime/web/nightly E2E orchestration.
- `package.json` — root workspace script wrappers may need matching E2E commands.
- `web/package.json` — browser lane scripts should align with the Playwright harness rather than only Vitest.
- `.compozy/tasks/e2e/_techspec.md` — source of truth for the required lane names and tiered execution semantics.
- `internal/api/httpapi/static.go` — reminds the command wiring that browser E2E must target the daemon-served asset path.

### Dependent Files
- `web/playwright.config.ts` — consumed by browser-lane targets and wrappers.
- `web/e2e/` — browser E2E scenario tree that the new web-lane commands must run.
- `internal/daemon/daemon_integration_test.go` — runtime-lane command filters or package selectors will target these scenarios.
- `internal/api/httpapi/httpapi_integration_test.go` — transport parity scenarios become part of the runtime lane selection.
- `internal/api/udsapi/udsapi_integration_test.go` — transport parity scenarios become part of the runtime lane selection.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — The command surface must preserve the lane split rather than flatten it.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — This task directly implements the tiered runtime/web/nightly execution model.

## Deliverables
- New Makefile E2E targets for runtime, web, combined, and nightly lanes
- Matching Mage targets for the same E2E lanes
- Root and workspace scripts aligned with the new runtime/web/nightly command surface
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for lane selection and command wiring behavior **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Command-selection helpers map runtime, web, combined, and nightly lanes to the intended package/script set
  - [x] Lane definitions exclude credentialed nightly coverage from default PR-required entrypoints
  - [x] Browser-lane command wiring preserves daemon-served execution mode assumptions
- Integration tests:
  - [x] `make test-e2e-runtime` runs the intended runtime scenario slice without sweeping unrelated integration packages
  - [x] `make test-e2e-web` runs the Playwright/browser lane without invoking nightly-only coverage
  - [x] Combined and nightly entrypoints invoke the expected underlying Mage and script wiring
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- The repo has explicit runtime, web, combined, and nightly E2E entrypoints
- Default PR-required E2E no longer depends on the broad `test-integration` target semantics
- Local and automation-facing command surfaces consistently reflect the TechSpec lane model
