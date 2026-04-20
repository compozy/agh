---
status: completed
title: Wire Playwright visual snapshot baseline for web/
type: infra
complexity: medium
dependencies:
  - task_11
  - task_13
  - task_14
---

# Task 16: Wire Playwright visual snapshot baseline for web/

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Extend the Playwright visual harness from task 11 to cover the `web/` app: baseline every existing top-level route in the rewritten shell, every `web/src/**/stories/*.stories.tsx` file that renders a full screen, and the `/design-system` showcase. Every subsequent domain task (Phase 3–6) adds snapshots for its own routes.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST extend `web/e2e/playwright.config.ts` or add a dedicated visual project for `web/` that runs against a dev server (`pnpm --filter web dev`) + the `web/` Storybook.
- MUST reuse the helper utilities from task 11 (story URL enumeration, snapshot threshold).
- MUST generate baseline PNGs for every top-level route outer frame after the task 14 shell rewrite (page interiors are still "pre-migration" for domains not yet in Phase 3+).
- MUST generate baseline PNGs for every existing `*.stories.tsx` under `web/src/routes/_app/**/stories/` and `web/src/systems/**/components/stories/`.
- MUST add a `pnpm test:visual:web` script.
- MUST wire a CI job that runs after task 14 merges and passes before any Phase 3 task merges.
- SHOULD document the "snapshots drift during Phase 3–6" expectation in `packages/ui/README.md` (task 12) and the workflow for updating baselines per phase.
</requirements>

## Subtasks

- [x] 16.1 Extend Playwright config for web visual tests.
- [x] 16.2 Write visual spec covering the new shell (sidebar, header) across top-level routes.
- [x] 16.3 Enumerate and snapshot each `*.stories.tsx` file in `web/`.
- [x] 16.4 Generate and commit baselines; wire CI job.
- [x] 16.5 Add `pnpm test:visual:web` script.

## Implementation Details

See ADR-005 for the visual parity rationale. TechSpec "Monitoring and Observability" covers snapshot count tracking.

### Relevant Files

- `web/e2e/playwright.config.ts` — existing config to extend.
- `web/e2e/visual.spec.ts` (or similar) — new spec.
- `web/package.json` — script.
- `web/src/routes/**` — route sources for URL enumeration.
- `web/src/storybook/route-story.tsx` — the existing route-story wrapper used for stories.
- `packages/ui/tests/visual.spec.ts` — helpers to reuse (if shared) or inspired patterns.
- **Design references** (read-only, do not edit):
  - `docs/design/web-inspiration/src/app.jsx` — authoritative list of routes to snapshot.
  - `docs/design/web-inspiration/styles/app.css` — layout skeleton that the baseline captures in the shell frame.

### Dependent Files

- CI workflow file.
- Every subsequent domain task adds snapshots via this infrastructure.

### Related ADRs

- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Playwright web visual project + spec.
- Baseline PNGs for shell + each top-level route + every `*.stories.tsx` in web.
- CI job running visual tests on web branch PRs.
- Unit tests with 80%+ coverage for any helper logic added **(REQUIRED)**.
- Integration tests that the baseline passes clean against the current `main` state **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Story URL enumeration matches all `*.stories.tsx` file IDs under `web/src/**`.
  - [ ] Route URL enumeration matches all public routes in `web/src/routes/**`.
- Integration tests:
  - [ ] `pnpm test:visual:web` produces zero diffs after baseline commit.
  - [ ] A 2px padding change to any primitive causes at least one web baseline diff.
  - [ ] Forced reduced motion yields deterministic snapshots for routes with animations.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `pnpm test:visual:web` passes clean against the Phase 2 state.
- Every top-level route + every `*.stories.tsx` in `web/` has at least one baseline PNG.
- CI job configured and green on `main`.
- `make verify` passes.
