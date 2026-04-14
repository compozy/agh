# Multica Task Analysis

## Overview
Multica does not model “tasks” as a separate abstract work graph. The primary domain object is the `issue`, and execution is represented by a persisted `agent_task_queue` row that gets created when an issue is assigned to an agent or when a comment/mention trigger fires. That makes Multica closer to an issue-centric execution system than a generic workflow engine.

The system distinguishes between the work item and its execution history:
- `issue` holds the durable human-facing work state.
- `agent_task_queue` holds each execution attempt and its lifecycle.
- `task_message` and `task_usage` hold the detailed run transcript and cost telemetry.

## Task/Issue Mechanisms
Multica’s explicit task model is the queue table plus lifecycle transitions:
- `queued` -> `dispatched` -> `running` -> `completed` or `failed`, with `cancelled` as an interrupt path.
- Claiming is runtime-aware and atomic, with per-agent concurrency limits and `SKIP LOCKED` queue selection.
- Duplicate work is actively coalesced: the app prevents multiple pending tasks for the same issue, and also blocks duplicate pending work for the same `(issue, agent)` pair.

The implicit workflow layer is issue-driven:
- Assigning an issue to an agent enqueues work.
- Commenting on an issue can enqueue follow-up work for the assigned agent.
- Mentioning an agent enqueues work for that specific agent.
- Issue status changes are intentionally not coupled to task completion; the agent is expected to manage issue status through the CLI during execution.

Hierarchy exists, but it is issue hierarchy, not task hierarchy:
- `parent_issue_id` defines sub-issue structure.
- `ChildIssueProgress` aggregates completion across child issues.
- There is no separate “subtask execution tree”; execution still attaches to an issue and its queue entries.

## Relevant Code Paths
Server-side orchestration and lifecycle live in a small set of high-traffic paths:
- `server/internal/service/task.go` owns enqueue, claim, start, complete, fail, cancel, progress, session reuse, and event broadcasting.
- `server/internal/handler/issue.go` decides when to enqueue or cancel tasks on assignment changes and comment triggers.
- `server/internal/handler/daemon.go` exposes the daemon-facing claim/start/progress/complete/fail/message/status routes.
- `server/internal/daemon/daemon.go` polls runtimes, claims tasks, spawns agent CLIs, streams messages, reports completion, and reuses prior workdirs/sessions.

The persistence model is defined in migrations and SQL:
- `server/migrations/001_init.up.sql` creates `agent_task_queue` with status, priority, timestamps, result, and error.
- `server/migrations/003_task_context.up.sql` adds task context and a pending-task index.
- `server/migrations/020_task_session.up.sql` adds `session_id` and `work_dir` for resumable execution.
- `server/migrations/022_task_lifecycle_guards.up.sql` and `server/migrations/037_fix_pending_task_unique_index.up.sql` enforce queue coalescing and pending-task safety.
- `server/migrations/026_task_messages.up.sql` persists run transcripts.
- `server/pkg/db/queries/agent.sql`, `chat.sql`, `task_message.sql`, and `task_usage.sql` define the queue, transcript, and usage queries.

Operator surfaces are deliberately broad:
- CLI: `server/cmd/multica/cmd_issue.go` exposes `issue runs` and `issue run-messages`, plus create/update/assign/status/comment flows.
- CLI: `server/cmd/multica/cmd_daemon.go` manages daemon lifecycle and runtime registration.
- UI: `apps/web/app/(dashboard)/issues/page.tsx`, `apps/web/app/(dashboard)/issues/[id]/page.tsx`, `apps/web/app/(dashboard)/my-issues/page.tsx`, `apps/web/app/(dashboard)/agents/page.tsx`, and `apps/web/app/(dashboard)/runtimes/page.tsx`.
- UI detail views: `packages/views/issues/components/agent-live-card.tsx` and `packages/views/agents/components/tabs/tasks-tab.tsx`.

