# TechSpec: Tasks UI in `web/`

## Executive Summary

This initiative implements the full `docs/design/paper/tasks` surface as a first-class AGH feature in `web/`, exposed through a dedicated `/_app/tasks` area with sidebar navigation, route-level orchestration hooks, and a self-contained `web/src/systems/tasks/` domain system. There is no `_prd.md` for `tasks-ui`; this document uses the Paper task artboards plus the existing screen-by-screen capability analysis under `.compozy/tasks/tasks-ui/analysis/` as the authoritative product input.

The implementation strategy is to split responsibilities cleanly across three backend layers and one frontend system. The task manager remains responsible for task CRUD, draft publication, dependencies, run lifecycle, and task-native point reads. The observer/read side owns dashboard, inbox, and aggregate projections. A dedicated task live layer exposes timeline, task stream, tree live view, and run detail contracts so the web app does not stitch task detail with session SSE as its primary architecture. The primary trade-off is a larger first-wave backend scope in exchange for a stable frontend architecture, cleaner contracts, and real parity with the Paper screens.

## Design References

All task screens live in the `AGH` Paper file (page `Page 1`). PNG exports are committed under `docs/design/paper/tasks/` and kept in sync with the Paper artboards listed below.

| Screen | Local export | Paper artboard (node id) |
|--------|--------------|--------------------------|
| Dashboard | `docs/design/paper/tasks/AGH Tasks — Dashboard@2x.png` | `AGH Tasks — Dashboard` (`TXD-0`) |
| Inbox | `docs/design/paper/tasks/AGH Tasks — Inbox@2x.png` | `AGH Tasks — Inbox` (`U5Y-0`) |
| List (Split View) | `docs/design/paper/tasks/AGH Tasks — List (Split View)@2x.png` | `AGH Tasks — List (Split View)` (`SFL-0`) |
| Kanban View | `docs/design/paper/tasks/AGH Tasks — Kanban View@2x.png` | `AGH Tasks — Kanban View` (`SS7-0`) |
| Empty State | `docs/design/paper/tasks/AGH Tasks — Empty State@2x.png` | `AGH Tasks — Empty State` (`T1V-0`) |
| Create Modal | `docs/design/paper/tasks/AGH Tasks — Create Modal@2x.png` | `AGH Tasks — Create Modal` (`T7W-0`) |
| Detail (Events SSE) | `docs/design/paper/tasks/AGH Tasks — Detail (Events SSE)@2x.png` | `AGH Tasks — Detail (Events SSE)` (`TDL-0`) |
| Run Detail | `docs/design/paper/tasks/AGH Tasks — Run Detail@2x.png` | `AGH Tasks — Run Detail` (`TK9-0`) |
| Multi-Agent Live | `docs/design/paper/tasks/AGH Tasks — Multi-Agent Live@2x.png` | `AGH Tasks — Multi-Agent Live` (`TR5-0`) |

## System Architecture

### Component Overview

The implementation consists of these main components:

- `web/src/routes/_app/tasks.tsx`: top-level tasks workspace area, header, view switching, empty states, and route composition.
- `web/src/routes/_app/tasks.$id.tsx`: task detail deep link for split view and live timeline scenarios.
- `web/src/routes/_app/tasks.$id.runs.$runId.tsx`: run detail deep link for transcript, metrics, and run control.
- `web/src/hooks/routes/use-tasks-page.ts`: page orchestration for list, kanban, dashboard, inbox, create modal, search params, and selected task state.
- `web/src/hooks/routes/use-task-detail-page.ts`: orchestration for task detail, timeline, and live task stream state.
- `web/src/hooks/routes/use-task-run-page.ts`: orchestration for run detail and task-run live updates.
- `web/src/systems/tasks/`: domain system containing adapters, query options, query keys, hooks, components, formatters, and types.
- `internal/task/*`: authoritative task write model, validation, storage integration, detail reads, draft publication, dependencies, and run lifecycle commands.
- `internal/observe/tasks.go` plus new observer-backed task view handlers: dashboard, inbox, and aggregate read models.
- `internal/api/core/*`, `internal/api/httpapi/routes.go`, `internal/api/udsapi/routes.go`, and `internal/api/spec/spec.go`: shared handler, transport wiring, and OpenAPI documentation for new task contracts.
- `internal/extension/host_api_tasks.go`: Host API parity for new task-native point reads and aggregate read models where extension consumers need them.

