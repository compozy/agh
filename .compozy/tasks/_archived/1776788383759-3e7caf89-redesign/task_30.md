---
status: completed
title: Rewrite Settings shell (save-bar, page-actions, restart-banner)
type: frontend
complexity: medium
dependencies:
  - task_13
  - task_14
---

# Task 30: Rewrite Settings shell (save-bar, page-actions, restart-banner)

<critical name="redesign-execution">
**Agent:** Run implementation with the designer agent defined in `.claude/agents/designer.md` — **execution mode only** (e.g. `mode: execute`, `execution mode`, "ship it"). **Do not** use plan mode (no `questions_v2`, no brainstorm-only passes) for these tasks.

**Mandatory skills:** `agh-design`, `design-taste-frontend`, `minimalist-ui` — activate before writing or changing UI.

**Design system:** `DESIGN.md` (repo root) is the authoritative design-system spec; tokens and rules there override informal styling.
</critical>

## Overview

Rewrite every reusable Settings-wide surface that each sub-route inherits: the page shell (eyebrow + title + status line + actions + body + footer layout), the dirty-state save bar, the header-level page actions, the daemon-restart banner, and the shared field / section primitives used across all 10 sub-routes. This task establishes the Phase 6 foundation — task 31 and task 32 consume these shells verbatim.

<critical>
- ALWAYS READ the PRD and TechSpec before starting
- REFERENCE TECHSPEC for implementation details — do not duplicate here
- FOCUS ON "WHAT" — describe what needs to be accomplished, not how
- MINIMIZE CODE — show code only to illustrate current structure or problem areas
- TESTS REQUIRED — every task MUST include tests in deliverables
</critical>

<requirements>
- MUST rewrite `settings-page-shell.tsx`, `settings-save-bar.tsx`, `settings-page-actions.tsx`, `settings-restart-banner.tsx`, the shared primitives (`settings-section-card.tsx`, `settings-field-row.tsx`, `settings-status-line.tsx`, `settings-stat-grid.tsx`, `settings-collection-header.tsx`, `settings-source-badge.tsx`), and rewrite the Settings index route (`web/src/routes/_app/settings/index.tsx`) on top of `@agh/ui` `PageHeader`, `Section`, `Button`, `Field`, `Alert`, `Kbd`, `MonoBadge`, and `Sheet` / modal patterns for confirmations.
- MUST rewrite the `_app/settings.tsx` layout route (or add one if missing) so every sub-route nests inside the new shell, including sticky save-bar slot at the bottom and restart-banner slot below the header.
- MUST preserve all existing behaviors: dirty-state detection, Discard / Save buttons, restart trigger polling, operation-id display, success / failure / warning tones, dismiss behavior, last-applied label.
- MUST NOT introduce any imports from `@/components/ui/*` or `@/components/design-system/*` — those folders are deleted in Phase 2.
- MUST use Lucide icons at DESIGN.md sizes (`size-3.5` / `size-4`) and `@agh/ui` tokens only; no hardcoded hex colors.
- MUST expose a stable `data-testid="settings-page-<slug>-*"` contract unchanged from the current shell so sub-route tests keep passing.
- MUST wire the restart-banner with `@agh/ui` `Alert` variants (warning / success / danger) instead of bespoke tone CSS and preserve `role="alert"` / `role="status"` semantics.
- SHOULD render the primary page header using `PageHeader` from `@agh/ui` with an eyebrow `SETTINGS / <Page>` and the action slot populated by `SettingsPageActions`.
- SHOULD delete any now-dead helper CSS (custom tone classes, token-string templates) that the rewrite supersedes.
</requirements>

## Subtasks

