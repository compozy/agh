# Task Memory: task_10.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build the `general`, `memory`, and `observability` settings pages on top of the shared shell (task_08) and `@/systems/settings` (task_09) with restart banners, save bars, and the Memory "Trigger now" consolidate action.

## Important Decisions
- Introduced `web/src/systems/settings/components/` with `SettingsPageShell`, `SettingsSectionCard`, `SettingsFieldRow`, `SettingsStatusLine`, `SettingsSaveBar`, and `SettingsRestartBanner` so later section pages (task_11+) can reuse the same presentational primitives.
- Route-level orchestration lives in per-section hooks under `web/src/hooks/routes/use-settings-{general,memory,observability}-page.ts` to satisfy the `compozy-react/max-component-complexity` oxlint rule and keep route files presentational.
- Memory "Trigger now" reuses `useConsolidateMemory` from `@/systems/knowledge` instead of introducing a duplicate adapter; the action button is gated on `actions.consolidate.available && health.dream_enabled` from the envelope.
- Observability log-tail is surfaced as a read-only capability card (transport + stream URL) linking to the existing `OBSERVABILITY_LOG_TAIL_PATH`; no new streaming UI is introduced in this task.

## Learnings
- The router plugin regenerates `src/routeTree.gen.ts` only via `vite build` (or `vite dev`); running `bun run vite build --mode development` is the fastest one-shot regeneration when adding new file-based routes.
- `tsgo` is strict about `as const` array literals for OpenAPI types (`available_scopes`); section envelope fixtures should be typed via `SettingsGeneralSection` etc. instead of relying on const inference.
- `useMutation.mutate(body, { onSuccess })` lets route-hook tests assert the envelope-driven `lastAppliedLabel` ("Saved · restart required to apply" vs "Saved · applied immediately") without inspecting mutation internals.
- Vitest module mocks of `@/systems/settings/adapters/settings-api` must also re-export `SettingsApiError` because the route-level hooks reference it for error classification.

## Files / Surfaces
- Added: `web/src/routes/_app/settings/{general,memory,observability}.tsx` + route tests `-general.test.tsx`, `-memory.test.tsx`, `-observability.test.tsx`.
- Added: `web/src/hooks/routes/use-settings-{general,memory,observability}-page.{ts,test.tsx}`.
- Added: `web/src/systems/settings/components/{index,settings-page-shell,settings-section-card,settings-field-row,settings-status-line,settings-save-bar,settings-restart-banner}.tsx` plus `settings-restart-banner.test.tsx`.
- Regenerated: `web/src/routeTree.gen.ts` (touched by the tanstack/router plugin).

## Errors / Corrections
- Initial route components failed oxlint's `max-component-complexity` rule; extracted the render logic into section sub-components and moved state to dedicated route hooks.
- Observability hook-level test rejected a `mock.mockRejectedValue(new Error(...))` until `SettingsApiError` was added to the mocked adapter module; hook tree-shakes via class-based `instanceof` check.

## Ready for Next Run
- task_11+ section pages (Skills / Automation / Network) can import the shared primitives from `@/systems/settings/components` and follow the hook-per-route pattern (`use-settings-<slug>-page.ts`) to keep complexity within lint thresholds.
- If a future task needs to ingest log-tail content directly, extend `settingsObservabilityLogTailPath()` / envelope `log_tail` metadata — the current UI already consumes both fields.
