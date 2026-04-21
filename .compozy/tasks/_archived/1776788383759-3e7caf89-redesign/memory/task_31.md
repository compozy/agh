# Task Memory: task_31.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Rewrote the 5 Phase 6 batch 1 settings sub-routes (`/settings/general`, `/settings/memory`, `/settings/skills`, `/settings/providers`, `/settings/automation`) on top of the task-30 Settings shell + `@agh/ui` primitives only. Replaced raw `<input>` with `@agh/ui` `Input`, raw `<table>` with `@agh/ui` `Table`, and the providers + skills empty-state divs with `@agh/ui` `Empty`. Switched providers' API-key-state pill from `Pill` to `MonoBadge` and the action-result banner from a hand-rolled tone div to `@agh/ui` `Alert`. Hooks, payload shapes, and `data-testid` contracts unchanged. 9 visual baselines added/refreshed; 98 settings route tests + 1485 web tests + 312 visual baselines all green. Task complete.

## Important Decisions

- **Reused task-30 shell verbatim.** No new shared primitives in `web/src/systems/settings/components/` — all rewrites compose `SettingsPageShell` + `SettingsFieldRow` + `SettingsSectionCard` + `SettingsStatGrid` + `SettingsCollectionHeader` + `SettingsEditorDialog` + `SettingsDeleteDialog` + `SettingsSourceBadge` from task 30.
- **Aspirational test items NOT implemented.** The task spec's unit test list references a fictional `auto-resume` Switch, "45m → 2700s" `Input` parsing, and per-skill PATCH endpoints (`/api/settings/skills/<id>`). The actual `useSettings*Page` hooks + OpenAPI schema do not expose those — they emit a single section-level PATCH with the whole config. Keeping the hooks unchanged was a hard requirement, so I implemented tests for the actual surface (Empty primitive integration, dirty + Save wiring, `data-testid` contracts) instead. Hook-layer PATCH contracts are already covered by `use-settings-*-page.test.tsx`.
- **Skills disabled list switched from custom `<ul>` to `@agh/ui` Table.** Per the task spec "MUST render list / table content through `@agh/ui` `Table` with mono meta columns where appropriate". The "Identifier" column carries the mono skill id; the trailing column hosts the `Switch`. The `data-testid="settings-page-skills-disabled-list"` now lives on the wrapping bordered div, not the `<ul>`.
- **Providers API-key-state badge moved from `Pill` → `MonoBadge`** with tone `success` (SET) / `warning` (MISSING). Per memory shared-decisions: small mono status chips should be `MonoBadge`, not `Pill`. The `DEFAULT` chip beside the provider name stayed a `Pill` since it's a label, not a status.
- **Providers status dot switched from raw `<span>` to `@agh/ui` `StatusDot`** with tone `warning` (binary-missing / unconfigured) / `success` (installed). The previous custom `dot` style mapped warning → warning and the "ok" state to `text-tertiary` (neutral grey). Switched the ok state to `success` (DESIGN.md tone vocab) — visually green now, consistent with other domain status helpers.
- **Providers action-result banner now consumes `@agh/ui` `Alert` + `AlertAction`.** Replaces the bespoke tone-mapped `<div>`; uses `Alert variant="success"` for the saved kind and `Alert variant="info"` for the deleted kind. The dismiss button lives in the `AlertAction` slot (absolutely positioned top-right by the primitive).
- **Reusable `StorybookFieldDirtySetup({ testId, value })` helper.** Generic version of `StorybookGeneralDraftDirtySetup` — typed at any input by `data-testid`. Used by the new memory/skills/providers/automation `Dirty` stories. Providers needed a bespoke 2-stage setup (`StorybookProvidersDirtySetup`) because the dirty state lives inside an editor dialog (open it first, then dirty the input).
- **`Input` ≈ legacy custom input visually.** Despite Input using `rounded-lg` + `bg-transparent` + `text-base md:text-sm` and the legacy code using `rounded-md` + `bg-surface-elevated` + `text-sm`, the visual diff stays under the 0.1% threshold for `general` baselines (no general baseline regenerated). Skills + providers baselines regenerate primarily because of Table primitive switch + Empty primitive switch.

## Learnings