## Transferable Patterns
Patterns worth borrowing for AGH core tasks/subtasks:
- Keep the work record and execution record separate. A task should be a durable execution instance, not the same row as the logical work item.
- Use explicit lifecycle transitions with timestamps, not ad hoc booleans.
- Make queue claiming atomic and runtime-aware, with concurrency caps and lock-based deduplication.
- Persist transcripts/messages separately from the task row so live updates, reconnect, and audit are all first-class.
- Keep a resumable execution envelope (`session_id`, `work_dir`, prior-run linkage) when the runtime supports it.
- Expose the same state through REST, CLI, and UI so operators can inspect from whichever surface is convenient.

## Risks/Mismatches
Several Multica choices do not map cleanly to AGH’s likely core-task design:
- Multica is issue-first; AGH core tasks may need to be first-class even when there is no issue object.
- Multica leans on issue assignment as the trigger boundary; AGH may need a more explicit task/subtask orchestration model than “assign to agent and run.”
- Issue status is intentionally managed by the agent during execution, which could conflict with a stricter AGH workflow engine where state transitions are centrally owned.
- The queue is intentionally coalescing and single-issue oriented; AGH subtasks may need fan-out, dependency edges, or fan-in semantics that Multica does not model.
- `agent_task_queue` is doing a lot of jobs at once: queue state, history, telemetry, resume metadata, and chat support. That is practical, but it is a weaker fit for a clean AGH core abstraction.
- `server/internal/service/task.go` and `server/internal/handler/daemon.go` are both god-files. The pattern works, but it is not a good structural template for AGH if the goal is extensible core-task composition.

## Open Questions
- Does AGH need a distinct subtask entity, or is a task run plus dependency metadata enough?
- Should AGH tasks be issue-bound at all, or only optionally linked to issues/workspaces?
- Should task creation be trigger-based like Multica, or explicitly scheduled/orchestrated by a higher-level controller?
- Should AGH persist execution context snapshots, or fetch live context at execution time like Multica does?
- If AGH supports agent-driven state updates, should those be allowed to change task status directly, or only emit events that a controller reconciles?

## Evidence
Vault evidence:
- `/Users/pedronauck/dev/knowledge/multica/wiki/codebase/concepts/Codebase Overview.md`
- `/Users/pedronauck/dev/knowledge/multica/wiki/codebase/concepts/Module Health.md`
- `/Users/pedronauck/dev/knowledge/multica/raw/codebase/files/server/internal/service/task.go.md`
- `/Users/pedronauck/dev/knowledge/multica/raw/codebase/files/server/internal/handler/issue.go.md`
- `/Users/pedronauck/dev/knowledge/multica/raw/codebase/files/server/internal/handler/daemon.go.md`

Repo evidence:
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/internal/service/task.go`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/internal/handler/issue.go`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/internal/handler/daemon.go`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/internal/daemon/daemon.go`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/pkg/db/queries/agent.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/pkg/db/queries/chat.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/pkg/db/queries/task_message.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/pkg/db/queries/task_usage.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/001_init.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/003_task_context.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/020_task_session.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/022_task_lifecycle_guards.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/026_task_messages.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/033_chat.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/migrations/037_fix_pending_task_unique_index.up.sql`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/cmd/multica/cmd_issue.go`
- `/Users/pedronauck/dev/knowledge/.resources/multica/server/cmd/multica/cmd_daemon.go`
- `/Users/pedronauck/dev/knowledge/.resources/multica/CLI_AND_DAEMON.md`
- `/Users/pedronauck/dev/knowledge/.resources/multica/SELF_HOSTING.md`
- `/Users/pedronauck/dev/knowledge/.resources/multica/apps/web/app/(dashboard)/issues/page.tsx`
- `/Users/pedronauck/dev/knowledge/.resources/multica/apps/web/app/(dashboard)/issues/[id]/page.tsx`
- `/Users/pedronauck/dev/knowledge/.resources/multica/apps/web/app/(dashboard)/my-issues/page.tsx`
- `/Users/pedronauck/dev/knowledge/.resources/multica/apps/web/app/(dashboard)/agents/page.tsx`
- `/Users/pedronauck/dev/knowledge/.resources/multica/apps/web/app/(dashboard)/runtimes/page.tsx`
- `/Users/pedronauck/dev/knowledge/.resources/multica/packages/views/issues/components/agent-live-card.tsx`
- `/Users/pedronauck/dev/knowledge/.resources/multica/packages/views/agents/components/tabs/tasks-tab.tsx`
