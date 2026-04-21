---
status: completed
title: Rewrite Daemon status and home dashboard
type: frontend
complexity: low
dependencies:
  - task_13
  - task_14
---

# Task 29: Rewrite Daemon status and home dashboard

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/daemon/**` (connection/health status indicator) and the index route `/` (home dashboard) on top of `@agh/ui` primitives. The home dashboard becomes a small landing page summarizing daemon health and high-level workspace metrics, replacing the current "Select a session to begin" empty screen. **These screens are NOT in the `docs/design/web-inspiration` mock** — per ADR-004 ("non-mocked screens derivation rule") + TechSpec Phase 5, visuals derive from `DESIGN.md` patterns using `PageHeader`, `Metric`, `StatusDot`, `ConnectionIndicator`, and `Section` only.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `connection-status.tsx` as a thin wrapper over `@agh/ui` `ConnectionIndicator`, removing the current `@/components/design-system/connection-indicator` import.
- MUST rewrite `routes/_app/index.tsx` as a home dashboard composing `@agh/ui` `PageHeader`, `Metric` grid, `StatusDot`, `ConnectionIndicator`, and `Section`; the current `Terminal` empty shell is deleted.
- MUST render at least the following metrics on the dashboard: active sessions count, workspaces count, agents count, daemon uptime — all pulled from existing `useDaemonHealth`/workspace/agent hooks via a new route-level view-model hook.
- MUST surface daemon status via `StatusDot` tones (`success` / `warning` / `danger` / `neutral`) + a short label, and the persistent connection pill via `ConnectionIndicator`.
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*`.
- MUST keep the route path `/` (index under `/_app/`) unchanged.
- MUST NOT introduce visuals that are not already documented in `DESIGN.md` or covered by existing `@agh/ui` primitives — per ADR-004 non-mocked rule.
- SHOULD keep the dashboard purely presentational (no side effects); data fetching lives in a route-level hook.
</requirements>

## Subtasks

- [x] 29.1 Audit `systems/daemon/components/**`, `systems/daemon/hooks/**`, and `routes/_app/index.tsx` to catalogue existing behavior and data sources.
- [x] 29.2 Rewrite `connection-status.tsx` as a one-line composition over `@agh/ui` `ConnectionIndicator`.
- [x] 29.3 Build a small route-level hook that aggregates daemon health, workspace count, agent count, and active session count.
- [x] 29.4 Rewrite `routes/_app/index.tsx` on `PageHeader` + `Section` + `Metric` grid + `StatusDot` + `ConnectionIndicator`; add loading, error, and degraded-daemon states.
- [x] 29.5 Update or add Storybook stories for the dashboard: healthy, degraded, disconnected, empty-workspace, loading, error.
- [x] 29.6 Regenerate Playwright visual snapshot baselines for `/` (healthy, degraded, disconnected, empty).
- [x] 29.7 Run `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.

## Implementation Details

Follow TechSpec "Impact Analysis" row `web/src/systems/workspace/**, daemon/**, agent/**` — "modified (visual, derived)". ADR-004 Phase 5 requires daemon + home dashboard to be derived from the system pattern using Sidebar + PageHeader + Metric + Empty consistently; task 28 covers workspace + agent, this task covers daemon + home.

### Relevant Files

- `web/src/systems/daemon/components/connection-status.tsx` — rewrite target.
- `web/src/systems/daemon/components/stories/*.stories.tsx` — update for new primitive.
- `web/src/routes/_app/index.tsx` — rewrite target (home dashboard).
- `web/src/systems/daemon/hooks/**` — unchanged consumers of the new dashboard.
- `packages/ui/src/components/{page-header,metric,status-dot,connection-indicator,section}.tsx` — primitives consumed.

### Dependent Files

- `web/e2e/__snapshots__/` — new baselines for `/` route states.
- `web/src/routes/_app/-index.test.tsx` (new) — route test asserting metric cards, status dot, and connection indicator.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout — non-mocked screens derivation rule](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `connection-status.tsx` on `@agh/ui` `ConnectionIndicator`.
- Rewritten `routes/_app/index.tsx` home dashboard composed from `@agh/ui` primitives only.
- Route-level hook that aggregates daemon + workspace + agent + session counts.
- Storybook stories for healthy / degraded / disconnected / empty / loading / error dashboard states.
- Playwright snapshot baselines for `/` healthy, degraded, disconnected, and empty **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for status transitions (healthy → degraded → disconnected) **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] `ConnectionStatus` forwards `status` prop to `ConnectionIndicator` and renders the matching tone.
  - [x] Home dashboard renders `PageHeader` with title "Home" and the daemon `StatusDot` tone mapping (`success` for healthy, `warning` for degraded, `danger` for disconnected, `neutral` for unknown).
  - [x] Home dashboard renders a `Metric` grid with active sessions, workspaces, agents, and uptime values.
  - [x] Loading state renders skeletons for each `Metric` card.
  - [x] Error state renders an `Empty` (or equivalent) region with the error message.
  - [x] Disconnected state renders the `ConnectionIndicator` in `disconnected` tone and a short recovery hint.
- Integration tests:
  - [x] Storybook `play()` transitions the story from healthy → degraded → disconnected and asserts `StatusDot` + `ConnectionIndicator` tones update.
  - [x] Playwright visual snapshot match for `/` healthy, degraded, disconnected, and empty states.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `web/src/systems/daemon/**` and `web/src/routes/_app/index.tsx` contain zero imports from `@/components/ui/*` or `@/components/design-system/*`.
- `/` renders only through `@agh/ui` primitives + domain hooks.
- Playwright baseline snapshots committed for all four `/` states.
- `make verify` passes.