- `@agh/ui` `Input` exposes `type`, `min`, `placeholder`, `value`, `onChange`, `disabled`, `data-testid`, `className` via `{...props}` spread on the underlying Base UI Input. RTL `toHaveValue("string")` and `toHaveValue(8)` (number) both work because the rendered DOM is still `<input>`.
- `Table` from `@agh/ui` returns `<div data-slot="table-container"><table data-slot="table" />`. Wrapping it in a bordered `<div data-testid="settings-page-providers-list">` keeps the existing testid contract while letting the primitive own the rounding + table chrome.
- `Alert` primitive grid layout (`*:[svg]:row-span-2`) expects an SVG icon at index 0; otherwise the description hugs the left edge. `AlertAction` slot is absolutely positioned top-right, so a row-style "icon + message + dismiss" pattern naturally falls into place when you pass icon → AlertDescription → AlertAction.
- The visual baseline harness re-screenshots every story; new stories appear under `web/tests/visual/__snapshots__/<story-id>-chromium-darwin.png`. After bulk story addition, run `bun run test:visual:update` once, then re-run `bun run test:visual` to confirm clean.

## Files / Surfaces

**Rewritten routes** (`web/src/routes/_app/settings/`):
- `general.tsx` — Input swaps for default-agent / default-provider / default-environment / session-timeout. Pills + Switch unchanged. Hint label "DEFAULT/CONFIG.TOML/OPTIONAL/SECONDS" preserved.
- `memory.tsx` — Input swap for global-dir + dream-agent + dream-min-hours + dream-min-sessions + dream-check-interval. Switch + button unchanged. Added `disabled` cascade on dream fields when dream is off.
- `skills.tsx` — Disabled list rewritten as `@agh/ui` `Table` (Skill / Identifier / Disabled columns) + `Empty` primitive for zero-state. Marketplace policy fields use `@agh/ui` `Input`. Switch unchanged.
- `providers.tsx` — Catalog rewritten as `@agh/ui` `Table` (Provider / Command / Default model / API key env / Source / Actions). Empty state uses `@agh/ui` `Empty` (Database icon). Status indicator switched to `StatusDot`. API-key-state chip switched to `MonoBadge`. Action result banner switched to `Alert` + `AlertAction`. Editor dialog uses `Input` for every field.
- `automation.tsx` — Input swaps for timezone + max-concurrent-jobs + fire-limit-max + fire-limit-window. Switch unchanged.

**Updated stories** (`web/src/routes/_app/settings/stories/`):
- `-memory.stories.tsx` — added `Dirty` story (uses `StorybookFieldDirtySetup`).
- `-skills.stories.tsx` — added `Dirty` story; renamed empty story comment to call out the `@agh/ui` `Empty` baseline.
- `-providers.stories.tsx` — added `Dirty` story (with bespoke `StorybookProvidersDirtySetup` 2-stage helper) + renamed empty story comment.
- `-automation.stories.tsx` — added `Dirty` story.
- `-general.stories.tsx` — unchanged (already had Dirty/Saving + restart variants from task 30).

**Storybook helpers**:
- `web/src/storybook/settings-state-helpers.tsx` — added `StorybookFieldDirtySetup({ testId, value })` for reusable dirty-state seeding across any settings sub-route.

**Updated unit tests**:
- `web/src/routes/_app/settings/-skills.test.tsx` — added "renders the @agh/ui Empty card when no skills are disabled" test.
- `web/src/routes/_app/settings/-providers.test.tsx` — replaced "renders the empty state" with "renders the @agh/ui Empty card when the catalog is empty".

**Visual baselines** (`web/tests/visual/__snapshots__/`):
- Modified: `routes-app-settings-stories-providers--default`, `routes-app-settings-stories-providers--empty`, `routes-app-settings-stories-skills--default`, `routes-app-settings-stories-skills--disabled-empty`, `routes-app-settings-stories-skills--restart-banner` (Table + Empty primitive swaps).
- Added: `routes-app-settings-stories-automation--dirty`, `routes-app-settings-stories-memory--dirty`, `routes-app-settings-stories-providers--dirty`, `routes-app-settings-stories-skills--dirty`.

## Errors / Corrections

- Initial first-pass providers ActionResultBanner used a flex container with the dismiss button inline — looked janky next to `Alert`'s `has-data-[slot=alert-action]:pr-18` padding. Fixed by moving the dismiss button into the `AlertAction` slot which the primitive already absolutely-positions top-right.

## Ready for Next Run

- Task 32 (Settings batch 2 — `mcp-servers` / `hooks-extensions` / `observability` / `environments` / `network`) can reuse `StorybookFieldDirtySetup({ testId, value })` for any per-field dirty baseline and `Input/Table/Empty` swaps using the same patterns.
- For collection routes (mcp-servers / environments / hooks-extensions) that already have a Table — confirm whether the existing tables consume `@agh/ui` `Table` or roll their own; this task only swept the providers + skills tables.
- Providers row uses `StatusDot tone="success"` for installed providers (green dot) — consistent with other domain status helpers. If the visual feedback round preferred neutral tone for "installed", swap back to `tone="neutral"` in `providerStateTone`.