- [x] 30.1 Audit the current Settings shell pieces and the `_app/settings/*` routes to list every prop, slot, and `data-testid` consumed by sub-routes.
- [x] 30.2 Rewrite `settings-page-shell.tsx` on top of `@agh/ui` `PageHeader` + `Section`, with slots for actions, banner, body, and footer save-bar.
- [x] 30.3 Rewrite `settings-save-bar.tsx` as a `@agh/ui` `Section`-wrapped footer bar with `Button` primary + ghost, respecting dirty / invalid / saving / error / warning states.
- [x] 30.4 Rewrite `settings-restart-banner.tsx` as a `@agh/ui` `Alert` composition covering idle-warning, polling, success, and failure tones + dismiss.
- [x] 30.5 Rewrite `settings-page-actions.tsx`, `settings-section-card.tsx`, `settings-field-row.tsx`, `settings-status-line.tsx`, `settings-stat-grid.tsx`, `settings-collection-header.tsx`, `settings-source-badge.tsx`, and `web/src/routes/_app/settings/index.tsx` as thin `@agh/ui` compositions.
- [x] 30.6 Add / update the `_app/settings.tsx` layout route to render the shell once and pass the `<Outlet />` into the body; ensure restart-banner + save-bar slots are provided by the consumer page.
- [x] 30.7 Generate Playwright snapshot baselines for the shell states (idle / dirty / saving / restart-warning / restart-polling / restart-success / restart-failure) under `/settings/general` used as the shell representative.

## Implementation Details

See TechSpec "Impact Analysis" row for `web/src/routes/_app/settings/**` and ADR-004 Phase 6 description. DESIGN.md §4 defines the PageHeader, Section, button sizes, and the Alert tint formula the banner must consume. The mock at `docs/design/web-inspiration/src/pages-session.jsx` contains `SettingsPage` with 7 sections — use it as the structural reference for the shell composition (eyebrow + title + body + sticky footer) even though its local `settings` content differs from the real routes.

### Relevant Files

- `web/src/systems/settings/components/settings-page-shell.tsx` — rewrite target.
- `web/src/systems/settings/components/settings-save-bar.tsx` — rewrite target.
- `web/src/systems/settings/components/settings-page-actions.tsx` — rewrite target.
- `web/src/systems/settings/components/settings-restart-banner.tsx` — rewrite target (export of `RestartBannerState` must survive).
- `web/src/systems/settings/components/settings-section-card.tsx`, `settings-field-row.tsx`, `settings-status-line.tsx`, `settings-stat-grid.tsx`, `settings-collection-header.tsx`, `settings-source-badge.tsx` — shared primitives consumed across every sub-route.
- `web/src/systems/settings/components/index.ts` — barrel file.
- `web/src/routes/_app/settings.tsx` (layout) — host for the shell; create if missing.
- `web/src/routes/_app/settings/index.tsx` — Settings index landing page; rewrite as a `PageHeader` + `Section` summary using the shared shell.
- `docs/design/web-inspiration/src/pages-session.jsx` — `SettingsPage` shape reference.

### Dependent Files

- Every `web/src/routes/_app/settings/<page>.tsx` — renders inside this shell (tasks 31 + 32).
- `web/src/systems/settings/components/settings-editor-dialog.tsx` and `settings-delete-dialog.tsx` — reuse `@agh/ui` `Dialog` / `Sheet` primitives from the shell refresh.
- `web/e2e/**` — existing Playwright specs assert `data-testid="settings-page-*"`.

### Related ADRs

- [ADR-001: Consolidate UI primitives into @agh/ui](adrs/adr-001.md) — primitives consumed by the shell.
- [ADR-002: Greenfield migration — delete without backwards-compat](adrs/adr-002.md) — delete bespoke shell CSS alongside the rewrite.
- [ADR-004: Phased rollout](adrs/adr-004.md) — Phase 6 foundation for sub-routes.
- [ADR-005: Visual parity via Playwright snapshots](adrs/adr-005.md) — shell snapshots gate this task.

## Deliverables

