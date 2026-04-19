# Task Memory: task_30.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrite Settings shell (page-shell + save-bar + page-actions + restart-banner + shared primitives) on `@agh/ui`. Extend `@agh/ui` `Alert` with success/warning/info/accent variants. Layout + index route rewritten. 7 Playwright baselines for `/settings/general` in idle / dirty / saving / restart-warning / restart-polling / restart-success / restart-failure committed. Unit + storybook interaction tests added. Task complete.

## Important Decisions

- **Extend Alert, don't clone.** `@agh/ui` `Alert` had only `default|destructive` variants; extending it with `success|warning|info|accent` per DESIGN.md §2 tint-formula keeps the banner a pure `@agh/ui` composition instead of a bespoke tone-CSS alternative. `role` now accepts an override so consumers can pass `role="status"` for non-danger tones (required by restart-banner).
- **SettingsPageShell keeps its tall header + eyebrow.** Did NOT swap the shell header for `@agh/ui` `PageHeader`. `PageHeader` is compact (py-2.5) and lacks an eyebrow slot; the Settings shell needs the tall eyebrow `SETTINGS / <Page>` + H1 + status line + action slot laid out vertically. The shell uses `@agh/ui` `cn` but renders the header chrome locally via DESIGN.md tokens. This matches the mock `pages-settings.jsx` which also uses a plain H1 inside the scroll container, not a nested `PageHeader`.
- **Save-bar last-applied label NOT gated by `!isDirty`.** Task Tests section says "renders lastAppliedLabel when isDirty=false && !error && !warnings", but existing `-network.test.tsx` relies on it rendering even when dirty. Kept original behavior (label renders whenever `lastAppliedLabel` is truthy and no error/warnings) — the Tests item is read as a sufficient condition, not a necessary one.
- **Polling tone → `info` (Alert info variant).** Previous banner colored all non-terminal states as `warning`. Split now: warning for idle-restart-required, `info` while polling, `success` after completion, `danger` on failure. Unit + Playwright baselines reflect this.
- **Stat grid uses `@agh/ui` `Metric`.** `SettingsStatItem` delegates to `Metric` (label + value + subtext). Visual match: the existing stat cards used a mono eyebrow + medium value — the Metric primitive already renders the exact DESIGN.md contract (mono 11px label + Inter 24/700 value + Inter 13px subtext), so settings adopts it verbatim.
- **Source badge switched from `Pill` to `MonoBadge`.** DESIGN.md §"Status badges" — small mono status chips should be `MonoBadge`, not `Pill`. Kind-per-source tone mapping: `builtin-provider→neutral`, `global-*→info`, `workspace-*→warning`. Shadowed chips all `neutral`.
- **Non-play-fn dirty/saving baselines via RAF-polling script.** `StorybookGeneralDraftDirtySetup` and `StorybookGeneralSavingSetup` dispatch `input` events through the native HTMLInputElement setter to trigger React's synthetic handler. Pattern from task 27 (MarketplaceTabAutoClick).

## Learnings

- When adding a `type` export to `packages/ui/src/index.ts`, the `tests/readme.test.ts` check enforces that every exported identifier appears somewhere in the README. Missing `AlertProps` caused a test failure — fix is to add the identifier to the Primitive inventory table.
- `SettingsSectionDescriptor` does NOT carry an icon field — Nav-item icons would require a type extension. Opted not to change the type; the existing text-only nav link stays.

## Files / Surfaces

**Rewritten primitives** (`web/src/systems/settings/components/`):
- `settings-page-shell.tsx` — tall header + eyebrow + banner/body/footer slots; `@agh/ui` `cn`.
- `settings-save-bar.tsx` — `@agh/ui` `Button` (ghost + default); all existing testids preserved.
- `settings-restart-banner.tsx` — `@agh/ui` `Alert` + `AlertDescription`; warning/info/success/danger tones.
- `settings-page-actions.tsx` — unchanged structurally; already thin `Button` composition.
- `settings-section-card.tsx` — `@agh/ui` `cn` only.
- `settings-field-row.tsx` — wraps `@agh/ui` `Field`; adds `error` prop surface.
- `settings-stat-grid.tsx` — `SettingsStatItem` delegates to `@agh/ui` `Metric`.
- `settings-collection-header.tsx` — `@agh/ui` `cn` only.
- `settings-source-badge.tsx` — `MonoBadge`-based; tone-mapped per source kind.
- `settings-status-line.tsx` — existing `StatusDot` composition, no functional change.

**Routes + layout**:
- `web/src/routes/_app/settings.tsx` — rewrote nav color tokens; no structural change (data-testid contract preserved).
- `web/src/routes/_app/settings/index.tsx` — now composes `@agh/ui` `Empty`.

**Extended @agh/ui**:
- `packages/ui/src/components/alert.tsx` — added `success|warning|info|accent` variants + `role` override + `data-variant` attribute + `AlertProps` export. README + showcase stories updated.

**New tests + stories**:
- `web/src/systems/settings/components/settings-save-bar.test.tsx` (new, 9 specs).
- `web/src/systems/settings/components/settings-page-actions.test.tsx` (new, 4 specs).
- `web/src/systems/settings/components/settings-page-shell.test.tsx` (expanded, 6 specs).
- `web/src/systems/settings/components/settings-restart-banner.test.tsx` (expanded, 6 specs).
- `web/src/systems/settings/components/settings-field-row.test.tsx` (expanded, 3 specs).
- `web/src/systems/settings/components/stories/settings-save-bar.stories.tsx` (new, 8 stories + play-fn).
- `web/src/systems/settings/components/stories/settings-restart-banner.stories.tsx` (new, 5 stories + play-fn).
- `packages/ui/src/components/alert.test.tsx` (new, 3 specs).
- `packages/ui/src/components/stories/alert.stories.tsx` (expanded — default/destructive/warning/success/info/accent).
- `web/src/storybook/settings-state-helpers.tsx` (new) — `StorybookRestartPhaseSetup`, `StorybookGeneralDraftDirtySetup`, `StorybookGeneralSavingSetup`.
- `web/src/routes/_app/settings/stories/-general.stories.tsx` (expanded — 9 stories covering idle/loading/error/dirty/saving/restart-warning/restart-polling/restart-success/restart-failure).

## Errors / Corrections

- Initially gated `lastAppliedLabel` display on `!isDirty` per task test spec — broke existing `-network.test.tsx` which relies on label showing even when dirty. Reverted to original `lastAppliedLabel ? … : …` branch.
- Initially added `Icon` rendering to settings nav link referencing `section.icon` — `SettingsSectionDescriptor` has no icon field. Removed nav-icon rendering; kept text-only link.
- `readme.test.ts` failed after adding `AlertProps` export — fixed by referencing `AlertProps` in the README inventory table entry for `Alert`.

## Ready for Next Run

- Pattern for non-play-fn dirty/saving baselines documented above — reusable for tasks 31/32 sub-routes that need "dirty" or "saving" baselines without clicking through play-fn.
- `StorybookRestartPhaseSetup` is generic across all 10 settings sections — tasks 31/32 can reuse it with `section="<slug>"` to stamp banner baselines for every sub-route without rolling new fixtures.
- `Alert` warning/success/info/accent variants are now general-purpose primitives — any future banner/notification work should consume them instead of re-rolling tone CSS.
