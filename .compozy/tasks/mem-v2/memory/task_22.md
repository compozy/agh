# Task Memory: task_22.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Replaced the speculative `InspectorMemoryDoc[]` model with a Memory v2 forensic session ledger surface in the inspector memory tab. The inspector now renders ledger meta lineage (workspace, root/parent session, spawn depth, path, checksum, version, created/stopped) plus ledger event metadata (sequence + event_type + emitted_at), with truthful loading / unavailable / error states.
- Source of truth is the generated `getMemorySessionLedger` operation already exposed in `web/src/generated/agh-openapi.d.ts`; no new endpoints, no new write paths.

## Important Decisions

- Modeled a single `InspectorMemoryState` prop with optional `ledger`, `isLoading`, `error`. The route owns the TanStack Query call (`useSessionLedger`) and feeds the inspector — components stay pure/presentational per `web/CLAUDE.md`.
- Treated `SessionLedgerUnavailableError` (404) as the truthful "not yet materialized" empty state, not as an error. Non-404 errors render the dedicated forensic error empty.
- Did not render arbitrary `event.payload` JSON. Only contract-approved redaction-safe fields (`sequence`, `event_type`, `emitted_at`) are surfaced.
- Refactored `agents.$name.sessions.$id.tsx` to export `SessionPage` directly (`export function SessionPage()`) — replacing `export { SessionPage }` because rolldown bundling rejected the latter form. The vitest spec now imports `SessionPage` directly without `Route as unknown as`.
- Added `sessionLedgerOptions` with `retry: false` for `SessionLedgerUnavailableError` so the truthful empty state appears immediately for not-yet-materialized sessions instead of retrying.
- Remediation: route gates `useSessionLedger` on `session.state === "stopped"` rather than firing immediately. Materialization only happens at `OnSessionEnd` (ADR-006), so calling the API earlier is a guaranteed 404 that cached and prevented a natural refetch on the active→stopped transition.
- Remediation: kept the Memory tab name. Only the inner events panel is renamed to "Ledger events" because its content is the full ordered ledger event stream (transcript + memory + lifecycle + redaction metadata) per ADR-006, not memory events alone.

## Learnings

- `tanstack/router` route files must use `export function FooPage()` directly when paired with a re-export; declaring a separate `export { FooPage }` triggers a rolldown parse-error during the production build even when tsc is happy. Reproduced in `make web-build` and confirmed against the existing `MemorySettingsPage` shape.
- `oxlint` flags `...(maybe ?? {})` in object spreads as `no-useless-fallback-in-spread`; spreading a possibly-undefined value with `...maybe?.field` is the canonical fix.
- The default `bottomTab` in the inspector is already `"memory"`, so component tests only need an explicit click for safety; the section mounts by default.

## Files / Surfaces

- `web/src/systems/session/components/session-inspector.tsx` — replaced `MemorySection` with ledger-aware meta + events panels and wired the new `InspectorMemoryState`/`InspectorSessionLedger` types. After remediation: events panel header/empty-state copy renamed from "Memory events" to "Ledger events" so it matches the ADR-006 contract (full session ledger event stream, not memory events alone). Memory tab itself unchanged.
- `web/src/systems/session/components/session-inspector.test.tsx` — covers ready / unavailable / 404 / loading / error / read-only / events-empty states. Remediation added: assertion that the events panel uses "Ledger events" wording and renders non-memory event types (`session.started`, `transcript.user`, `session.stopped`).
- `web/src/systems/session/adapters/session-api.ts` — added `fetchSessionLedger` and `SessionLedgerUnavailableError`.
- `web/src/systems/session/adapters/session-api.test.ts` — remediation added focused tests for `fetchSessionLedger`: success path, 404 → `SessionLedgerUnavailableError`, non-404 → `SessionApiError`, abort signal forwarded.
- `web/src/systems/session/lib/query-keys.ts` — added `ledger(id)` key.
- `web/src/systems/session/lib/query-options.ts` — `sessionLedgerOptions` accepts `{ enabled }` and combines it with `!!id` so the query only enables when both the id and the caller's lifecycle gate are satisfied.
- `web/src/systems/session/hooks/use-sessions.ts` — `useSessionLedger(id, { enabled })` accepts an explicit lifecycle gate. The doc-comment makes it explicit that the caller must wait for `session.state === "stopped"`.
- `web/src/systems/session/types.ts` — added `SessionLedgerResponse` / `SessionLedgerMeta` / `SessionLedgerEvent` (generated-contract typed).
- `web/src/systems/session/index.ts` — barrel exposes the hook, options, adapter, error class, and types.
- `web/src/routes/_app/agents.$name.sessions.$id.tsx` — gates `useSessionLedger` with `enabled: session.state === "stopped"` so the query never fires while the session is active/starting/stopping. The first fetch happens when `sessionDetailOptions` polling reports a stopped state, avoiding the stale-404 cache poisoning bug.
- `web/src/routes/_app/-agents.$name.sessions.$id.test.tsx` — mocks `useSessionLedger(id, options)`. Asserts `enabled: false` for `active`/`starting`/`stopping`, `enabled: true` for `stopped`, and continues to verify ledger state forwarding (success + loading).

## Errors / Corrections

- First attempt used `export { SessionPage };` after `Route = createFileRoute(...)`; `make web-build` failed with `Export 'SessionPage' is not defined` from rolldown. Fixed by switching to `export function SessionPage() { ... }`.
- First test fixture used `...(overrides?.meta ?? {})` for spread; `bunx oxlint` flagged the empty-object fallback. Replaced with `...overrides?.meta`.
- Remediation 1: original copy labelled the events panel "Memory events" / "No memory events" even though the panel renders the full `ledger.jsonl` event stream (transcript + memory + lifecycle + redaction metadata per ADR-006). Renamed to "Ledger events" / "No ledger events" to keep the inspector truthful as a forensic ledger consumer. The Memory tab name itself stays — only the panel inside it is renamed.
- Remediation 2: `useSessionLedger(sessionId)` fired immediately even while the session was active. For an active session the ledger never exists, so the query cached a `SessionLedgerUnavailableError` (404) before the `OnSessionEnd` materializer had a chance to write `ledger.jsonl`. Fixed by adding an `enabled` flag through `sessionLedgerOptions` / `useSessionLedger` and gating from the route on `session.state === "stopped"`. The first fetch now happens on the active→stopped transition (driven by `sessionDetailOptions` polling), so we never poison the cache with a stale 404 while the session is still live.

## Ready for Next Run

- `make verify` passes (Go + Bun monorepo). Focused vitests for the inspector, route, and adapter all green after remediation.
- Next phase: task 23 (runtime / sessions docs in `packages/site`) can rely on these ledger semantics being live in the inspector.
