# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Current State
- Task_02 write-model support is in place and verified: task lifecycle reconciliation is manager-owned, and later task reads/APIs should reuse it rather than infer draft, approval, or attempt semantics client-side.
- Task_03 read-model support is now verified: manager-owned `ListTasks`/`GetTask` expose enriched dependency references, counts, active-run summaries, latest activity, and title/identifier search so later API/frontend tasks should reuse these point reads instead of stitching extra task joins client-side.
- Task_04 live-read support is now verified: manager-owned timeline, replayable stream, task-tree, and run-detail reads expose backend-owned task live models so later task API/frontend work should consume these surfaces instead of stitching generic session SSE on the client.
- Task_05 dashboard reads are now verified: `observe.QueryTaskDashboard` is the canonical aggregate for dashboard cards and charts, exposing observer-owned totals, queue backlog, filtered health warnings, active-run summaries, and freshness metadata so later API/frontend tasks should not reconstruct dashboard state from raw summary/metrics/health queries client-side.
- Task_12 frontend shell is live: the `/_app/tasks` route family (base + `$id` + `$id/runs/$runId`) uses TanStack auto-nesting with a shared `TasksPageShell` wrapping `WorkspacePageShell`. Later tasks should render screen content inside this shell rather than forking new route-shell layouts.

## Shared Decisions
- Durable actor-scoped inbox triage state lives in GlobalDB `task_triage_state` and is exposed through the task store `GetTaskTriageState` / `UpsertTaskTriageState` surface.
- Stable task live replay semantics are backed by persisted `task_events` sequence order; reconnect-aware task streaming should reuse `after_sequence` rather than inventing a second cursor model.
- OpenAPI response documentation now supports explicit per-response content types in `internal/api/spec/spec.go`; task-native live streams should be documented as `text/event-stream` in the shared spec/codegen surface instead of falling back to generic session-stream assumptions.
- Tasks frontend entrypoint: `/_app/tasks` is the canonical route and `TasksPageShell` is the canonical shell. Sub-surfaces (list/kanban/dashboard/inbox/detail/run-detail/multi-agent live) must mount inside this shell instead of creating parallel `/_app` areas.

## Shared Learnings
- `internal/task/manager.go` now owns draft publication (`PublishTask`) plus approval-aware and attempt-aware canonical status reconciliation, which downstream task/API/read-model work should treat as authoritative.
- The daemon boot/runtime path only enables task services when the backing registry satisfies the full `task.Store` contract, so store/test doubles must keep pace when new live-read methods are added.

## Open Risks

## Handoffs
- Tasks 07-09 should serialize and transport the task live payloads directly, preserving sequence metadata and optional runtime-summary fields instead of recomputing them in handlers or clients.
- Tasks 07, 13, and 16 should transport and consume the task dashboard payload directly, preserving its freshness and warning fields instead of deriving Paper dashboard cards from lower-level observer buckets.