Data flow is intentionally split:

- Task-native screens such as create, list, kanban, detail, publish, dependency editing, and run commands call task manager-backed endpoints.
- Aggregate operator screens such as dashboard and inbox call observer-backed endpoints shaped for UI read models.
- Live task screens subscribe to task-native SSE and timeline endpoints, with session links used for transcript drill-down rather than as the primary join mechanism.
- Frontend components remain presentational. Route hooks compose server state, local UI state, and URL state, then pass flat props to system components.

## Implementation Design

### Core Interfaces

The backend needs a task-native live/read surface that other API handlers can depend on:

```go
type TaskLiveService interface {
	Timeline(ctx context.Context, taskID string, query taskpkg.EventQuery, actor taskpkg.ActorContext) ([]TaskTimelineItem, error)
	Stream(ctx context.Context, taskID string, actor taskpkg.ActorContext) (<-chan TaskStreamEvent, error)
	Tree(ctx context.Context, taskID string, actor taskpkg.ActorContext) (*TaskTreeLiveView, error)
	RunDetail(ctx context.Context, runID string, actor taskpkg.ActorContext) (*TaskRunDetailView, error)
}
```

The enriched list contract should be explicit instead of forcing the frontend to derive operational badges from multiple payloads:

```go
type TaskListItem struct {
	Task            contract.TaskSummaryPayload `json:"task"`
	Priority        taskpkg.Priority            `json:"priority"`
	MaxAttempts     int                         `json:"max_attempts"`
	Draft           bool                        `json:"draft"`
	ChildCount      int                         `json:"child_count"`
	DependencyCount int                         `json:"dependency_count"`
	ActiveRun       *TaskRunChip                `json:"active_run,omitempty"`
	LastActivityAt  time.Time                   `json:"last_activity_at,omitempty"`
}
```

Frontend code depends on generated OpenAPI types plus a dedicated tasks system barrel:

- `web/src/systems/tasks/types.ts`: exports request, response, read-model, and view-model types derived from OpenAPI operations.
- `web/src/systems/tasks/adapters/tasks-api.ts`: the only layer allowed to call `apiClient`.
- `web/src/systems/tasks/lib/query-keys.ts` and `query-options.ts`: co-located query keys and fetchers for list, detail, timeline, tree, run detail, dashboard, and inbox.
- `web/src/systems/tasks/hooks/use-tasks.ts`, `use-task-actions.ts`, `use-task-live.ts`, `use-task-dashboard.ts`, and `use-task-inbox.ts`: reusable query and mutation hooks.
- `web/src/systems/tasks/components/*`: split-view panels, dashboard cards, inbox list, create modal, kanban board, timeline, and run detail UI.

### Data Models

Core task domain changes:

- Add `TaskStatusDraft` as a first-class non-runnable status.
- Add `priority` as a first-class task field with a constrained enum, not metadata.
- Add `max_attempts` as a first-class task field that governs enqueue/retry policy.
- Add optional `approval_policy` and `approval_state` to support the approval card/template and inbox approvals lane.
- Keep `metadata` for extensibility, but stop using it as the primary home for Paper-critical semantics.

Core task read-model additions:

- `TaskListItem`: enriched list/kanban summary with counts, blockers, active run chip, latest failure summary, and last activity.
- `TaskTimelineItem`: normalized timeline row joining task-domain events with task-native live state and optional linked session/run context.
- `TaskRunDetailView`: run detail payload with run record, task reference, linked session reference, timestamps, tool-call counts, token usage when available, current activity, and live execution status.
- `TaskTreeLiveView`: parent task plus descendants, each with status, owner, active run, linked session, and latest activity for the multi-agent live screen.
- `TaskDashboardView`: cards, totals, health, queue depth, and recent active runs shaped for the Paper dashboard.
- `TaskInboxItem`: triage-oriented item with lane, unread/read state, archived state, approval state, blocking reason, latest activity, and linked task/run references.

Persistent storage changes:

