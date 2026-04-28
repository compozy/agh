---
name: task_12 — Providers and Environments collection pages
description: Implementation notes for the providers and environments settings collection pages
type: project
---

# Task Memory: task_12.md

## Objective Snapshot

- Build `/_app/settings/providers` and `/_app/sandbox` pages with list/detail/edit/delete flows.
- Factor shared collection-editor and delete-dialog primitives so task_13 (mcp-servers) and task_14 (hooks) can reuse them.
- Make PUT=full-replacement, DELETE overlay-reveals-builtin, and source precedence visible in the UI.

## Important Decisions

- Shared collection primitives live in `@/systems/settings/components/` next to the existing shell primitives. The new pieces are `SettingsCollectionHeader`, `SettingsSourceBadge`, `SettingsEditorDialog`, `SettingsDeleteDialog`.
- Editor dialogs own validation messaging, warnings, and loading spinner; page hooks own the draft state machine and mutation wiring.
- Editor state is a tagged union `closed | create | edit` so the selected entry stays pinned on validation/conflict errors (delete target uses the same shape).
- Providers use a table layout; environments use a card grid — the Paper designs make them structurally different, so sharing list rendering would add abstraction without benefit.
- Environment draft preserves nested profile keys (`daytona`, `network`, `env`) and replays them in the PUT so edits don't silently strip them; the dialog renders a "preserved on save" hint when any exist.
- Delete button is disabled when the effective source is `builtin-provider` (no overlay to remove); the same rule applies to environments' builtin profiles.
- The fallback note in the delete dialog is only shown when the entry has a `fallback` payload (builtin revealed after overlay delete). Environments get a usage-count note instead.

## Learnings

- `editor.mode` narrowing: once `if (editor.mode === "closed") return null;` runs, the remaining type is `"create" | "edit"`. Re-checking `=== "closed"` later trips TS2367. Derive `draft` directly from the narrowed union.
- `useMutation.reset()` before opening/closing dialogs keeps `mutation.data`/`error` from leaking across open/close cycles (otherwise previous warnings reappear on the next open).
- Route tests that mock `@/hooks/routes/use-settings-<slug>-page` don't need to stand up QueryClient — use `vi.mock(... useSettingsXPage: () => pageState)` and drive state via `beforeEach`. Hook tests that exercise actual adapters mock `@/systems/settings/adapters/settings-api` and use the real QueryClient wrapper.
- Last-action banners belong in the page hook state (not in a store) because they're cleared on dismiss and otherwise tied to mutation lifecycle.

## Files / Surfaces

- `web/src/systems/settings/components/settings-source-badge.tsx` (new) — effective + shadowed source pills
- `web/src/systems/settings/components/settings-collection-header.tsx` (new) — eyebrow/summary/action row
- `web/src/systems/settings/components/settings-editor-dialog.tsx` (new) — shared create/edit modal
- `web/src/systems/settings/components/settings-delete-dialog.tsx` (new) — shared confirm with fallback slot
- `web/src/systems/settings/components/index.ts` (updated) — export new primitives + `SettingsSource` type
- `web/src/hooks/routes/use-settings-providers-page.ts` (new)
- `web/src/hooks/routes/use-settings-environments-page.ts` (new)
- `web/src/routes/_app/settings/providers.tsx` (new)
- `web/src/routes/_app/sandbox.tsx` (new)
- `web/src/routeTree.gen.ts` (manual edit) — added providers + environments routes to every map/union (vite plugin regenerates only during dev/build)
- Tests: `-providers.test.tsx`, `-environments.test.tsx`, `use-settings-providers-page.test.tsx`, `use-settings-environments-page.test.tsx`, plus unit tests for the three new shared primitives

## Errors / Corrections

- Initial hook returned a `LastAction` union type with `| null` which leaked `null` into render code; switched to `ProviderLastAction` plus a module-local `LastAction` alias and import the plain union from the page.
- Environment editor list was nested inside `{draft ? (...) : null}` even though `draft` was always defined after the early-return guard; removed the redundant conditional to fix TS2367 and keep JSX flat.
- Imported `@tanstack/react-query`'s `useQuery` plus `settingsGeneralOptions` early, then removed that coupling — the collection envelopes already carry enough metadata for the status line, and extra section queries would add redundant traffic.

## Ready for Next Run

- task_13 (mcp-servers) can build on the same primitives. The editor dialog accepts any child body, so the scoped target/scope/workspace pickers fit cleanly. The delete dialog's `fallbackNote` slot can carry the "lower-precedence source becomes effective" message after MCP overlay delete.
- task_14 (hooks & extensions) can reuse `SettingsCollectionHeader`, `SettingsSourceBadge`, and both dialogs for the hook list.
- Route-tree edits in `web/src/routeTree.gen.ts` will be overwritten on the next `web-dev`/`web-build`; the generator picks up the new files automatically.
