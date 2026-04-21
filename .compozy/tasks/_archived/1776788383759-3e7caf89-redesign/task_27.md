---
status: completed
title: Rewrite Skills domain (installed + marketplace)
type: frontend
complexity: medium
dependencies:
  - task_13
  - task_14
---

# Task 27: Rewrite Skills domain (installed + marketplace)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/skill/**` as a two-tab surface. The **Installed** tab is a split-pane with a searchable, source-grouped list and a detail panel exposing version, author, enable/disable toggle, capabilities, and recent calls. The **Marketplace** tab is a card grid with install / details actions. Domain adapters, hooks, and query wiring stay unchanged — only the visual layer is rewritten on top of `@agh/ui` primitives per ADR-001.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `skill-list-panel.tsx`, `skill-detail-panel.tsx`, and `marketplace-view.tsx` as compositions over `@agh/ui` `SplitPane`, `Tabs`, `PageHeader`, `SearchInput`, `Pills`, `Switch`, `MonoBadge`, `Button`, `Section`, `Card`, and `Table`.
- MUST replace the local installed/marketplace `PillButton` toggle with `@agh/ui` `Tabs`, preserving the `useSkillsPage` `activeTab` state.
- MUST drive enable/disable from `@agh/ui` `Switch`, wired to the existing `handleEnable` / `handleDisable` mutations with `isActionPending` for the disabled state.
- MUST render capabilities and recent-calls sections via `Section` + `Table`; render version/author/source via `PageHeader` meta + `MonoBadge`.
- MUST replace the marketplace rows with a responsive `Card` grid using `Pills` for the category filter (`ALL`, `TESTING`, `DATABASE`, `DEPLOY`, `AI`, `DEVOPS`, `SECURITY`) and a per-card install `Button` (or "Installed" `MonoBadge` when already installed).
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*`.
- MUST keep `web/src/routes/_app/skills.tsx` as a thin route wiring `useSkillsPage` into the new composition; route path `/skills` is unchanged.
- SHOULD extract badge tone + category mapping into `lib/` so components stay presentational.
</requirements>

## Subtasks

- [x] 27.1 Audit `web/src/systems/skill/components/**`, `routes/_app/skills.tsx`, and `useSkillsPage` to inventory props and behaviors that must survive.
- [x] 27.2 Rewrite `skill-list-panel.tsx` on `SplitPane.list` with `SearchInput`, source-grouped rows, and `MonoBadge` source chips.
- [x] 27.3 Rewrite `skill-detail-panel.tsx` on `SplitPane.detail` with `PageHeader`, `Switch` (enable/disable), `Section` blocks for Overview / Capabilities / Recent Calls, and `Table` for call history.
- [x] 27.4 Rewrite `marketplace-view.tsx` as a `Card` grid with `Pills` category filter, install `Button`, and `Empty` state when filters match nothing.
- [x] 27.5 Rewrite `routes/_app/skills.tsx` to compose `PageHeader` + `Tabs` (Installed / Marketplace) + the new panels.
- [x] 27.6 Update or add Storybook stories for installed-list, detail, marketplace-grid, loading, error, empty (no installed, no marketplace results), and disabled-install.
- [x] 27.7 Regenerate Playwright visual snapshot baselines for `/skills` (installed-empty, installed-populated, detail-open, marketplace-grid).

## Implementation Details

Follow TechSpec "Impact Analysis" row `web/src/systems/skills/**` — visual-only rewrite, data flow untouched. The mock `SkillsPage` is the visual target. The split-pane + tabs composition mirrors the Network / Automation / Bridges / Knowledge tasks.

### Relevant Files

- `web/src/systems/skill/components/skill-list-panel.tsx` — rewrite target.
- `web/src/systems/skill/components/skill-detail-panel.tsx` — rewrite target.
- `web/src/systems/skill/components/marketplace-view.tsx` — rewrite target.
- `web/src/routes/_app/skills.tsx` — thin route rewrite.
- `web/src/systems/skill/components/stories/*.stories.tsx` — update for new primitives.
- `web/src/hooks/routes/use-skills-page.ts` — unchanged consumer of new components.
- `packages/ui/src/components/{split-pane,tabs,page-header,search-input,pills,switch,mono-badge,button,section,card,table,empty}.tsx` — primitives consumed.

### Dependent Files

- `web/e2e/__snapshots__/` — new baselines for `/skills` route states.
- `web/src/routes/_app/-skills.test.tsx` — route test updated to assert new testids.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `skill-list-panel.tsx`, `skill-detail-panel.tsx`, and `marketplace-view.tsx`.
- Rewritten `routes/_app/skills.tsx` with `Tabs` wiring.
- Storybook stories covering list / detail / marketplace / loading / error / empty / disabled-install.
- Playwright snapshot baselines for `/skills` installed-empty, installed-populated, detail-open, and marketplace-grid **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for tab switch, enable/disable `Switch`, and marketplace category filter **(REQUIRED)**.

## Tests

- Unit tests:
  - [ ] `Tabs` switch between `Installed` and `Marketplace` updates `useSkillsPage.activeTab` and swaps rendered content.
  - [ ] `SkillListPanel` groups rows by source in the order BUNDLED → WORKSPACE → MARKETPLACE → USER → ADDITIONAL.
  - [ ] `SearchInput` filters rows by name, description, and tags (case-insensitive).
  - [ ] `SkillDetailPanel` renders version + author in `PageHeader` meta and source as `MonoBadge`.
  - [ ] `Switch` in detail panel fires `handleEnable` / `handleDisable` and is disabled when `isActionPending` is true.
  - [ ] `MarketplaceView` renders a `Card` grid; clicking install calls `onInstall(name)`; already-installed cards render an "Installed" `MonoBadge` instead of an install button.
  - [ ] Category `Pills` filter marketplace cards and show `Empty` when nothing matches.
- Integration tests:
  - [ ] Storybook `play()` switches tabs and asserts marketplace grid renders.
  - [ ] Storybook `play()` toggles the detail `Switch` and asserts the enable/disable mutation fires with the correct skill name.
  - [ ] Playwright visual snapshot match for `/skills` installed-empty, installed-populated, detail-open, and marketplace-grid states.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `web/src/systems/skill/**` contains zero imports from `@/components/ui/*` or `@/components/design-system/*`.
- `/skills` renders only through `@agh/ui` primitives + domain hooks.
- Playwright baseline snapshots committed for all four `/skills` states.
- `make verify` passes.