- Extend the durable task record to persist `priority`, `max_attempts`, `approval_policy`, and `approval_state`.
- Add a persistent `task_triage_state` table keyed by `(task_id, actor_ref)` for read/unread, archived, dismissed, and last_seen activity.
- Reuse existing task event storage for timeline/history; do not create a parallel timeline store.
- Reuse observer projections for summary and metrics, but add explicit inbox/dashboard projection queries and API conversion layers.

### API Endpoints

Task manager-backed endpoints:

- `GET /api/tasks`: returns enriched `TaskListItem[]` for split view and kanban. Add filters `query`, `priority`, `include_drafts`, `approval_state`, `scope`, `workspace`, `status`, `owner_kind`, `owner_ref`, `parent_task_id`, `network_channel`, and `limit`.
- `POST /api/tasks`: creates a task or draft. Accepts `title`, `description`, `scope`, `workspace`, `owner`, `priority`, `max_attempts`, `draft`, `network_channel`, `approval_policy`, and `metadata`.
- `GET /api/tasks/{id}`: returns enriched detail view with task data, children, dependencies, run summaries, and task metadata needed for the split view.
- `PATCH /api/tasks/{id}`: updates mutable task fields including title, description, owner, network channel, priority, max attempts, approval policy, and metadata.
- `POST /api/tasks/{id}/publish`: transitions a draft task into runnable state, with the manager computing the resulting status as `ready` or `blocked`.
- `POST /api/tasks/{id}/cancel`: cancels the task tree with existing lifecycle behavior.
- `POST /api/tasks/{id}/children`: creates a child task with the same first-class fields.
- `POST /api/tasks/{id}/dependencies` and `DELETE /api/tasks/{id}/dependencies/{depends_on_id}`: unchanged commands, but detail/list responses must now carry richer blocker references.
- `GET /api/tasks/{id}/runs`: returns run summaries for the task.
- `GET /api/task-runs/{id}`: new point-read endpoint returning `TaskRunDetailView`.
- Existing task-run lifecycle endpoints remain and are reused by the UI for claim, start, attach, complete, fail, and cancel flows.

Task live endpoints:

- `GET /api/tasks/{id}/timeline`: paginated or limited task timeline endpoint returning `TaskTimelineItem[]`.
- `GET /api/tasks/{id}/stream`: task-native SSE endpoint for timeline and live status updates.
- `GET /api/tasks/{id}/tree`: task tree live-view endpoint returning `TaskTreeLiveView`.
- `POST /api/tasks/{id}/approve` and `POST /api/tasks/{id}/reject`: new approval commands for approval-backed inbox lanes.
- `POST /api/tasks/{id}/triage/read`, `POST /api/tasks/{id}/triage/archive`, and `POST /api/tasks/{id}/triage/dismiss`: new triage mutations backing inbox actions.

Observer-backed read-model endpoints:

- `GET /api/observe/tasks/dashboard`: dashboard payload combining summary, metrics, health, queue, and active-run cards for the Paper dashboard.
- `GET /api/observe/tasks/inbox`: inbox payload grouped by lane such as `my_work`, `approvals`, `failed_runs`, `blocked`, and `archived`, with filters for workspace, owner, lane, unread, and query.
- `GET /api/observe/tasks/metrics`: optional lower-level metrics endpoint for future reuse if dashboard composition needs to be split later. If introduced, it remains an implementation detail of the read side rather than a UI-mandated surface.

Response and status conventions:

- Use existing `200/201/404/409/422/503` semantics already established by task endpoints.
- Document `text/event-stream` explicitly for `/api/tasks/{id}/stream`.
- Mirror new HTTP endpoints in UDS and OpenAPI so generated frontend types stay authoritative.

## Integration Points

Not applicable for external services. This initiative stays within the existing AGH daemon, observer, storage, OpenAPI, and `web/` codebase boundaries.

## Impact Analysis

