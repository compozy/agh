# Task Memory: task_11.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Delivered settings routes and orchestration hooks for `skills`, `automation`, and `network`, layered on `@/systems/settings` primitives from task_09 and the page-shell pattern from task_10.
- Skills page splits save flow into two independent mutations: `applied_now` disabled-skills card and `restart_required` marketplace-policy card, keeping each diff isolated so `ClassifyMutation` never mixes behaviors.
- Automation and network pages use the single-envelope save flow (all-fields restart-required) — matches the runtime-apply matrix.

## Important Decisions
- Split skills page into two save slots instead of one unified save bar because `internal/settings.ClassifyMutation` rejects payloads that mix `skills.disabled_skills` (applied_now) with any other `skills.*` change (restart_required). Single-shot save would fail server-side the moment users edit both at once; per-card saves keep each diff in a single behavior bucket.
- Operational deep links are hardcoded per page (`to="/skills"`, `/automation`, `/network`) instead of iterating `envelope.links` — TanStack Router's `Link.to` is a typed union of known paths and each settings page has exactly one known operational target.
- Skills hook uses two `useUpdateSettingsSkills()` mutation instances (one per card) so independent pending/error/data states don't clobber each other while both writes share the same section invalidation.

## Learnings
- `routeTree.gen.ts` must be regenerated manually when adding child routes because the vite plugin only runs during dev/build. Mirror the existing pattern (import, `.update({id, path, getParentRoute})`, add to `FileRoutesByFullPath`, `FileRoutesByTo`, `FileRoutesById`, `FileRouteTypes`, the module declaration, and the `AppSettingsRouteChildren` interface plus constant).
- Batching `setDraft` and a save handler inside the same `act()` in tests captures the stale draft because React callbacks close over the previous render's state — wrap state update and handler in separate `act()` calls when testing mutation payload shape.
- Route files prefixed with `-` (for example `-skills.test.tsx`) are treated as co-located non-route modules by TanStack Router's file-based routing plugin, so tests can live next to the route module without polluting the route tree.

## Files / Surfaces
- `web/src/routes/_app/settings/{skills,automation,network}.tsx` — new settings pages.
- `web/src/hooks/routes/use-settings-{skills,automation,network}-page.ts` — new orchestration hooks.
- `web/src/routes/_app/settings/-{skills,automation,network}.test.tsx` — route render tests.
- `web/src/hooks/routes/use-settings-{skills,automation,network}-page.test.tsx` — hook tests.
- `web/src/routeTree.gen.ts` — added three child routes under `/settings`.

## Errors / Corrections
- Initial skills hook tests failed because `toggleDisabled` (sets draft state) plus `handleSaveDisabled` (captures draft via closure) were batched in a single `act()`. Fix: split into sequential `act()` calls.
- First skills page draft referenced a missing `SkillsSaveCard` component; replaced with inline `PolicySection` + shared `SaveControls` component so policy save stays colocated with its fields.

## Ready for Next Run
- Tasks 12–14 should follow the same `use-settings-<slug>-page` hook + `SettingsPageShell` + `SettingsSaveBar` pattern; for any section with mixed `applied_now` / `restart_required` behavior, split the save into separate mutation slots (see skills page) to avoid `ClassifyMutation` mixed-behavior rejection.
