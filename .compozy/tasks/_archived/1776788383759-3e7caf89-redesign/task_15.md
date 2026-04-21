---
status: completed
title: Rewrite /design-system showcase and delete design-system folder
type: refactor
complexity: medium
dependencies:
  - task_06
  - task_07
  - task_14
---

# Task 15: Rewrite /design-system showcase and delete design-system folder

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/components/design-system/design-system-showcase.tsx` (and its route `/design-system`) so it consumes `@agh/ui` primitives directly instead of defining its own. After the rewrite, delete the `web/src/components/design-system/` folder entirely — this is the closing act of Phase 2 consolidation.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `design-system-showcase.tsx` to import only from `@agh/ui` + Lucide + other domain-neutral libs. No imports from `@/components/design-system/*` or `@/components/ui/*`.
- MUST cover every primitive exported from `@agh/ui` (as of task 10) in the showcase, grouped: Foundations (tokens, typography), Buttons + Pills, Inputs + Search, Status + Metric + MonoBadge + KindChip, Feedback (Alert, Empty), Dialog + Sheet + Popover + Tooltip, Code + Chat, Sidebar + SplitPane.
- MUST link the showcase section headers to the repo-root `DESIGN.md` anchors for each section.
- MUST DELETE `web/src/components/design-system/` folder entirely after the showcase rewrite is complete (the last file inside, aside from showcase itself, was removed by tasks 06 + 07).
- MUST preserve the route `/design-system` in TanStack Router — only the component content changes.
- MUST NOT introduce new primitives in the showcase — it only consumes existing ones.
- SHOULD update the route frontmatter / metadata if it currently references old primitive paths.
</requirements>

## Subtasks

- [x] 15.1 Rewrite `design-system-showcase.tsx` as a pure `@agh/ui` consumer organized by primitive category.
- [x] 15.2 Add a "Tokens" section at the top that renders swatches for the full color + radii + type scale from `packages/ui/src/tokens.css`.
- [x] 15.3 Move the rewritten showcase file to `web/src/components/design-system-showcase.tsx` (outside the deleted folder) or inline it into the `/design-system` route file.
- [x] 15.4 Delete the `web/src/components/design-system/` directory entirely.
- [x] 15.5 Update the `/design-system` route file to import from the new location and verify the route renders.
- [x] 15.6 Run `make verify`.

## Implementation Details

See TechSpec "Impact Analysis" row for `/design-system` route + `design-system/` folder deletion. ADR-001 frames the closure. Most of the primitives showcased are introduced by tasks 02–10; this task is the visible proof that the consolidation succeeded.

### Relevant Files

- `web/src/components/design-system/design-system-showcase.tsx` — source to rewrite + move.
- `web/src/routes/**/design-system.tsx` (or similar) — route file consuming the showcase.
- `packages/ui/src/index.ts` — primitive source for the showcase.
- `packages/ui/src/tokens.css` — token source for the swatches section.
- **Design references** (read-only, do not edit):
  - `DESIGN.md` — section anchors the showcase links to.
  - `docs/design/design-system/README.md` — showcase mirrors the README's primitive grouping.
  - `docs/design/design-system/preview/*.html` — every preview card becomes a subsection in the showcase route.

### Dependent Files

- No other web code depends on `design-system-showcase.tsx` — it's a route-local consumer.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)

## Deliverables

- Rewritten showcase composed from `@agh/ui` primitives.
- `web/src/components/design-system/` directory deleted entirely.
- `/design-system` route still works and renders the showcase.
- Unit tests with 80%+ coverage for the showcase component **(REQUIRED)**.
- Integration test verifying the route renders **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] Showcase renders a section for each primitive group.
  - [x] Token swatch section renders every CSS variable defined in `packages/ui/src/tokens.css`.
  - [x] Section headers render as links to the appropriate DESIGN.md anchor.
  - [x] Showcase component imports only from `@agh/ui` + neutral libraries (asserted via a lint test or a file-content check).
- Integration tests:
  - [x] Navigating to `/design-system` in dev server renders the showcase without errors. (Covered via RTL `render(<DesignSystemShowcase />)` in `design-system-showcase.test.tsx`; the route is a thin `createFileRoute` wrapper around the same component.)
  - [ ] Playwright snapshot baseline committed for the showcase page. **Deferred to task 16** — the web-side Playwright visual harness does not exist yet. Stable selectors published in shared workflow memory under "Handoffs".
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `web/src/components/design-system/` directory does not exist.
- `/design-system` route renders the rewritten showcase.
- Every `@agh/ui` primitive has a visible example in the showcase.
- `make verify` passes.
