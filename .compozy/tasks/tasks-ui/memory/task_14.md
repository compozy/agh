# Task Memory: task_14.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Deliver list, kanban, empty-state, and create-modal surfaces for `/_app/tasks` on top of the task_13 system, keeping orchestration in `useTasksPage` and components presentational.
- Render the right-hand split-view detail as a lightweight preview that links to `/tasks/$id`; full detail/timeline/runs experience belongs to task_15.

## Important Decisions
- Kanban groups `draft+pending` into "Pending" and `failed+canceled` into "Failed" so the Paper layout matches today's task statuses without inventing new domain states.
- Create modal owns six task templates as frontend presets (`one_shot/recurring/epic/remote_peer/human_in_loop/blank`); recurring forces `draft=true` regardless of submit button. Templates layer defaults onto the user-edited payload via `applyTemplateToCreatePayload`.
- Submission flow: `useCreateTask` followed by `useEnqueueTaskRun` only when the template's `enqueueOnSubmit=true` and the user did not click "Save draft". Failures during enqueue surface a partial-success toast but keep the created task selected.
- The split-view detail panel inline-fetches the existing `getTask` payload via `useTask` (already provided by task_13). The deep `/tasks/$id` route remains task_15's surface.
- To satisfy the project's `compozy-react/max-component-complexity` (>=7 handlers), modal handlers are extracted into `use-tasks-create-modal-form.ts` co-located with the modal component.

## Learnings
- TanStack Router Link in unit tests requires a manual mock that strips `to`/`params` to prevent DOM warnings when QueryClientProvider isn't routed. Reused from existing pattern in `-tasks.test.tsx`.
- `vitest run` (`bun run test:raw`) is the supported runner; `bun test` doesn't expose `vi.mocked` and breaks the existing tasks-system test suite.
- `TaskRecord` from generated types includes `description`, `priority`, `max_attempts`, `approval_*`, but the list payload does NOT include `description`; preview falls back to placeholder text when description isn't populated.

## Files / Surfaces
- web/src/hooks/routes/use-tasks-page.ts — extended with create draft state, template id, owner filter, kanban groupings, mutation orchestration (`submitCreateTask`, `handlePublishTask`).
- web/src/routes/_app/tasks.tsx — renders shell + mode pills + create button + (empty | kanban | split-view) + create modal.
- web/src/systems/tasks/components/{task-card,tasks-list-panel,tasks-detail-preview-panel,tasks-kanban-board,tasks-empty-state,tasks-create-modal,use-tasks-create-modal-form}.tsx (and `.test.tsx`).
- web/src/systems/tasks/lib/{task-templates,task-grouping}.ts (+ `.test.ts`); task-formatters.ts gains `taskOwnerKindLabel`, `taskOwnerLabel`, `formatRelativeTime`, `formatAttemptLabel`.
- web/src/systems/tasks/index.ts — re-exports for new components, helpers, and templates so route consumers stay on the barrel.

## Errors / Corrections
- Initial detail panel referenced `summary?.recent_runs`, which doesn't exist; switched to `detail.runs` and the dependency_references array.
- Modal originally had 8 handlers; refactored into `useTasksCreateModalForm` to satisfy `compozy-react/max-component-complexity`.
- Updated `-tasks.test.tsx` to mount inside `QueryClientProvider` and mock the entire tasks-api adapter so the route can call `useTask` etc. without network access.

## Ready for Next Run
- Task_15 owns the deep `/tasks/$id` route. The lightweight preview panel here uses `useTask` directly; task_15 should replace it (or pass enriched timeline/runs sections through props) so the deep route can fully take over.
- Owner select uses a plain `<select>` for now; a future polish pass should swap in `Select` from `components/ui/select` once we have a paginated agent picker.
