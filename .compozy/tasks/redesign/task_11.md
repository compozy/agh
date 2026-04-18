---
status: pending
title: Wire Playwright visual snapshot harness for @agh/ui
type: infra
complexity: high
dependencies:
  - task_02
  - task_03
  - task_04
  - task_05
  - task_06
  - task_07
  - task_08
  - task_09
  - task_10
---

# Task 11: Wire Playwright visual snapshot harness for @agh/ui

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Set up Playwright as the visual-regression gate for every story under `packages/ui/src/components/stories/`. Generate the baseline PNGs, wire the snapshot job into CI, and lock thresholds. From this task onward, every primitive story is snapshot-protected: API or visual drift fails CI.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST add a Playwright project in `packages/ui/` (or extend the existing `web/e2e/playwright.config.ts` to cover packages/ui) that runs Chromium against the built Storybook.
- MUST use Playwright's `toHaveScreenshot` with a 0.1% pixel-diff threshold.
- MUST force `prefers-reduced-motion: reduce` at the Playwright context level so animated stories are snapshotted in their resting state.
- MUST generate baseline PNGs for every story in `packages/ui/src/components/stories/` and commit them under `packages/ui/src/components/stories/__snapshots__/`.
- MUST add a `pnpm test:visual` script in `packages/ui/package.json` (or monorepo root) that runs the Playwright visual suite.
- MUST add a CI job that runs this script on `ubuntu-22.04` pinned to Playwright's bundled Chromium.
- MUST fail the CI job on any baseline drift; include a `--update` flag path for intentional visual changes.
- SHOULD pin the Storybook dev server port so snapshots are deterministic across runs.
</requirements>

## Subtasks

- [ ] 11.1 Add Playwright config for packages/ui (or extend existing config).
- [ ] 11.2 Write a test that iterates over Storybook story URLs and calls `toHaveScreenshot`.
- [ ] 11.3 Generate initial baselines for all primitive stories.
- [ ] 11.4 Add `pnpm test:visual` script and a GitHub Actions job wiring.
- [ ] 11.5 Document the `--update` workflow in `packages/ui/README.md` (task 12 consumer).

## Implementation Details

See ADR-005 for the choice rationale + threshold justification. TechSpec "Monitoring and Observability" describes the snapshot count tracking expectation.

Storybook 10 exposes a `stories.json` or `index.json` endpoint at `/index.json` that lists all stories — use it to enumerate URLs.

### Relevant Files

- `packages/ui/playwright.config.ts` — new or extended.
- `packages/ui/tests/visual.spec.ts` (or similar) — new.
- `packages/ui/package.json` — add script + dev deps.
- `.github/workflows/*` — add visual snapshot CI job.
- `packages/ui/.storybook/main.ts` — for story URL discovery.

### Dependent Files

- All primitive stories from tasks 02–10.
- Task 16 (web Playwright baseline) follows a similar pattern.

### Related ADRs

- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Playwright visual config + test runner.
- Baseline PNGs for every `@agh/ui` story.
- CI job running visual tests on every PR.
- Unit tests with 80%+ coverage for any helper logic (e.g., story URL discovery utility) **(REQUIRED)**.
- Integration tests for the snapshot workflow itself (run in dry-run on a known story) **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Story URL enumeration helper returns all story IDs from a mock `index.json`.
  - [ ] Snapshot filename convention matches story ID + viewport.
- Integration tests:
  - [ ] Running `pnpm test:visual` against the baseline produces zero diffs.
  - [ ] Modifying one primitive's padding by 2px causes at least one snapshot test to fail.
  - [ ] `prefers-reduced-motion: reduce` forced at context prevents animated stories from being non-deterministic.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `pnpm test:visual` succeeds on `main` after baseline commit.
- CI fails a branch that introduces a visual drift above the 0.1% threshold.
- Every `@agh/ui` story has at least one baseline PNG committed.
