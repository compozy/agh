---
status: completed
title: Add Metric, MonoBadge, KindChip, StatusDot, ConnectionIndicator to @agh/ui
type: refactor
complexity: high
dependencies:
  - task_01
---

# Task 07: Add Metric, MonoBadge, KindChip, StatusDot, ConnectionIndicator to @agh/ui

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Introduce the compact signal primitives: `Metric` (large mono value + label), `MonoBadge` (identifier pill), `KindChip` (protocol kind marker), `StatusDot` (tinted dot + pulse), `ConnectionIndicator` (dot + label composite). Migrate legacy `design-system/{metric,metric-strip,status-dot,connection-indicator}` call sites to these new primitives in the same PR.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `Metric`, `MonoBadge`, `KindChip`, `StatusDot`, `ConnectionIndicator` primitives in `packages/ui/src/components/` and export them from `packages/ui/src/index.ts`.
- MUST follow DESIGN.md §4 for `StatusDot` tones (success/warning/danger/info/accent/neutral) + optional `pulse`.
- MUST follow DESIGN.md §4 for `Metric` layout (11px mono uppercase label, 24px Inter bold value, optional subtext, semantic coloring).
- MUST follow the `KindChip` spec for protocol kinds (5px radius, lowercase mono, accent-tint background, accent text).
- MUST rewrite all importers of `@/components/design-system/{metric,metric-strip,status-dot,connection-indicator}` to use the new primitives in the same PR.
- MUST DELETE `metric.tsx`, `metric-strip.tsx`, `status-dot.tsx`, `connection-indicator.tsx` from `web/src/components/design-system/` after migration.
- MUST add stories for each primitive covering all tone variants and pulse/non-pulse states.
</requirements>

## Subtasks

- [x] 7.1 Implement the five primitives with stories covering all tones + variants.
- [x] 7.2 Replace `design-system/metric` and `design-system/metric-strip` importers with compositions of `@agh/ui` `Metric`.
- [x] 7.3 Replace `design-system/status-dot` importers with `@agh/ui` `StatusDot`.
- [x] 7.4 Replace `design-system/connection-indicator` importers with `@agh/ui` `ConnectionIndicator`.
- [x] 7.5 Delete the four migrated files from `web/src/components/design-system/`.
- [x] 7.6 Run `make verify` and fix regressions (frontend portion clean; Go lint failures are pre-existing issues tracked in shared workflow memory "Open Risks" and out of scope for task_07).

## Implementation Details

DESIGN.md §4 lists the exact visual specs. The mock `docs/design/web-inspiration/src/primitives.jsx` shows `Metric` composition; `sidebar.jsx` shows `StatusDot` usage patterns. `MonoBadge` and `KindChip` are new — no legacy equivalents exist in `web/` (they are introduced by the redesign).

`ConnectionIndicator` composes `StatusDot` + a label; it is a higher-order primitive but still generic enough to live in `@agh/ui` (DESIGN.md §4 documents it).

### Relevant Files

- `web/src/components/design-system/metric.tsx`, `metric-strip.tsx`, `status-dot.tsx`, `connection-indicator.tsx` — migration sources.
- `packages/ui/src/components/` — destination.
- `packages/ui/src/index.ts` — add exports.
- `packages/ui/src/components/stories/` — destination for stories.
- `docs/design/web-inspiration/src/primitives.jsx` + `sidebar.jsx` — reference composition.
- DESIGN.md §4 — Metric / StatusDot / MonoBadge / KindChip visual specs.

### Dependent Files

- ~5 importers of `design-system/status-dot` (per inventory).
- ~1 importer each of metric-strip, connection-indicator, metric (some in dashboard + bridges domain).
- Task 15 deletes the remaining `design-system/design-system-showcase.tsx` after this task empties the folder.
- Many future domain tasks consume `MonoBadge`, `KindChip`, `StatusDot` fresh.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)

## Deliverables

- Five new primitives with stories.
- Four design-system files migrated + deleted.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for tone variants + pulse **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `Metric` renders label + value + optional subtext in the mock-specified sizes/weights.
  - [ ] `Metric` applies semantic color to the value when `tone="success|danger|warning"`.
  - [ ] `MonoBadge` renders with mono font + 6px radius + provided label uppercase.
  - [ ] `KindChip` renders lowercase label with accent-tint background.
  - [ ] `StatusDot` with `pulse` applies the pulse animation; without `pulse`, no animation runs.
  - [ ] `StatusDot` tone maps to the correct semantic color token.
  - [ ] `ConnectionIndicator` composes `StatusDot` + label with correct tone per connection state.
  - [ ] Under `prefers-reduced-motion: reduce`, `StatusDot` pulse is suppressed.
- Integration tests:
  - [ ] Storybook `play()` cycles through tone variants of `StatusDot` and verifies the correct CSS custom property.
  - [ ] A domain screen previously using `design-system/metric-strip` renders with equivalent visual output (Playwright snapshot parity with baseline).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `rg "@/components/design-system/(metric|metric-strip|status-dot|connection-indicator)" web/src` returns zero matches.
- Five primitives exported from `packages/ui/src/index.ts`.
- Four files deleted from `web/src/components/design-system/`.
- `make verify` passes.
