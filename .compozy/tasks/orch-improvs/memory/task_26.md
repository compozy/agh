# Task Memory: task_26

## Status

Completed on 2026-05-05 with explicit PASS evidence from `make verify`.

## Objective Snapshot

- Implement the web data layer for task orchestration, review state, context bundles, SSE resume data, and bridge notification diagnostics.
- The task was `type: frontend`; the local Codex loop dispatched implementation to Claude Opus.
- Completion required generated OpenAPI-derived types, TanStack Query adapters/hooks, MSW fixtures, focused loading/empty/error/mutation tests, web gates, and final `make verify`.

## Important Decisions

- Used generated OpenAPI contract helpers from `web/src/lib/api-contract.ts` and `web/src/generated/agh-openapi.d.ts`; no duplicate hand-written DTO shapes.
- Preserved the existing `web/src/systems/tasks` dependency flow: `adapters -> lib/query-* -> hooks -> components`, with explicit public exports in `index.ts`.
- Kept implementation scoped to the web data layer and fixtures/tests; no other generated artifact required regeneration.
- Profile and bridge subscription reads use the default cadence (15s stale, 30s refetch); reviews and context bundle use the live cadence (5s stale, 15s refetch). Empty-id queries are disabled via `enabled` guards.
- Mutation hooks invalidate the narrow related keys plus task detail, timeline, run details, lists, dashboard, inbox, agent context, and context bundle when the mutation can affect those projections.
- Bridge notification cursor diagnostics fixtures cover both populated and zero-state cursors per task 25 lifecycle semantics.
- Task-level review status enums use the canonical domain set (`requested|routed|in_review|recorded|circuit_opened|canceled`); UI does not surface `approved`/`rejected` as statuses (those are outcomes per task 19/22).

## Learnings

- `task_26` depended on completed tasks 18, 19, 23, 24, and 25; the generated contract already contained execution profile, run review, task context bundle, `latest_event_seq`, task stream, and bridge notification subscription diagnostics.
- Existing task fixtures already included `latest_event_seq`; new fixtures extend the same generated-contract-derived fixture pattern.
- MSW-backed hook tests need a `globalThis.fetch` stub that delegates to `getResponse(handlers, request, { baseUrl: window.location.origin })` because `apiClient` is created with `baseUrl = window.location.origin` and openapi-fetch passes the absolute URL into the runtime fetch. Stubbing `vi.stubGlobal("fetch", fn)` works the same way the existing per-adapter tests do.
- `make verify` exhibited transient SQLite `context deadline exceeded` failures in `internal/heartbeat`, `internal/observe`, `internal/soul`, and `internal/store/globaldb` under the `-race -parallel=4` lane. Focused re-runs (`go test -race -count=1 -parallel=2`) and a second `make verify` invocation passed cleanly. This is consistent with the documented load-induced SQLite contention pattern in workflow memory.

## Files / Surfaces Touched

- `web/src/systems/tasks/types.ts` — added profile, review, bridge notification, agent context bundle, and stream-timeline aliases.
- `web/src/systems/tasks/adapters/tasks-api.ts` — added 11 typed adapter functions covering profile (3), reviews (5), agent context (2), and bridge notifications (4); plus exported `buildTaskStreamUrl` for cursor-seeded `/api/tasks/{id}/stream` URL construction.
- `web/src/systems/tasks/lib/query-keys.ts` — added hierarchical keys for profile, reviews (run/task/detail), agent context, context bundle, stream, and bridge notifications (root/list/detail).
- `web/src/systems/tasks/lib/query-options.ts` — added 8 `queryOptions` factories with cadence + enabled guards.
- `web/src/systems/tasks/hooks/use-task-profile.ts`, `use-task-reviews.ts`, `use-task-context-bundle.ts`, `use-task-notifications.ts` — read + mutation hooks per surface.
- `web/src/systems/tasks/hooks/use-task-stream.ts` — cursor-seeded `EventSource` hook with optional factory, `onEvent`/`onError` callbacks, SSR/disabled/empty-id/missing-EventSource guards, parses generated `TaskStreamPayload`, invalidates task detail/timeline/runs/run-detail/lists/dashboard/inbox/context-bundle/agent-context/reviews/bridge-notification roots without inferring authority, registers `addEventListener` for the canonical named task SSE event types (`task.created/updated/published/approved/rejected/canceled/child_created/dependency_added/dependency_removed/run_enqueued/run_claimed/run_starting/run_session_bound/run_started/run_completed/run_failed/run_canceled/run_force_stopped/run_recovered/run_rejected/run_lease_extended/run_lease_expired/run_released/execution_profile_updated/execution_profile_deleted/run_review_requested/run_review_bound/run_review_recorded/run_review_approved/run_review_rejected/run_review_blocked/run_review_error/run_review_timeout/run_review_invalid_output/run_review_retry_enqueued/run_review_circuit_opened/run_review_canceled/notification_delivered`) so EventSource actually delivers the daemon's `event: <type>` frames, keeps `onmessage` for defensive unnamed-frame support, and removes named listeners on cleanup when `removeEventListener` is present before closing the source.
- `web/src/systems/tasks/mocks/fixtures.ts` — added builders and constants for execution profile, run reviews, verdict result, bridge notification subscription cursor, context bundle, and agent context.
- `web/src/systems/tasks/mocks/handlers.ts` — added MSW handlers for all 11 new operations.
- `web/src/systems/tasks/mocks/index.ts` — extended exports.
- `web/src/systems/tasks/index.ts` — extended public barrel including `buildTaskStreamUrl`, `useTaskStream`, and the stream hook option/factory types.
- `web/src/systems/tasks/adapters/tasks-api-orchestration.test.ts` — adapter coverage for all 11 operations.
- `web/src/systems/tasks/lib/query-keys.test.ts`, `lib/query-options.test.ts` — extended with orchestration-specific tests.
- `web/src/systems/tasks/mocks/fixtures.test.ts` — orchestration fixture contract assertions.
- `web/src/systems/tasks/hooks/use-task-orchestration-hooks.test.tsx` — adapter-mock hook coverage.
- `web/src/systems/tasks/hooks/use-task-orchestration-msw.test.tsx` — MSW-backed adapter+query stack coverage.
- `web/src/systems/tasks/hooks/use-task-stream.test.tsx` — URL builder coverage (encoded ids, `after_sequence=0`, missing seed, empty id rejection) and cursor-seeded EventSource lifecycle (factory called with seeded URL, **named-SSE** payload parsing + query invalidation through `addEventListener("task.run_started", ...)` and listener registration assertions for `task.run_review_requested` / `task.notification_delivered`, defensive unnamed-message fallback through `onmessage`, named-SSE malformed payload routes through `onError`, connection error forwarding, factory reuse on identical inputs, cleanup removes named listeners when `removeEventListener` is present and still closes the source via a no-removeEventListener fixture).

