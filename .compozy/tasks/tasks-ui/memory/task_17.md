# Task Memory: task_17.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Ship the multi-agent live experience on top of the task-tree live read.
- Parent + descendant agent cards, active-run + session drill-downs, interleaved timeline, and stable fallbacks (loading, disconnected, no-descendants, no-active) — all inside the existing `/tasks/$id` route family via a new `"agents"` panel.

## Important Decisions
- Added a new `"agents"` `TaskDetailPanel` rather than a new route; this keeps tabs stable, avoids parallel live-route plumbing, and lets the live tab live next to Overview/Runs/Timeline/Children/Dependencies with a live pill and descendant count.
- Multi-agent state is derived inside `useTaskDetailPage` (`deriveMultiAgentView`) directly from the tree query — no per-descendant detail or run fetches, and no SSE stitching. State machine: `loading → disconnected → no-descendants → no-active → ready` (the first matching state wins).
- Root node is marked `isPrimary` + "Primary · Pinned"; descendants indent by `depth × 16px` and carry hierarchy cues (`data-depth`, `data-is-root`, `parent_task_id` lineage) so layout stays stable as tree data changes.
- Live pill (count) derives from tree `active_run.status ∈ {queued, claimed, starting, running}` plus the detail-level `isLive` for the root. This keeps a single source of truth for "live" across root and descendants.
- Timeline section reuses `TasksTimelinePanel` verbatim. Route wires the same timeline read already loaded for the Events tab so switching into Agents does not trigger a separate fetch.
- Session drill-down stays on `TaskRunDetailSessionLink`'s `/session/$id` contract — the multi-agent card exposes an inline "Open session" / "Open run" / "Open task" affordance per descendant when an active run exists.

## Learnings
- `vi.mock("@tanstack/react-router")`'s Link stub strips `to` before rendering, so tests must assert on `data-testid`/textContent, not `href`.
- `oxfmt` will collapse multi-line arrays and reformat templates after first run; it also flagged a dead `identifier` local on my first draft (caught by tsgo, not oxlint).
- `TaskTreeView.root.task.owner` matches the same `owner` shape already used by `taskOwnerLabel`, but its `kind` enum (`pool`/`automation`/…) differs from task-detail's kind enum; the `agentLabel` helper falls back to `owner.ref → owner.kind → identifier → id`.

## Files / Surfaces
- `web/src/hooks/routes/use-task-detail-page.ts` — added `"agents"` panel type, `MultiAgentView`/`MultiAgentAgent`/`MultiAgentLiveState` types, and `deriveMultiAgentView` helper.
- `web/src/systems/tasks/components/tasks-multi-agent-panel.tsx` (new) — panel + agent card + live pill + avatar + interleaved timeline wrapper.
- `web/src/systems/tasks/components/tasks-multi-agent-panel.test.tsx` (new) — 8 tests covering loading/disconnected/no-descendants/no-active/ready/live-links/failure-summary.
- `web/src/systems/tasks/index.ts` — barrel re-export for `TasksMultiAgentPanel`.
- `web/src/routes/_app/tasks.$id.tsx` — added "Agents" tab with count+live, mounted `TasksMultiAgentPanel` when active.
- `web/src/routes/_app/-tasks.$id.test.tsx` — added tree-with-descendant fixture plus agents-tab + disconnected-fallback integration tests.
- `web/src/hooks/routes/use-task-detail-page.test.tsx` — added 5 tests for multi-agent derivation, loading/disconnected/no-descendants states, and panel switching.

## Errors / Corrections
- Initial card draft held `const identifier` even though it wasn't rendered anywhere; tsgo caught the dead local and I removed it.
- First test asserted `sessionLink.toHaveAttribute("href")` against the Link stub → removed since the stub drops `to`/`params` before rendering.

## Ready for Next Run
- Task_19 QA: golden path is `/_app/tasks → pick a task with children → Agents tab`. Expected surfaces: `tasks-multi-agent-live-count` pill, `tasks-multi-agent-agent-<id>` cards per root + descendants, and `tasks-multi-agent-timeline-live` when any agent is running. Fallbacks: `tasks-multi-agent-loading`, `tasks-multi-agent-disconnected`, `tasks-multi-agent-empty`, `tasks-multi-agent-no-active`.
- If a future task wants per-task SSE on the Agents tab, extend `use-task-live.ts` with a `useTaskStream` hook and swap it into `useTaskDetailPage` — the derivation helper already only depends on the final `tree` shape, so the SSE work would not change the component contract.