- Rewritten `settings-page-shell.tsx`, `settings-save-bar.tsx`, `settings-page-actions.tsx`, `settings-restart-banner.tsx`, shared primitives (including `settings-collection-header.tsx` and `settings-source-badge.tsx`), and the `settings/index.tsx` route composed exclusively from `@agh/ui`.
- Layout route `_app/settings.tsx` hosting the shell with outlet + banner + save-bar slots.
- Zero remaining imports from `@/components/ui/*` or `@/components/design-system/*` in `web/src/systems/settings/components/**` and `web/src/routes/_app/settings/**`.
- Playwright snapshot baselines committed for `/settings/general` in idle / dirty / saving / restart-warning / restart-polling / restart-success / restart-failure states **(REQUIRED)**.
- Unit tests with 80%+ coverage **(REQUIRED)**.
- Storybook interaction tests for save-bar state transitions and restart-banner tones **(REQUIRED)**.

## Tests

- Unit tests:
  - [x] `SettingsPageShell` renders `SETTINGS / <title>` eyebrow + H1 + body + footer slots with the `data-testid="settings-page-<slug>-*"` contract intact.
  - [x] `SettingsSaveBar` disables the Save button when `isDirty=false`, and enables it when `isDirty=true && isSaving=false && isInvalid=false`.
  - [x] `SettingsSaveBar` shows the `Loader2` spinner + "Saving…" label when `isSaving=true` and re-enables after `isSaving` flips to `false`.
  - [x] `SettingsSaveBar` renders `error` text with the danger tone when `error` is non-null, and `warnings` list with warning tone when `error` is null and `warnings.length > 0`.
  - [x] `SettingsSaveBar` renders the `lastAppliedLabel` with the success check when `isDirty=false && !error && !warnings`.
  - [x] `SettingsPageActions` renders "Restart daemon" using `@agh/ui` `Button` variant `outline`, disables it while `restart.isTriggerPending || restart.isPolling`, and calls `restart.trigger()` on click.
  - [x] `SettingsRestartBanner` returns null when `restart.isVisible=false`.
  - [x] `SettingsRestartBanner` renders warning tone + "Changes saved. Restart the daemon to apply." when `isRestartRequired=true && !isPolling && !isSuccessful`.
  - [x] `SettingsRestartBanner` renders polling tone + `status` suffix (`Restarting daemon · <status>`) when `isPolling=true`.
  - [x] `SettingsRestartBanner` renders danger tone + `failureReason` suffix when `isFailed=true`, and exposes `role="alert"`.
  - [x] `SettingsRestartBanner` renders the Dismiss button when `isSuccessful || isFailed` and calls `restart.dismiss()` on click.
  - [x] `SettingsFieldRow` forwards `label`, `description`, `error`, and renders children inside the `@agh/ui` `Field` container.
- Integration tests:
  - [x] Storybook `play()` on `SettingsSaveBar` flips `isDirty` true→false and asserts Discard + Save disable and the "No unsaved changes" placeholder returns.
  - [x] Storybook `play()` on `SettingsRestartBanner` cycles warning → polling → success tones and asserts the Dismiss button only appears in success / failure states.
  - [x] Playwright snapshot baseline for `/settings/general` in idle / dirty / saving / restart-warning / restart-polling / restart-success / restart-failure states matches the committed PNGs within 0.1% threshold.
- Test coverage target: >=80%
- All tests must pass

## Success Criteria

- All tests passing and `make verify` green.
- Test coverage >=80% across the rewritten shell files.
- `grep -r "from \"@/components/ui/" web/src/systems/settings web/src/routes/_app/settings` returns zero hits.
- `grep -r "from \"@/components/design-system" web/src/systems/settings web/src/routes/_app/settings` returns zero hits.
- Playwright baseline snapshots committed for the seven shell states under `/settings/general`.
- `data-testid="settings-page-<slug>-*"` selectors remain stable so every sub-route test in tasks 31 + 32 continues to pass unchanged.
