---
status: pending
title: Rewrite Session domain inspector panel
type: frontend
complexity: medium
dependencies:
  - task_20
---

# Task 22: Rewrite Session domain inspector panel

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Introduce the right-hand inspector panel on the Session page — a 320px fixed column exposing four sections: Trace timeline (last 6 events), Usage (token + cost metrics), Memory (loaded documents), and Files (files read during the session). The panel composes `@agh/ui` primitives (`Section`, `Metric`, `MonoBadge`, `StatusDot`, `Tabs`, `ScrollArea`) and consumes existing data hooks unchanged. This is the third visible surface of Phase 4 (ADR-004) and the final piece of the Session rewrite.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST create `web/src/systems/session/components/session-inspector.tsx` (and split children per section if clarity demands) composed from `@agh/ui` `Section`, `Metric`, `MonoBadge`, `StatusDot`, `Tabs`, and `ScrollArea`.
- MUST render four sections in this order: `Trace`, `Usage`, `Memory`, `Files`. When the viewport height cannot fit all four stacked, the panel MUST switch to a `Tabs` layout with the same four labels; the choice is a CSS-driven threshold, not a runtime prop.
- MUST show the last 6 events in `Trace` with per-row mono timestamp + kind `MonoBadge` + status `StatusDot` + one-line label; older events remain reachable through a "View all" ghost link that routes to the full trace view.
- MUST render `Usage` as a two-column `Metric` grid: tokens in / tokens out / total cost / estimated rate, using the semantic-value coloring from DESIGN.md §3 (positive green, negative red, neutral primary).
- MUST render `Memory` as a list of loaded documents with `MonoBadge` for doc kind + title + byte size; render the empty state via `@agh/ui` `Empty` when no docs are loaded.
- MUST render `Files` as a list of read file paths with mono styling and a trailing read-count; the list lives inside a `ScrollArea` so long sessions don't push the panel height.
- MUST read data exclusively through existing hooks — `useSessionTranscript`, `useSessionChat`, `useSessionHistory`, and any hooks the Trace/Usage/Memory/Files slices already expose; this task adds NO new fetch logic.
- MUST NOT import from `@/components/ui/**` or `@/components/design-system/**`.
- SHOULD collapse to a hidden drawer on narrow viewports (<1200px) using the same responsive pattern as the app sidebar.
</requirements>

## Subtasks

- [ ] 22.1 Audit existing session hooks to confirm Trace, Usage, Memory, and Files data are already exposed; list any gaps and open a focused follow-up rather than extending this task.
- [ ] 22.2 Build `session-inspector.tsx` shell with the four-section stacked layout bound to the hooks.
- [ ] 22.3 Build the `Tabs`-based compact layout triggered by the CSS viewport threshold.
- [ ] 22.4 Mount the inspector inside `web/src/routes/_app/session.$id.tsx` at 320px fixed width to the right of the thread.
- [ ] 22.5 Write or add Storybook stories: all four sections populated, each section empty, compact (tabbed) layout, narrow-viewport drawer.
- [ ] 22.6 Run `make web-lint`, `make web-typecheck`, `make web-test`, and smoke the live route.

## Implementation Details

See TechSpec "Impact Analysis" — Phase 4 Session domain. DESIGN.md §4 covers `Metric`, `MonoBadge`, `StatusDot`, and `Section`; §5 covers the panel width + flat-depth model. The mock in `docs/design/web-inspiration/` shows the inspector column layout. No new API contracts — every data read flows through existing hooks.

### Relevant Files

- `web/src/systems/session/components/session-inspector.tsx` — new.
- `web/src/systems/session/hooks/use-session-transcript.ts` — trace + memory data source.
- `web/src/systems/session/hooks/use-session-chat.ts` — usage + file-read accounting.
- `web/src/systems/session/hooks/use-session-history.ts` — historical trace rows.
- `web/src/routes/_app/session.$id.tsx` — mount point.

### Dependent Files

- `web/src/systems/session/components/stories/*` — new inspector stories.
- `web/src/systems/session/index.ts` — barrel export for the new component.
- Playwright snapshot suite for the `/session/$id` route picks up the new column.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout — Phase 4 Session](adrs/adr-004.md)
- [ADR-005: Playwright visual snapshots](adrs/adr-005.md)

## Deliverables

- New `session-inspector.tsx` composed from `@agh/ui` primitives, mounted into the Session route.
- Storybook stories covering populated / empty-per-section / tabbed compact / drawer.
- Playwright visual snapshot baselines per story variant and an updated `/session/$id` route snapshot.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Integration test rendering the inspector against a fixture session with all four slices populated **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] Inspector with a trace of 10 events renders only the most recent 6 rows plus a `View all` ghost link.
  - [ ] Each trace row renders a mono timestamp, a kind `MonoBadge`, and a `StatusDot` whose tone maps `ok→success`, `warn→warning`, `error→danger`, `pending→accent`.
  - [ ] Usage section renders four `Metric` tiles (`Tokens in`, `Tokens out`, `Total cost`, `Estimated rate`) and colors a positive delta in `--color-success`, a negative delta in `--color-danger`.
  - [ ] Memory section with zero docs renders the `Empty` state; with docs, each row shows kind badge + title + byte size.
  - [ ] Files section list is wrapped in `ScrollArea` and renders one row per file with path + read-count.
  - [ ] Narrow viewport (<1200px simulated via CSS container query) collapses the panel into the `Tabs` layout and preserves the same four sections.
- Integration tests:
  - [ ] Storybook interaction: switch tabs in the compact layout and assert each tab's content becomes visible and remains accessible via keyboard arrows.
  - [ ] Rendering `session.$id` with a fixture session populates all four sections from their existing hooks without additional network calls beyond those already asserted for the thread (task 20).
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing.
- Test coverage >=80% for `session-inspector.tsx`.
- `make verify` and `make web-lint` + `make web-typecheck` pass with zero warnings.
- No imports from `@/components/ui/**` or `@/components/design-system/**` inside the inspector.
- Playwright baselines committed for every story variant and the updated `/session/$id` route.
- Inspector stays visually stable (no layout shift) while the thread streams new events in dev mode.
