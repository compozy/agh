# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Build `web/src/systems/settings` as the canonical frontend domain (types, adapters, query-keys/options, hooks, barrel) plus `web/src/hooks/routes/use-settings-page.ts` with restart polling and section state.

## Important Decisions
- SETTINGS_SECTIONS metadata moved to `systems/settings/lib/sections.ts`; `routes/_app/settings.tsx` re-exports SETTINGS_SECTIONS / SETTINGS_ROOT_PATH plus `SettingsSection` alias so prior tests remain stable.
- Restart polling state lives in a Zustand store (`settings-restart-store.ts` + `use-settings-restart-store.ts`) so the shell and section pages stay in sync across navigations; status polling stops as soon as `ready`/`failed` terminal states are reached.
- Mutation hooks call `recordMutation` in `onSuccess` and invalidate only the affected section or collection family (providers root + named detail, environments root + named detail, hooks root + hooks-extensions section, full mcp-servers root for scope parity).
- Query keys scoped MCP lists by `scope` + `workspace_id` to keep global and workspace caches isolated while sharing one adapter.

## Learnings
- Generated OpenAPI response types under `updateSettings*` 200s share the same mutation envelope, so `SettingsMutationResult = OperationResponse<"updateSettingsGeneral", 200>` is a safe canonical alias for `write_target`, `behavior`, and restart metadata fields across every section.
- `queryOptions.refetchInterval` accepts a `(query) => number | false` signature; returning `false` on terminal status is the idiomatic way to stop polling without mutating state.
- `tsgo --noEmit` (invoked via `make web-typecheck`) enforces strict type re-export rules — `PendingSettingsMutation` had to be sourced from the store module (where it is declared), not from `types.ts`.

## Files / Surfaces
- Added: `web/src/systems/settings/{types,index}.ts`, `adapters/settings-api.ts`, `lib/{sections,restart-status,query-keys,query-options}.ts`, `hooks/{use-settings-sections,use-settings-collections,use-settings-mutations,use-settings-restart}.{ts,tsx}`, `stores/{settings-restart-store,use-settings-restart-store}.ts`, co-located `*.test.{ts,tsx}` files.
- Added: `web/src/hooks/routes/use-settings-page.{ts,test.tsx}`.
- Modified: `web/src/routes/_app/settings.tsx` (imports SETTINGS_SECTIONS from `@/systems/settings` and keeps re-export for compatibility).

## Errors / Corrections
- Initial `PendingSettingsMutation` re-export was placed in `types.ts` exports of `index.ts`; typecheck caught this and the export was moved under the stores section where the type actually originates.

## Ready for Next Run
- Downstream section pages (task_10+) should consume `@/systems/settings` for data and `@/hooks/routes/use-settings-page` for shared restart + active-section state.
- Restart banner UI can subscribe to `useSettingsRestart()` (hook) or `useSettingsRestartStore` (store) directly; the route-level shell can render a banner from `useSettingsPage().restart`.
- MCP collection pages must pass an explicit `{ scope, workspace_id, target }` filter — adapters trim whitespace and propagate values to both query keys and network calls.
