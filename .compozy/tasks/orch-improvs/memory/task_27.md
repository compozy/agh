# Task Memory: task_27

## Status

Completed on 2026-05-05 with explicit PASS evidence from `make verify`.

## Objective Snapshot

- Implement web UI surfaces for task execution profiles, review state/verdict guidance, and bridge notification cursor/SSE diagnostics.
- Use the task 26 web data layer exports from `web/src/systems/tasks` instead of duplicating generated DTOs or adapters.
- Keep UI truthful: execution profiles are typed task-owned overlays, review verdict authority remains task-service persisted state, notification cursors are delivery progress only, and SSE resume state comes from `latest_event_seq`/stream hook state rather than fake metrics.

## Important Decisions

- Implementation followed the route → page hook → presentational components pattern from `web/CLAUDE.md`. Components stay pure; data + mutations live in `@/hooks/routes/use-task-detail-orchestration-tab.ts`, and the route file `tasks.$id.tsx` plumbs props into a single composite `TasksDetailOrchestrationPanel`.
- Added a dedicated `Orchestration` tab on `tasks.$id` housing four cards (Execution Profile, Reviews, Bridge Notifications, Stream Resume) so existing route layout stays dense without inventing a parallel page.
- Run-level reviews show inside the run detail route via the same presentational `TasksReviewsCard` (`testId` + `testIdPrefix` props).
- Review UI is intentionally read-only. Verdict authority remains in reviewer-bound `submit_run_review`; web operators inspect persisted status/outcome/missing_work/next_round_guidance only. The shared review card surfaces a disclaimer reinforcing that operator sessions cannot bind a verdict.
- Profile editing is a JSON textarea over the typed contract because the profile shape is large and side-table-heavy. Edit/delete buttons are disabled when the task has an active run (matches `task.Service` mutation rejection during active runs). The runtime is still authoritative — toast surfaces server rejections.
- Bridge notification UI exposes cursor diagnostics truthfully, including a dedicated zero-state branch when `last_sequence === 0 && !last_delivery_id && !last_delivered_at`.
- Stream resume UI consumes `latest_event_seq` from the task summary as the SSE seed and derives connection state from `useTaskStream`'s `onEvent`/`onError` callbacks. No invented metrics; `idle` is shown until the first frame arrives, `connected` during normal operation, `error` on EventSource error, `disabled` when the orchestration tab is closed or `taskId` is empty.
- Component complexity rules required extracting the profile editor into `web/src/systems/tasks/hooks/use-profile-editor.ts` and merging the route-level data hooks into `web/src/hooks/routes/use-task-detail-route.ts`.
- Added Playwright spec `web/e2e/tasks-orchestration.spec.ts` exercising the orchestration tab against seeded runtime fixtures, asserting the default execution profile summary, empty states for reviews/notifications, and visible stream resume metrics.

## Learnings

- The `compozy-react(max-component-complexity)` lint and `compozy-react-hooks(no-mixed-hooks-and-components)` lint catch hook/handler bloat in route components and presentational cards. Solving them with separate route hooks (`use-task-detail-route.ts`) and dedicated hook files (`hooks/use-profile-editor.ts`) keeps components pure.
- AGH `EventSource` interactions in jsdom-style hook tests require a stub global with `addEventListener`/`removeEventListener` because `useTaskStream` registers named SSE listeners. Without the stub, `useTaskStream` short-circuits in the SSR/disabled branch and the orchestration hook tests can no longer assert seed-sequence + connection state.
- `oxlint` flags ignored `_error` catch parameters (`no-unused-vars`); use bare `catch` with a comment instead. The handler-surfaced toast is in the route hook, so component-side catches just keep dialogs open for retry.
- The `enableRunReviews` flag on `useTaskRunPage` lets existing run-page tests skip review fetching while keeping the default behaviour live. Existing adapter mocks must include `listTaskRunReviews` to avoid noisy unmocked-call traces.

## Files / Surfaces Touched

