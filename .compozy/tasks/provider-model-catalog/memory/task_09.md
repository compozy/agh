# Task Memory: task_09.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Web app consumes the daemon-owned model catalog for pre-session model selection, prefers active ACP `configOptions` when present, preserves manual model entry, and exposes catalog source status + refresh in Settings > Providers. Old `default_model`/`supported_models`/`supports_reasoning_effort` web assumptions are removed.

## Important Decisions

- Added a new `web/src/systems/model-catalog/` system instead of expanding `settings/` so the catalog has its own queryKeys, adapter, and barrel.
- Keep the session-create dialog pre-session: the active ACP `configOptions` consumer lives in the shared `deriveActiveSessionOptions` helper (model-catalog/lib) so any future in-session control inherits the same precedence rules without re-implementing them.
- Provider settings model metadata is preserved by snapshotting the original curated rows on the draft (`curated_snapshot`) and re-merging by ID at save; the textarea still edits ID-only lines but `supports_reasoning`/`reasoning_efforts`/etc. ride along.
- Settings provider card embeds `ProviderModelCatalogStatus` directly (uses `useProviderModelStatus` + `useRefreshProviderModels`); refresh-error and stale state are surfaced inline so the route page does not need a separate panel.

## Learnings

- `vi.mock("@/systems/model-catalog", …)` does not intercept calls made from the catalog hooks because `query-options.ts` imports the adapter file directly. Mock the adapter file (`@/systems/model-catalog/adapters/model-catalog-api`) when stubbing list/refresh/status.
- `useQuery` instances in this repo configure their own `retry` function (`shouldRetry`) which overrides the QueryClient default; tests that intend to surface errors must keep the rejection persistent (`mockRejectedValue`) and bump `waitFor` timeout above ~3s, or the per-query retry replays the success fixture.

## Files / Surfaces

- `web/src/systems/model-catalog/*` (new): types, adapter, query-keys/options, hooks, derive helper, tests
- `web/src/systems/session/hooks/use-session-create-dialog.ts`: catalog wiring + refresh action + stale/error/loading flags
- `web/src/systems/session/components/session-create-dialog.tsx`: stale/error/refresh status, distinct availability badges, refresh control
- `web/src/systems/session/components/model-command-select.tsx`, `reasoning-command-select.tsx`: structured options (catalog rows + reasoning options)
- `web/src/systems/settings/components/provider-card.tsx`, `provider-model-catalog-status.tsx`: per-provider catalog source list + refresh button
- `web/src/hooks/routes/use-settings-providers-page.ts`: snapshot-based curated metadata preservation
- `web/src/routes/_app.tsx`: passes new dialog props
- Tests updated: hook test, dialog component test, settings hook test, route test, app-layout test (model-catalog mocks added)

## Errors / Corrections

None during execution.

## Ready for Next Run

Task 09 complete and pushed only locally. The model-catalog system is reusable for any future in-session ACP `configOptions` UI; the helper already prefers ACP values over catalog metadata.
