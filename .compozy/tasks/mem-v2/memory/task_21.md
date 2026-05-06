# Task Memory: task_21.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

Render the entire backend Memory v2 settings payload (`GET /api/settings/memory`) on `/settings/memory` with truthful controls, refresh tests/mocks/stories, and remove the lingering `Route as any` cast in the route test.

## Important Decisions

- Exported `MemorySettingsPage` as a named export from `web/src/routes/_app/settings/memory.tsx` so the vitest spec imports the component directly. The vitest mock now stubs `createFileRoute` to a passthrough opts factory; no `as any` cast remains.
- The hook drops the legacy `consolidate`/`isConsolidating` API and exposes `handleTriggerDream`/`isTriggeringDream`. The action message now reads "Dream triggered", the backend `reason` when returned, or "Dream not triggered" as the fallback, aligning with the techspec hard cut from `consolidate` to `dream trigger`.
- Added `web/src/systems/settings/components/settings-decimal-input.tsx` for bounded weight/score editing. It mirrors `SettingsNumberInput` but accepts decimals with optional precision/min/max. Existing integer fields still flow through the integer-only `SettingsNumberInput`.
- Truthful read-only inputs (operator cannot mutate them safely) cover `extractor.inbox_path`, `extractor.dlq_path`, `session.ledger_root`, `session.unbound_partition`, `daily.rotate_format`, `daily.archive_path`, `workspace.toml_path`, and `controller.policy.allow_origins`. Every other backend key is editable through real input controls.
- The Trigger dream button is disabled unless `envelope.actions.consolidate.available && envelope.health.dream_enabled && draft.dream.enabled`, matching the backend action availability surface.

## Learnings

- The OpenAPI generator already produces the full Memory v2 payload (`getSettingsMemory` 200 + `updateSettingsMemory` request), so types/adapters needed no manual changes — only UI orchestration needed updating.
- `useSettingsMemoryPage` round-trips the entire `MemoryConfig` via `JSON.stringify` deep compare, so adding new fields to the form does not require new dirty/save plumbing.
- `vi.mock("@tanstack/react-router", () => ({ createFileRoute: () => (opts) => opts }))` keeps `Route = createFileRoute(...)({ component })` valid at module load while letting the spec import the component as a named export.
- Stories live outside the vitest runner (vitest only includes `*.test.{ts,tsx}` and `*.spec.*`), so the storybook play-test can import the new test ids without affecting the test gate.

## Files / Surfaces

- `web/src/routes/_app/settings/memory.tsx` — full Memory v2 surface across system, provider resilience, controller, controller LLM, recall, decisions, extractor, dream, session ledger, daily, file caps, and workspace identity sections; named-export component.
- `web/src/hooks/routes/use-settings-memory-page.ts` — `handleTriggerDream` + `isTriggeringDream` rename; truthful action messages.
- `web/src/hooks/routes/use-settings-memory-page.test.tsx` — updated assertion against the renamed handler.
- `web/src/routes/_app/settings/-memory.test.tsx` — imports `MemorySettingsPage` directly; covers controller/recall/extractor/decisions/dream/session/daily/file caps/workspace/provider sections plus dream availability gating.
- `web/src/routes/_app/settings/stories/-memory.stories.tsx` — story renamed to `DreamTriggered` and asserts the new "Dream triggered" copy via the new test id `settings-page-memory-dream-trigger`.
- `web/src/systems/settings/components/settings-decimal-input.tsx` — new decimal input with bounded validation.
- `web/src/systems/settings/components/index.ts` — exports `SettingsDecimalInput`.

## Errors / Corrections

- No implementation corrections. Local orchestration corrected this memory note after delegation so the recorded action-message behavior and focused-test evidence match the actual hook and logs.

## Validation evidence

- `cd web && bunx vitest run src/routes/_app/settings/-memory.test.tsx` → 1 file / 11 tests pass.
- `cd web && bunx vitest run src/hooks/routes/use-settings-memory-page.test.tsx src/systems/settings/adapters/settings-api.test.ts src/systems/settings/lib/query-options.test.ts src/systems/settings/hooks/use-settings-mutations.test.tsx` → 4 files / 43 tests pass.
- `make web-lint` → 0 warnings / 0 errors.
- `make web-typecheck` → clean.
- `make web-test` → 205 files / 1554 tests pass.
- `make web-build` → builds cleanly.
- `make verify` → 8359 Go tests pass + boundaries respected.

## Ready for Next Run

- 2026-05-05 task 21 completed with full `make verify` PASS. Next Phase B iteration should execute `task_22` (Web Session Inspector Memory Surface) with tasks `22-26` still pending after final loop gate.