| Component | Impact Type | Description and Risk | Required Action |
|-----------|-------------|---------------------|-----------------|
| `web/src/components/app-sidebar.tsx` | modified | Adds a first-class Tasks navigation entry. Low risk. | Add sidebar item and active-state handling. |
| `web/src/routes/_app/tasks*.tsx` | new | New top-level operator area and detail deep links. Medium risk due to route complexity. | Add route files and tests. |
| `web/src/hooks/routes/use-tasks-page.ts` and detail hooks | new | Central orchestration for tasks UI state, filters, and live subscriptions. Medium risk. | Add route hooks and route-level tests. |
| `web/src/systems/tasks/` | new | New frontend domain system for all task UI surfaces. Medium risk. | Add adapters, query keys/options, hooks, components, and tests. |
| `web/src/generated/agh-openapi.d.ts` | modified | New task and observe endpoints expand generated types. Low risk if spec stays authoritative. | Regenerate types after OpenAPI changes. |
| `internal/task/*` | modified | Domain expansion for draft, priority, max attempts, approval, enriched reads, and run detail. High risk because it changes core lifecycle semantics. | Extend domain types, manager logic, validation, and tests. |
| `internal/store/globaldb/*` | modified | Persist new task fields and triage state. Medium risk. | Add storage columns/tables and query helpers. |
| `internal/observe/tasks.go` | modified | New dashboard and inbox projections. Medium risk. | Add read-model queries and tests. |
| `internal/api/core/*` | modified | Shared handler layer for new task point reads, live APIs, and observe task views. Medium risk. | Add interfaces, handlers, parsers, and converters. |
| `internal/api/httpapi/routes.go` and `internal/api/udsapi/routes.go` | modified | Transport parity for new endpoints. Medium risk because route drift is possible. | Wire every new endpoint in both registries. |
| `internal/api/spec/spec.go` and OpenAPI generation | modified | New contracts required for frontend codegen. Medium risk. | Document every new payload and stream endpoint. |
| `internal/extension/host_api_tasks.go` | modified | Host API should stay aligned with new task-native reads and commands. Low to medium risk. | Extend Host API task methods where extension consumers benefit. |

## Testing Approach

### Unit Tests

Backend unit coverage must include:

- task validation for `priority`, `draft`, `max_attempts`, and approval semantics
- status transitions for draft creation, draft publication, approval gating, and retry exhaustion
- enriched list/detail/run/timeline converters
- observer dashboard and inbox projection logic
- SSE event normalization and cursor handling for task-native streams

Frontend unit coverage must include:

- tasks adapter error handling and request shaping
- query-key and query-options coverage for every new endpoint
- route hook state machines for list, kanban, dashboard, inbox, create modal, detail, and run detail flows
- formatter and filtering helpers for badges, lanes, counts, and timeline presentation
- presentational component states for loading, empty, error, and populated surfaces

### Integration Tests

Backend integration coverage must include:

- HTTP and UDS parity for all new task point-read and observe read-model endpoints
- persistence of new task fields and triage state
- draft publication behavior against real task reconciliation rules
- run detail and timeline queries against real stored task events and runs
- task stream behavior for live update, reconnect, and task-tree changes

Frontend integration coverage must include:

- route rendering with mocked hooks for `/_app/tasks`, task detail, and run detail
- query/mutation invalidation for create, publish, retry, approve, archive, and dismiss flows
- generated contract alignment tests for new task and observe payloads

Browser/E2E coverage must include at least one real browser flow for `/_app/tasks`:

- open Tasks from the sidebar
- create a draft task
- publish it into runnable state
- inspect it in split view
- open run detail
- observe a live update or fallback state
- validate dashboard or inbox navigation at least once in the same route family

Required verification gates before completion:

- `make verify`
- `make web-lint`
- `make web-typecheck`
- relevant backend and web test targets for the changed surfaces

## Development Sequencing

### Build Order

1. Backend domain expansion for task fields and statuses. No dependencies.
2. Storage and manager updates for `priority`, `draft`, `max_attempts`, approval, and triage state. Depends on step 1.
3. New task point-read APIs for enriched list/detail, draft publication, run detail, timeline, tree, and task SSE. Depends on step 2.
4. Observer-backed dashboard and inbox read models plus transport wiring. Depends on step 2.
5. OpenAPI and generated frontend types for every new endpoint and payload. Depends on steps 3 and 4.
6. Frontend `web/src/systems/tasks/` adapters, query layer, and types. Depends on step 5.
7. Frontend routes, route hooks, sidebar wiring, and presentational components for list, kanban, create, detail, run detail, dashboard, inbox, and multi-agent live. Depends on step 6.
8. Backend integration tests and frontend integration tests. Depends on steps 3, 4, 6, and 7.
9. Browser/E2E tasks flow for `/_app/tasks`. Depends on step 7 and at least one stable live backend path from steps 3 and 4.
10. Final verification and polish. Depends on steps 8 and 9.

