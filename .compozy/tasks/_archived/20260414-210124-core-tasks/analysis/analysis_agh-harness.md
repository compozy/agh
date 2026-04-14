# AGH Harness Analysis: Core Tasks/Subtasks Insertion Points

## Overview

AGH is session-centric today. `internal/session/` owns live runtime orchestration, `internal/store/` splits durable state into a global index plus per-session event DBs, `internal/observe/` is read/health oriented, and `internal/daemon/` is the only composition root. There is no task primitive yet, so a future core tasks/subtasks feature should be introduced as its own daemon-owned service and store, then surfaced through the same transport patterns that already exist for sessions, automation, bridges, and network.

The key constraint is architectural: tasks are not another session concern, and they are not just a new event type. If tasks are meant to span sessions, own dependencies, and expose durable ownership, they need a first-class boundary instead of being bolted onto prompt text, session metadata, or the observer layer.

## Current Harness Boundaries

`internal/session/manager.go:56-80` shows the manager owns active sessions, pending reservations, finalization, driver wiring, notifier fan-out, hooks, skills, MCP, workspace resolution, and prompt assembly. `internal/session/manager_lifecycle.go:18-259` and `internal/session/manager_prompt.go:15-269` reinforce that the runtime model is one process per session with turn-scoped prompting and explicit stop/finalize flows.

`internal/store/types.go:139-507`, `internal/store/globaldb/global_db.go:16-220`, `internal/store/globaldb/global_db_session.go:12-247`, and `internal/store/sessiondb/session_db.go:23-238` show the persistence split clearly: a global sessions table plus summary tables, and a separate append-only per-session SQLite database for events, token usage, and hook runs. That is a good fit for session lifecycle and observability, but it is not a task graph store.

`internal/api/core/interfaces.go:25-127`, `internal/api/httpapi/routes.go:55-136`, `internal/api/udsapi/routes.go:33-136`, `internal/cli/root.go:60-94`, and `internal/cli/session.go:16-230` show the transport story: API and CLI are already thin over daemon-owned services. They expose sessions, observe, automation, bridges, skills, memory, and network, but nothing task-shaped.

`internal/observe/observer.go:26-247`, `internal/observe/health.go:15-154`, `internal/observe/query.go:14-104`, and `internal/observe/reconcile.go:16-107` show the observer is a projection and repair surface. It consumes the registry and live session source, but it is not the authority for runtime mutation.

`internal/daemon/daemon.go:45-238`, `internal/daemon/boot.go:107-250`, `internal/daemon/boundary.go:18-115`, and `internal/daemon/composed_assembler.go:13-113` define the composition root rule set. Any new task service needs to be wired there and nowhere else.

## Existing Task-Like Concepts

The repo already has several task-adjacent patterns, but they are all narrower than a real task system. Automation jobs/triggers are scheduled or event-driven execution surfaces, not user-owned work items. Hook runs are audit records of execution, not a dependency graph. Bridges and network messages carry routed work, but they are transport mechanics, not durable task state.

The strongest prior art is in the archived task/delegation docs. `docs/ideas/from-claude-code/analysis_multi_agent.md:223-314` describes a structured task system with `create/update/list/get`, ownership, dependencies, and worker results delivered as task notifications. `docs/ideas/from-claude-code/filtered_recommendations.md:98-113` repeats the same shape and explicitly proposes a `tasks` table plus `agh task create/update/list/get`. `docs/ideas/orchestration/multi-agent-patterns-analysis.md:301-382` adds the related notion of workflow correlation and append-only handoff state.

The archived `.compozy` task artifacts also matter because they show how the harness has already been decomposed for other features. `_tasks.md` files under `.compozy/tasks/_archived/20260410-144004-session-resilience/`, `.compozy/tasks/_archived/20260412-040024-channels/`, and `.compozy/tasks/_archived/20260406-230650-skills-system/` all use explicit subtasks, dependency ordering, and daemon composition constraints. That is the right delivery shape for a future tasks feature.

## Integration Seams

The cleanest insertion point is a new service interface alongside `SessionManager` and `Observer` in `internal/api/core/interfaces.go`. That keeps the transport layer thin and makes the new capability pluggable through both HTTP and UDS the same way sessions, automation, and bridges already are.