- `web/src/systems/tasks/components/tasks-execution-profile-card.tsx` — profile inspection, JSON edit dialog, delete dialog with active-run guard.
- `web/src/systems/tasks/components/tasks-reviews-card.tsx` — review list with status/outcome/reviewer/round/missing_work/guidance, read-only disclaimer; reused in task-level + run-level surfaces via `testId`/`testIdPrefix` props.
- `web/src/systems/tasks/components/tasks-bridge-notifications-card.tsx` — bridge subscription cursor diagnostics with explicit zero-state, create form (bridge_instance_id/scope/delivery_mode/workspace_id/peer_id/group_id/thread_id/subscription_id) and delete confirmation.
- `web/src/systems/tasks/components/tasks-stream-resume-card.tsx` — `latest_event_seq` + SSE resume seed display with connection state pill (`idle|connected|error|disabled`).
- `web/src/systems/tasks/components/tasks-detail-orchestration-panel.tsx` — composite presentational panel rendered when `Orchestration` tab is active.
- `web/src/systems/tasks/hooks/use-profile-editor.ts` — extracted JSON editor state hook to satisfy the hook/component-separation lint.
- `web/src/hooks/routes/use-task-detail-orchestration-tab.ts` — orchestration tab data hook (profile/reviews/subscriptions queries, mutations, `useTaskStream` binding, toast surfaces).
- `web/src/hooks/routes/use-task-detail-route.ts` — route-level wrapper combining `useTaskDetailPage`, `useTaskDetailOrchestrationTab`, and the delete mutation to satisfy max-hook-call lint on the route component.
- `web/src/hooks/routes/use-task-detail-page.ts` — extended `TaskDetailPanel` union with `"orchestration"`.
- `web/src/hooks/routes/use-task-run-page.ts` — added `useTaskRunReviews` query with `enableRunReviews` flag, exposed `reviews`, `reviewsError`, `reviewsLoading`.
- `web/src/routes/_app/tasks.$id.tsx` — refactored to use the combined route hook and added the orchestration tab + panel.
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx` — appended `TasksReviewsCard` (run-level variant) below the existing run detail panels.
- `web/src/systems/tasks/index.ts` — public barrel exports for the new components.
- `web/e2e/fixtures/selectors.ts` + `selectors.test.ts` — added orchestration tab/card selectors with stable test IDs.
- `web/e2e/tasks-orchestration.spec.ts` — new Playwright spec exercising the orchestration tab against seeded runtime fixtures.
- Tests: `tasks-execution-profile-card.test.tsx`, `tasks-reviews-card.test.tsx`, `tasks-bridge-notifications-card.test.tsx`, `tasks-stream-resume-card.test.tsx`, `tasks-detail-orchestration-panel.test.tsx`, `use-task-detail-orchestration-tab.test.tsx`, plus extensions to `use-task-run-page.test.tsx`.

## Errors / Corrections

- First lint pass surfaced six errors: four `_error` catch parameters (oxlint `no-unused-vars`) and two `compozy-react(max-component-complexity)` violations (`TaskDetailRoute` 6 hook calls, `TasksExecutionProfileCard` behavior score 8). Fixed by changing catches to bare `catch {}`, extracting `useProfileEditor` into `hooks/use-profile-editor.ts`, and combining the route hooks into `useTaskDetailRoute`.
- Initial usage of `pillToneFromLegacyTone("default")` was an invalid argument for the legacy tone helper; replaced with the direct `PillTone` enum.
- Hardcoded `--color-warning-tint-border` token did not exist; switched to the standard `--color-divider` border with the warning tint background per `packages/ui/src/tokens.css`.
- Existing `useTaskRunPage` test mocks were missing `listTaskRunReviews`; added the mock so the run reviews query resolves cleanly during the existing happy-path tests.
- 2026-05-05 audit follow-up: codex flagged that `useTaskDetailOrchestrationTab` initialized `streamState` from `enabled && hasTaskId` but never resynced when the operator opened the orchestration tab — the live `useTaskStream` subscription stayed `"disabled"` until the first SSE frame or error. Fixed by adding a `useEffect` keyed on `streamEnabled` that resets `streamState` to `"idle"` (or `"disabled"`) and clears `streamErrorMessage` on every flip; new regression tests cover both transitions in `use-task-detail-orchestration-tab.test.tsx`.
- The new Playwright spec previously asserted `tasks-execution-profile-empty` against a seeded daemon, but `seedBrowserTasksOperatorFlow` lands tasks with the runtime's default execution profile (all `inherit`). Updated the spec and selectors to assert `tasks-execution-profile-summary` instead, matching truthful UI; the empty-state selector remains for unit coverage.

## Verification Evidence

- `bunx vitest run src/hooks/routes/use-task-detail-orchestration-tab.test.tsx` (focused regression): PASS — 1 file / 6 tests, including the new disabled→enabled and enabled→disabled streamState transitions.
- `bunx vitest run src/hooks/routes src/systems/tasks` (focused suite touching the fix): PASS — 61 files / 427 tests.
- `bun run test:e2e:daemon-served:raw -- e2e/tasks-orchestration.spec.ts` (workspace Playwright command, daemon-served lane): PASS — `e2e/tasks-orchestration.spec.ts:36:1 › operator inspects orchestration tab on a real seeded task` (1.3s). Built `.tmp/agh` + `.tmp/acpmock-driver` and exported `AGH_TEST_DAEMON_BIN` / `AGH_TEST_ACPMOCK_DRIVER_BIN` to invoke through the workspace bun script (running `bunx playwright test` directly produces a "two different versions of @playwright/test" duplicate-resolution error).
- `make web-lint`: 0 warnings / 0 errors.
- `make web-typecheck`: PASS.
- `make web-test`: 212 files / 1629 tests PASS.
- `make web-build`: PASS (vite build + tsgo --noEmit).
- `make verify`: PASS — DONE 8283 tests in 62.458s, OK: all package boundaries respected.

## Ready for Next Run

- Tasks 28 (orchestration profile docs) and 29 (review-gate / cursor docs) can reference the new orchestration tab as the user-facing surface for execution profile inspection/edit/delete, run/task reviews, bridge notification diagnostics, and SSE resume metrics. No changes are required to the task 26 data layer; the new components consume only the existing public exports from `@/systems/tasks`.
