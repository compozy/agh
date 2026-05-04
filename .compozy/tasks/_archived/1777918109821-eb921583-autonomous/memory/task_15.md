# Task Memory: task_15.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Honesty pass on Tasks UI: distinguish saved intent from coordinator-handoff execution, label publish/start/approval as the handoff boundary, surface coordination channel availability, and prove the manual-first ADR-010/ADR-012 bookends in Playwright. Completed.

## Important Decisions
- Single source of truth for lifecycle copy in `task-formatters.ts`: `taskLifecyclePhase`, `taskLifecyclePhaseLabel`, `taskLifecyclePhaseDescription`, `taskLifecyclePhaseTone`, `taskHandoffActionKey`, `taskHandoffActionCopy`, `runIsCoordinated`, `runCoordinationChannelLabel`. All UI surfaces consume those helpers instead of inlining strings.
- Operator handoff CTA is now "Start run" (was "Enqueue Run") with a tooltip naming coordinator handoff. Test IDs `tasks-detail-enqueue` / `tasks-detail-preview-enqueue` stay stable.
- Coordination channel shows as a violet `Channel: <display_name>` chip via `pillVariantFromTone("violet")` (the `Pill` component only accepts the canonical variants — never pass a raw tone).
- Adapter shapes were intentionally NOT changed: `publishTask`/`approveTask` keep returning `.task`. The detail/preview surfaces pull the active run + channel from the auto-refreshed `useTask` query, which already includes `summary.active_run.coordination_channel(_id)?`.

## Learnings
- Storybook fixture `buildTaskRunFixture`/`buildTaskRunRecordFixture` already populate `coordination_channel_id` and the embedded `coordination_channel`; new fixtures (saved intent, agent approval, queued coordinated, coordinator-enabled workspace) reuse them and stay aligned with generated `OperationResponse<...>` types.
- The web `Pill` component accepts only `accent | danger | default | info | success | warning`; route any "violet" semantic through `pillVariantFromTone("violet")` so the type stays narrow.
- Playwright `--list` is enough to validate spec syntax without spinning up the daemon-served lane locally.

## Files / Surfaces
- `web/src/systems/tasks/lib/task-formatters.ts` (+ `.test.ts`) — lifecycle/handoff/channel helpers and 14 new tests.
- `web/src/systems/tasks/components/tasks-detail-header.tsx`, `tasks-detail-preview-panel.tsx`, `tasks-detail-overview-panel.tsx`, `tasks-detail-runs-panel.tsx` — lifecycle pill, hint paragraph, coordination chip, saved-intent runs empty state, start-run tooltip.
- Component test updates: `tasks-detail-header.test.tsx`, `tasks-detail-preview-panel.test.tsx`, `tasks-detail-overview-panel.test.tsx`, `tasks-detail-runs-panel.test.tsx`.
- `web/src/systems/tasks/mocks/fixtures.ts` and `mocks/index.ts` — `savedIntentTaskFixture`, `awaitingApprovalTaskFixture`, `queuedCoordinatedTaskFixture`, `coordinatorEnabledWorkspaceFixture`, plus `mocks/fixtures.test.ts` to lock in the contract.
- `web/e2e/tasks-coordinator-handoff.spec.ts` — four ADR-010/ADR-012 cases (create-without-run, publish handoff with channel chip, approval handoff, manual session coexistence).
- `web/e2e/fixtures/selectors.ts` (+ `selectors.test.ts`) — added detail lifecycle/coordination/channel selectors and matching unit assertions.

## Errors / Corrections
- First pass passed `variant="violet"` directly to `Pill`. tsgo caught the union mismatch; switched all four sites to `variant={pillVariantFromTone("violet")}` and rebuilt.

## Ready for Next Run
- Task 16 (runtime docs + CLI references) can cite these test IDs and the violet "Channel: …" chip when documenting the coordinator handoff and ADR-012 channel binding visualization.
