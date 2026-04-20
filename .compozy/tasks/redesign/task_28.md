---
status: completed
title: Rewrite Workspace onboarding and Agent sidebar integrations
type: frontend
complexity: medium
dependencies:
  - task_13
  - task_14
---

# Task 28: Rewrite Workspace onboarding and Agent sidebar integrations

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite `web/src/systems/workspace/**` (workspace onboarding screen, setup dialog, workspace selector) and `web/src/systems/agent/**` (agent sidebar group, agent icon/glyph) on top of `@agh/ui` primitives. Both surfaces integrate with the already-rewritten `app-sidebar.tsx` from task 13, and workspace onboarding replaces the first-run shell when no workspace is registered. **These screens are NOT in the `docs/design/web-inspiration` mock** — per ADR-004 ("non-mocked screens derivation rule") + TechSpec Phase 5, visuals are derived from the existing design system (Sidebar + PageHeader + Dialog + Field + StatusDot + Pills) without inventing new primitives.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `workspace-setup.tsx` (both `WorkspaceOnboarding` and `WorkspaceSetupDialog`) as a composition over `@agh/ui` `Dialog`, `Field`, `Button`, `Pills`, `Section`, and `Empty`; the "global" vs "manual path" option cards MUST use `Section` + `Field` rather than hand-rolled `<section>` blocks.
- MUST rewrite `workspace-page-shell.tsx` on top of `@agh/ui` `PageHeader` + `Section` so every workspace-scoped route inherits the new chrome instead of the deleted `@/components/design-system/*` primitives.
- MUST rewrite `workspace-selector.tsx` (the sidebar rail workspace switcher) on top of `@agh/ui` `Avatar` + `StatusDot` + `Pills` (for the `HOME` / `PATH` markers), wired into the existing `active-workspace-store.ts`.
- MUST rewrite `agent-sidebar-group.tsx` as a composition over `@agh/ui` sidebar primitives exported in task 05 (group, label, action, menu, sub) plus the custom `AgentIcon` glyph; remove the direct `@/components/ui/collapsible` and `@/components/ui/sidebar` imports.
- MUST rewrite `agent-icon.tsx` as a small presentational glyph rendered with DESIGN.md stroke/size conventions and tone tokens; no `@/components/ui/*` imports.
- MUST preserve every existing behavior: onboarding → resolve workspace, dialog open/close, new-session action, agent tree expand/collapse, "No sessions" empty state, provider-specific glyph.
- MUST NOT import from `@/components/ui/*` or `@/components/design-system/*` (both folders are deleted in Phase 2).
- MUST NOT introduce visuals that are not already documented in `DESIGN.md` or covered by existing `@agh/ui` primitives — per ADR-004 non-mocked rule.
- SHOULD consolidate copy/strings for onboarding into a single `lib/` module so components stay presentational.
</requirements>

## Subtasks

- [x] 28.1 Audit `systems/workspace/components/**` and `systems/agent/components/**` plus the consuming hooks and stores to catalogue surviving props and behaviors.
- [x] 28.2 Rewrite `workspace-setup.tsx` — onboarding full-page layout + dialog variant — on `Dialog` / `Section` / `Field` / `Pills` / `Button`; keep the global-vs-manual split.
- [x] 28.3 Rewrite `workspace-selector.tsx` as a rail composition on `Avatar` + `StatusDot` + `Pills`, wired to `active-workspace-store.ts`; rewrite `workspace-page-shell.tsx` on `PageHeader` + `Section`.
- [x] 28.4 Rewrite `agent-sidebar-group.tsx` and `agent-icon.tsx` on the new `@agh/ui` sidebar primitives + tokenized glyph.
- [x] 28.5 Update or add Storybook stories: onboarding default, onboarding with path error, setup dialog open, workspace selector (empty / single / many / active), agent group (collapsed / expanded / no-sessions / disabled-new-session).
- [x] 28.6 Regenerate Playwright visual snapshot baselines for workspace onboarding + setup-dialog + app-sidebar-with-agent-tree states.
- [x] 28.7 Run `make web-lint`, `make web-typecheck`, `make web-test`, and `make verify`.

