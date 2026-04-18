# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
Combined Hooks & Extensions settings page delivering config-backed hook toggles, extension marketplace/policy edits (restart-required), and operational extension enable/disable (immediate) against `/api/settings/hooks-extensions` and the HTTP-visible `/api/extensions` parity.

## Important Decisions
- Kept the extension adapters and hooks co-located in `@/systems/settings` (not a new `extensions` system) because the Hooks & Extensions screen is the only consumer and task spec frames them as a supporting operational surface.
- Used `required: false/true` in hook declarations to drive the enable toggle — there is no separate `enabled` field in the OpenAPI hook schema; `required` is the closest semantic and is the field the backend reads to register the hook in the resolved graph.
- Kept extension mutations out of the settings restart banner (no `recordMutation` call) because their behavior is `action_trigger`, not a settings-config mutation; they should never ask for a daemon restart.
- Envelope `installed` summaries are the fallback when `/api/extensions` returns an empty list (or before it loads) so the page always renders something; the live extensions query still drives enable/disable targets when available.
- Re-exported `SettingsHooksExtensionsHook`, `SettingsHooksExtensionsInstalled`, `SettingsHooksExtensionsTransportParity`, and `SettingsExtensionEntry` from `@/systems/settings` public barrel to keep the route presentational.

## Learnings
- The shadcn `Switch` (Base UI) reports disabled via `aria-disabled`, not the HTML `disabled` attribute, so route tests must assert `aria-disabled` instead of `.toBeDisabled()` for switches.
- `RateLimit` shapes (`operator_write_rate_limit`, `snapshot_rate_limit`) are required objects in the PATCH body — the policy draft must always include them even when the user only edits the marketplace fields.
- The `declaration.required` default is `true` in OpenAPI; treat only explicit `false` as disabled when computing `hooksCounts.enabled`.

## Files / Surfaces
- `web/src/systems/settings/types.ts` — new envelope-derived types + `SettingsExtensionEntry`.
- `web/src/systems/settings/adapters/settings-api.ts` — `listSettingsExtensions`, `enableSettingsExtension`, `disableSettingsExtension`.
- `web/src/systems/settings/hooks/use-settings-collections.ts` — `useSettingsExtensions`.
- `web/src/systems/settings/hooks/use-settings-mutations.ts` — `useEnableSettingsExtension`, `useDisableSettingsExtension`, shared `invalidateExtensions` helper.
- `web/src/systems/settings/lib/{query-keys,query-options}.ts` — extensions query key + options.
- `web/src/systems/settings/index.ts` — public barrel exports for the new types/hooks/adapters.
- `web/src/hooks/routes/use-settings-hooks-extensions-page.ts` + test — orchestration hook + unit tests.
- `web/src/routes/_app/settings/hooks-extensions.tsx` + `-hooks-extensions.test.tsx` — new route + component tests.
- `web/src/routeTree.gen.ts` — mirrored the manual route registration (tanstack plugin only regenerates during dev/build).
- Updated `use-settings-mutations.test.tsx`, `settings-api.test.ts`, `query-keys.test.ts` for the new surfaces.

## Errors / Corrections
- Initial route test asserted `.toBeDisabled()` on the extension toggle which failed because shadcn `Switch` uses `aria-disabled`; switched to `toHaveAttribute("aria-disabled", "true")`.

## Ready for Next Run
- Backend task_04/task_05 already emit the `transport_parity` object and installed summaries consumed here — no contract changes required.
- Extension enable/disable is the first settings surface that calls `POST /api/extensions/{name}/enable|disable` from the web; future audit-style views should reuse `useSettingsExtensions` plus the `invalidateExtensions` helper rather than forking the extensions fetch.
