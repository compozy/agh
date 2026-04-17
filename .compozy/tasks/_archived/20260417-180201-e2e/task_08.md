---
status: completed
title: Playwright harness for daemon-served browser E2E
type: test
complexity: high
dependencies:
  - task_01
---

# Task 08: Playwright harness for daemon-served browser E2E

## Overview

Create the browser E2E harness that runs Playwright against the daemon-served AGH web app rather than a Vite-only surface. This task adds the shared browser fixture surface, scripts, and execution model that later browser workflow tasks will consume independently.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
1. MUST add a Playwright-based browser E2E harness under `web/e2e/` with shared fixtures for launching or attaching to a daemon-served UI runtime.
2. MUST run browser E2E against the daemon-served web assets, not a separate Vite preview or proxy path.
3. MUST add browser diagnostics such as traces, screenshots, console logs, and network logs to the shared artifact model introduced by `task_01`.
4. MUST add project-local scripts and configuration required to run Playwright from the `web` workspace.
5. SHOULD keep this task focused on harness and fixture setup only; route-specific operator journeys belong in later browser tasks.
</requirements>

## Subtasks
- [x] 8.1 Add Playwright configuration and shared browser fixtures under `web/e2e/`.
- [x] 8.2 Wire browser execution to the daemon-served asset path rather than Vite preview.
- [x] 8.3 Add shared browser artifact capture for traces, screenshots, console logs, and network logs.
- [x] 8.4 Add `web` workspace scripts needed to install and run Playwright locally and in automation.
- [x] 8.5 Add focused harness tests or smoke checks proving the browser fixture boots and reaches the daemon-served shell.

## Implementation Details

See TechSpec sections "Browser Data Flow", "PR-Required Browser E2E", and "Technical Dependencies". This task establishes the browser lane foundation; later browser tasks should depend on it rather than redefining execution mode or fixtures.

### Relevant Files
- `web/embed.go` — confirms the shipped web assets are embedded and available to the daemon-served path.
- `internal/api/httpapi/static.go` — serves the embedded web assets and anchors the daemon-hosted browser execution mode.
- `internal/api/httpapi/server.go` — runtime HTTP server boot path that the browser fixture must target.
- `web/package.json` — currently has Vitest-only test scripts and no Playwright integration.
- `package.json` — root workspace scripts may need wrappers once browser E2E exists.
- `web/src/routes/_app.tsx` — useful shell-level route target for the first browser fixture smoke path.

### Dependent Files
- `web/playwright.config.ts` — new shared Playwright configuration for the web workspace.
- `web/e2e/fixtures/runtime.ts` — new browser-runtime fixture layer for daemon-hosted E2E.
- `web/e2e/fixtures/artifacts.ts` — new browser artifact capture helpers.
- `web/e2e/session-onboarding.spec.ts` — later session/browser task will consume the shared harness.
- `web/e2e/network.spec.ts` — later network/browser task will consume the shared harness.

### Related ADRs
- [ADR-002: Separate Runtime and Browser E2E Lanes](adrs/adr-002.md) — This task implements the browser lane as a distinct but coordinated E2E surface.
- [ADR-005: Keep PR-Required E2E On Shipped Surfaces and Use Tiered Execution](adrs/adr-005.md) — Browser E2E is limited to shipped, in-scope web surfaces and must run in a maintainable PR lane.

## Deliverables
- Playwright configuration and shared browser fixtures under `web/e2e/`
- Daemon-served browser execution mode with shared artifact capture
- `web` workspace scripts for Playwright installation and execution
- Unit tests with 80%+ coverage **(REQUIRED)**
- Integration tests for browser harness boot and daemon-served shell reachability **(REQUIRED)**

## Tests
- Unit tests:
  - [x] Browser fixture helpers compute daemon-hosted URLs and artifact paths consistently
  - [x] Browser artifact helpers persist traces, screenshots, console logs, and network logs in stable locations
  - [x] Script/config helpers enforce daemon-served execution mode rather than Vite preview assumptions
- Integration tests:
  - [x] Playwright harness boots against a daemon-served UI and reaches the authenticated application shell or onboarding shell
  - [x] Browser fixture captures a trace and screenshot bundle after a successful smoke path
  - [x] Browser fixture captures console and network diagnostics after a forced failure path
- Test coverage target: >=80%
- All tests must pass

## Success Criteria
- All tests passing
- Test coverage >=80%
- A reusable Playwright harness exists under `web/e2e/`
- Browser E2E is anchored to the daemon-served asset path, not a separate preview stack
- Later browser workflow tasks can depend on shared fixtures instead of re-implementing execution setup