## Errors / Corrections

- The first MSW-backed hook test draft used a hardcoded `baseUrl: "http://localhost"`, which mismatched the `apiClient` baseUrl built from `window.location.origin` in jsdom. Fixed by deriving `baseUrl` from `window.location.origin` (or "http://localhost" when window is undefined) and replacing the `vi.stubGlobal` with a direct `globalThis.fetch` swap restored in `afterAll`.
- Local Codex audit caught a missing piece in the first delegation: stream generated types and `tasksKeys.stream(...)` were present, but no exported task stream hook or URL helper for cursor-seeded `/api/tasks/{id}/stream`. Added `buildTaskStreamUrl` (typed query encoder over `TaskStreamFilter.after_sequence`) and `useTaskStream` (cursor-seeded EventSource with parse+invalidate, SSR/disabled/empty-id guards, optional factory + onEvent/onError, cleanup closes source). Tests cover URL shape (including `after_sequence=0` and encoded ids), disabled/empty-id branches, parse + invalidation path, malformed payload + connection error routing, and factory reuse on identical inputs.
- Second Codex audit pass uncovered a runtime regression in `useTaskStream`: AGH task SSE writes named events through `internal/api/core/sse.go` (`WriteTaskStreamEvent` sets `event: <task event type>`), but the hook only assigned `source.onmessage = handleMessage`. `EventSource` never routes named SSE events to `onmessage`; they only reach listeners registered via `addEventListener("<type>", ...)`. With the default ACP/runtime emitting frames like `event: task.run_started` / `event: task.run_review_requested` / `event: task.notification_delivered`, the hook would have silently dropped real updates. Fix: register `addEventListener` for the full canonical task event-type list (orchestration + review-gate + bridge-notification + execution-profile), keep `onmessage` for defensive unnamed-frame parity, and on cleanup call `removeEventListener` for each named type when present before closing the source.

## Verification Evidence

- `make web-lint`: 0 warnings / 0 errors.
- `make web-typecheck`: passed.
- `make web-build`: passed (Vite production build).
- `bun run test` (web Vitest project): 205 files / 1583 tests passing.
- `make verify` (full monorepo): PASS — race gate `DONE 8283 tests in 87.403s`, `OK: all package boundaries respected`. Earlier transient SQLite timeouts under load confirmed flaky; focused reruns and a second `make verify` succeeded.
- Local Codex audit after delegation: `make verify` PASS without reduced parallelism or environment overrides. Evidence: Bun/Vitest gate `332 passed (332)` / `2150 passed (2150)`, web build PASS, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 75.399s`, and `OK: all package boundaries respected`.
- Stream-hook correction reverification: focused `vitest run src/systems/tasks/hooks/use-task-stream.test.tsx` `1 file / 11 tests passed`, `make web-lint` 0 warnings / 0 errors, `make web-typecheck` PASS, `make web-test` `206 files / 1594 tests passed`, `make web-build` PASS, full `make verify` `333 files / 2161 tests passed`, race gate `DONE 8283 tests in 61.850s`, `OK: all package boundaries respected`.
- Named-SSE correction reverification: focused `bunx vitest run src/systems/tasks/hooks/use-task-stream.test.tsx` `1 file / 13 tests passed`, `make web-lint` 0 warnings / 0 errors, `make web-typecheck` PASS, `make web-test` `206 files / 1596 tests passed`, `make web-build` PASS, full `make verify` `333 files / 2163 tests passed`, race gate `DONE 8283 tests in 10.605s`, `OK: all package boundaries respected`.
- Final local Codex verification after audit/comment normalization: `make verify` PASS on the current worktree, with Bun/Vitest `333 passed (333)` / `2163 passed (2163)`, web build PASS, `golangci-lint` `0 issues`, Go race gate `DONE 8283 tests in 66.726s`, and `OK: all package boundaries respected`.

## Ready for Next Run

- Tasks 27 (web UI), 28 (orchestration profile docs), and 29 (review-gate / cursor docs) can now consume the public exports from `@/systems/tasks` for execution profile, reviews, context bundle, bridge notification subscriptions, and stream resume metadata without duplicating contract types.