The authoritative data model should live in `internal/store`, likely with a dedicated global-db table set rather than inside `sessiondb`. Tasks need durable ownership, status, dependency edges, and workspace scope; those are global coordination concerns, not per-session turn history.

`internal/daemon/boot.go` is the only place that should compose the task runtime. If tasks need hooks, observer projections, or API exposure, those should be wired from the daemon downward, following the current pattern for sessions, observer, automation, bridges, and skills.

`internal/observe/observer.go` is a good place for projections, summaries, and health rollups if tasks need cross-task visibility. It should stay read-oriented; task mutation should remain in a dedicated manager/store pair.

The CLI should follow the current `agh session` pattern and get a top-level `agh task` command group that talks to the daemon API client rather than reading files directly. That keeps the control plane consistent with the rest of AGH.

## Risks/Mismatches

Do not hide tasks inside `SessionMeta`, `SessionEvent.Content`, `Session.Channel`, or prompt text. Those fields are session-scoped runtime data, and using them as the source of truth for tasks would create a workaround layer that is hard to query, hard to mutate, and hard to evolve.

Do not use `sessiondb` for task state. It is append-only history for one session, not a multi-entity coordinator. Tasks need ownership and dependency mutations that do not fit the event log model.

Do not make `observe.Observer` the primary task manager. It is explicitly a notifier/projection surface, and turning it into the authority would blur write/read boundaries and create hidden coupling.

Do not add task behavior directly to `internal/daemon/` types beyond composition. The daemon should wire the feature, not become the feature.

Do not repurpose automation jobs as tasks. Automation is trigger/execution oriented, while tasks need explicit ownership, dependency traversal, and possibly subtask fan-out.

Do not encode subtask state as nested sessions unless the intent is really a workflow of independent runtimes. That would inherit session crash/resume semantics and make the task graph much harder to reason about than a dedicated task model.

## Open Questions

- Are tasks global, workspace-scoped, or both? Other AGH surfaces already support both scopes, but the task feature should make that decision explicitly.
- Should a task spawn sessions, or should sessions emit task progress? The direction of authority matters for the long-term model.
- Do subtasks need their own lifecycle/state machine, or are they just edges in a DAG with inherited status?
- Should task progress have its own summary/event projection for `observe`, or is the task table plus linked sessions enough?
- Can network peers create or claim tasks directly, or is task control daemon-local only?

## Evidence

- `internal/session/manager.go:56-80`, `internal/session/manager_lifecycle.go:18-259`, `internal/session/manager_prompt.go:15-269`, `internal/session/query.go:15-205`
- `internal/store/types.go:11-507`, `internal/store/store.go:34-89`, `internal/store/globaldb/global_db.go:16-220`, `internal/store/globaldb/global_db_session.go:12-247`, `internal/store/sessiondb/session_db.go:23-238`
- `internal/api/core/interfaces.go:25-127`, `internal/api/core/handlers.go:168-320`, `internal/api/core/session_stream.go:29-107`, `internal/api/httpapi/routes.go:55-136`, `internal/api/udsapi/routes.go:33-136`
- `internal/observe/observer.go:26-247`, `internal/observe/query.go:14-104`, `internal/observe/health.go:15-154`, `internal/observe/reconcile.go:16-107`, `internal/observe/bridges.go:14-220`
- `internal/daemon/daemon.go:45-238`, `internal/daemon/boot.go:107-250`, `internal/daemon/boundary.go:18-115`, `internal/daemon/composed_assembler.go:13-113`
- `internal/cli/root.go:60-94`, `internal/cli/session.go:16-230`, `internal/cli/daemon.go:46-132`
- `docs/ideas/from-claude-code/analysis_multi_agent.md:223-314`, `docs/ideas/from-claude-code/filtered_recommendations.md:98-113`, `docs/ideas/orchestration/multi-agent-patterns-analysis.md:301-382`
- `.compozy/tasks/_archived/20260410-144004-session-resilience/task_01.md:13-34`, `task_02.md:14-35`, `.compozy/tasks/_archived/20260412-040024-channels/task_01.md:13-28`, `task_07.md:16-31`