## Implementation Details

Follow TechSpec "Impact Analysis" row `web/src/systems/workspace/**, daemon/**, agent/**` — "modified (visual, derived)". ADR-004 Phase 5 states these screens are not in the mock and must reuse Sidebar + PageHeader + SplitPane + Metric + Empty consistently. This task is the visual derivation for workspace + agent; task 29 derives daemon + home dashboard.

### Relevant Files

- `web/src/systems/workspace/components/workspace-setup.tsx` — rewrite target (onboarding + dialog).
- `web/src/systems/workspace/components/workspace-selector.tsx` — rewrite target (sidebar rail).
- `web/src/systems/workspace/components/workspace-page-shell.tsx` — rewrite target (PageHeader + Section scaffold for workspace routes).
- `web/src/systems/agent/components/agent-sidebar-group.tsx` — rewrite target.
- `web/src/systems/agent/components/agent-icon.tsx` — rewrite target.
- `web/src/components/app-sidebar.tsx` — consumer from task 13; imports the rewritten agent + workspace components.
- `web/src/systems/workspace/hooks/use-workspace-setup-content.ts`, `systems/workspace/stores/active-workspace-store.ts` — unchanged consumers.
- `packages/ui/src/components/{dialog,field,button,pills,section,empty,avatar,status-dot,sidebar}.tsx` — primitives consumed.

### Dependent Files

- `web/e2e/__snapshots__/` — new baselines for onboarding, setup dialog, and sidebar integrations.
- Existing tests `workspace-setup.test.tsx`, `workspace-selector.test.tsx`, `agent-sidebar-group.test.tsx`, `agent-icon.test.tsx` — updated for new primitives + testids.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md)
- [ADR-002: Greenfield migration](adrs/adr-002.md)
- [ADR-004: Phased rollout — non-mocked screens derivation rule](adrs/adr-004.md)
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md)

## Deliverables

- Rewritten `workspace-setup.tsx`, `workspace-selector.tsx`, `workspace-page-shell.tsx`, `agent-sidebar-group.tsx`, `agent-icon.tsx` composed from `@agh/ui` primitives only.
- Storybook stories for every variant listed in subtask 28.5.
- Playwright snapshot baselines for workspace onboarding, setup dialog, and app-sidebar-with-agent-tree (collapsed + expanded) **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for submit manual path, use-global-workspace, agent group expand/collapse, new-session action **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] `WorkspaceOnboarding` renders onboarding hero, home card, and manual-path card; calls `onWorkspaceResolved(id)` after a successful submit.
  - [x] `WorkspaceSetupDialog` opens via `open` prop, closes on success, and routes the same submit paths as onboarding.
  - [x] Manual path `Field` shows `manualError` text and an error-tone state when validation fails.
  - [x] "Use global workspace" `Button` is disabled when `globalUnavailableReason` is set, enabled otherwise.
  - [x] `WorkspaceSelector` renders one `Avatar` per workspace, highlights the active one, and calls `setActiveWorkspace(id)` on click.
  - [x] `AgentSidebarGroup` renders the agent name + provider glyph, expands/collapses on trigger click, and shows the `No sessions` empty label when no children are passed.
  - [x] `AgentSidebarGroup` "+" action calls `onNewSession(agent.name)` and is disabled when `newSessionDisabled`.
  - [x] `AgentIcon` renders distinct glyphs for each supported provider.
- Integration tests:
  - [x] Storybook `play()` submits a manual workspace path and asserts `onWorkspaceResolved` fires.
  - [x] Storybook `play()` expands the agent group and asserts rendered sessions from the children slot.
  - [x] Playwright visual snapshot match for workspace onboarding, setup dialog, and sidebar with agent tree expanded.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing
- Test coverage >=80%
- `web/src/systems/workspace/**` and `web/src/systems/agent/**` contain zero imports from `@/components/ui/*` or `@/components/design-system/*`.
- Onboarding + setup dialog + sidebar integrations render only through `@agh/ui` primitives + domain hooks/stores.
- Playwright baseline snapshots committed for the workspace + sidebar-integration states.
- `make verify` passes.