### Technical Dependencies

- OpenAPI generation must include every new task and observe contract before frontend adapter work can stabilize.
- HTTP and UDS route parity is a blocking dependency for contract correctness.
- Sidebar and route-tree changes depend on the new tasks route existing and being testable.
- Task SSE requires a clearly defined event payload and reconnection contract before frontend live hooks are finalized.

## Monitoring and Observability

Operational visibility must cover both command-side and read-side behavior:

- Track task stream connection count, reconnect count, disconnect reason, and stream lag.
- Track dashboard and inbox query latency plus payload size.
- Track task publish, approve, reject, archive, dismiss, and retry command counts.
- Track queue depth, stuck runs, approval backlog, unread inbox counts, and triage throughput.
- Log task live events with `task_id`, `run_id`, `workspace_id`, `owner_kind`, `owner_ref`, and `origin`.
- Log observer read-model failures separately from task manager command failures so dashboard/inbox regressions are diagnosable.
- Surface warning conditions when task stream falls back to polling or when observer projections are stale beyond the accepted freshness threshold.

## Technical Considerations

### Key Decisions

- Decision: tasks become a first-class feature area at `/_app/tasks`.
  Rationale: the Paper surface is a core operator area, not an auxiliary dialog.
  Trade-offs: more route/sidebar integration work up front.
  Alternatives rejected: isolated or hidden rollout.

- Decision: `priority`, `draft`, and `max_attempts` become first-class task semantics.
  Rationale: the UI depends on them structurally, not cosmetically.
  Trade-offs: broader manager and storage changes.
  Alternatives rejected: metadata-driven or UI-only conventions.

- Decision: live task screens use dedicated task APIs, timeline endpoints, and SSE.
  Rationale: avoids client-side joins between task detail and session streams.
  Trade-offs: new contracts and transport work.
  Alternatives rejected: browser-side stitching and generic observe stream reuse.

- Decision: dashboard and inbox live on observer-backed read models.
  Rationale: these are aggregate operator views, not point reads.
  Trade-offs: more task-related APIs across two backend layers.
  Alternatives rejected: task-manager-only aggregates and client-derived dashboards.

- Decision: verification includes backend integration, frontend integration, and browser/E2E coverage.
  Rationale: this feature spans contracts, projections, live streams, and the main operator UI.
  Trade-offs: slower implementation and test effort.
  Alternatives rejected: unit-only or no-browser coverage.

### Known Risks

- Draft and approval semantics can complicate the current task lifecycle.
  Mitigation: define explicit transition rules and test publication/approval matrices thoroughly.

- Dual backend sources, task manager plus observer, can drift semantically.
  Mitigation: keep task manager authoritative for point reads and observer authoritative for aggregates; document contract boundaries clearly.

- Task SSE can fail or lag under rapid run updates.
  Mitigation: define stable event IDs, reconnect behavior, and visible fallback-to-polling states in the frontend.

- The tasks route can become overloaded if every screen becomes its own ad hoc state machine.
  Mitigation: keep route hooks focused, keep components presentational, and split dashboard/inbox/detail/live concerns inside `web/src/systems/tasks/`.

- Inbox triage semantics can overreach the current notion of operator identity.
  Mitigation: scope triage state to the existing actor context model and keep the first implementation explicitly local-operator oriented.

## Architecture Decision Records

- [ADR-001: First-Class Tasks Area in the Main App Shell](adrs/adr-001.md) — Tasks ship as a primary `/_app` feature with sidebar navigation and a dedicated frontend domain system.
- [ADR-002: Expand the Task Domain for Paper-Parity Semantics](adrs/adr-002.md) — `priority`, `draft`, and `max_attempts` become first-class task capabilities instead of metadata or UI conventions.
- [ADR-003: Add Dedicated Task Live Surfaces Instead of Client-Side Stitching](adrs/adr-003.md) — Task detail, run detail, and multi-agent live views depend on task-native live APIs and SSE.
- [ADR-004: Use Observer-Backed Read Models for Dashboard, Inbox, and Aggregate Task Views](adrs/adr-004.md) — Dashboard and inbox are implemented as dedicated observer-backed read models rather than overloaded task-manager payloads.
