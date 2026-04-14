# Overview
GoClaw does not model "tasks" as a single primitive. It uses three coordination layers:

- Subagents for self-cloned parallel work.
- Delegation for permission-gated agent-to-agent handoffs.
- Team tasks for the shared board / issue-like lifecycle.

For AGH core tasks/subtasks, the important pattern is the shared task-board layer, not the full delegation stack. GoClaw's implementation is intentionally richer than a minimal task engine: it carries ownership, locks, progress, comments, attachments, audit events, notification routing, and post-turn recovery.

# Task/Issue Mechanisms
There is no separate first-class "issue" entity in the evidence collected. The issue-like surface is `team_tasks`:

- Each task has a UUID, a sequential `task_number`, and a human-readable `identifier` like `T-003-xxxx`.
- Task scope is explicit: `team_id`, `channel`, `chat_id`, and sometimes `peer_kind` / `local_key`.
- Ownership is explicit: `owner_agent_id`, `created_by_agent_id`, `assignee_user_id`, and `status`.
- Lifecycle states include `pending`, `in_progress`, `in_review`, `completed`, `blocked`, `failed`, `cancelled`, and `stale`.
- Dependencies use `blocked_by`, with automatic unblocking when blockers finish.
- Context lives alongside the task: comments, event log, attachments, progress fields, and follow-up metadata.

The task tool surface is broad. `team_tasks` supports create, list, get, claim, complete, cancel, approve, reject, comment, progress, attach, update, ask_user, clear_ask_user, retry, and search. That breadth is useful for humans, but likely too much for AGH core tasks v1.

# Relevant Code Paths
Vault sources:

- `qmd://goclaw/wiki/concepts/agent-teams-and-delegation.md` describes the three coordination primitives, team roles, task lifecycle, and task dispatch.
- `qmd://goclaw/wiki/concepts/gateway-and-rpc-protocol.md` describes WS RPC exposure and the gateway control plane.
- `qmd://goclaw/wiki/concepts/http-api-surface.md` describes the HTTP surface and confirms team/task control is mostly gateway-led, not broad REST CRUD.
- `qmd://goclaw/wiki/concepts/security-rbac-and-crypto.md` describes the layered permission model.
- `qmd://goclaw/wiki/concepts/web-and-desktop-ui-layer.md` shows the UI is shared React + Wails, with the frontend consuming the same backend protocol.

Repo sources:

- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/team_store.go` defines the task data model and store interfaces.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/pg/teams_tasks.go` implements task creation, list filtering, identifiers, and scope filtering.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/pg/teams_tasks_lifecycle.go` implements claim, assign, complete, cancel, fail, review, approve, reject, and dependency unblocking.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/pg/teams_tasks_activity.go` persists comments, audit events, and attachments.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tasks_tool.go` defines the agent-facing task tool schema.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tasks_lifecycle.go` maps claim/complete/approve/reject actions onto store transitions and broadcasts events.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tasks_read.go` shows list/get/search shapes and how blockers are expanded.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tool_validation.go` enforces dependency validity, team membership, and escalation policy.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/gateway/methods/teams_tasks.go` and `teams_tasks_mutations.go` expose RPC methods for task CRUD and dispatch.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/http/team_events.go` exposes event history over HTTP.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway.go` wires the WS + HTTP control plane.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway_managed.go` wires team tools in managed mode.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway_consumer_post_turn.go` handles post-turn auto-complete / auto-fail / dispatch.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway_lifecycle.go` starts task recovery on boot.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/channels_cmd.go` and `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/setup_cmd.go` expose the CLI setup surface, but not the task lifecycle itself.
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/mcp/bridge_server.go` exposes `team_tasks` over the MCP bridge.

# Transferable Patterns
The strongest transferable ideas for AGH are:

- Use a single persisted task record with a small number of stable identifiers: UUID for storage, sequence number for human UX, and a human-readable stable identifier.
- Separate current task state from immutable task history. GoClaw keeps comments, audit events, and attachments in separate tables instead of stuffing everything into the task row.
- Make ownership explicit and machine-checkable. Claim/assign are separate operations, and the owner is tracked separately from creator/assignee.
- Use state transitions as the source of truth, not optimistic UI state. Claim, complete, approve, reject, and cancel all hit the store first, then broadcast events.
- Treat dependencies as data, not implicit workflow. `blocked_by` is persisted and later resolved.
- Keep task execution and task announcement separate. GoClaw dispatches work first, then announces results after the run finishes.
- Use locks and renewal for long-running work. The task lock plus renewal heartbeat is a practical pattern for avoiding stale recovery races.
- Gate task actions by role and team membership, not by caller intent alone.

# Risks/Mismatches
GoClaw is richer than AGH core tasks should probably be at first:

- It has three coordination primitives, not one. Copying subagents, delegation, and team tasks all at once would overcomplicate AGH.
- It encodes a lead/member team hierarchy and a large action matrix. AGH should not inherit that unless it truly needs multi-agent project orchestration.
- It uses several transport surfaces for the same concept: agent tool, WS RPC, HTTP event history, export/import, and MCP bridge. AGH should keep the surface area smaller unless there is a real user need.
- It relies on managed-mode / PostgreSQL assumptions for the richer coordination model. AGH should not force a storage/runtime split unless the architecture calls for it.
- Its notification/announce pipeline is heavily optimized for conversational agents. AGH core tasks may not need that much automatic rephrasing or batching.
- The lifecycle is tightly coupled to message-bus side effects and post-turn recovery. For AGH, that coupling should be introduced only where it provides clear value.

# Open Questions
- Should AGH core work items be only one persisted task type with parent/child links, or should it have a separate issue layer?
- Should AGH keep subagents as a separate runtime primitive, or model them as nested tasks under the same lifecycle?
- How much of GoClaw's team-role model is actually needed for AGH, versus a simpler owner/assignee/reviewer model?
- Do we need task comments, event history, and attachments on day one, or just the task row plus status transitions?

# Evidence
- `qmd://goclaw/wiki/concepts/agent-teams-and-delegation.md`
- `qmd://goclaw/wiki/concepts/gateway-and-rpc-protocol.md`
- `qmd://goclaw/wiki/concepts/http-api-surface.md`
- `qmd://goclaw/wiki/concepts/security-rbac-and-crypto.md`
- `qmd://goclaw/wiki/concepts/web-and-desktop-ui-layer.md`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/team_store.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/pg/teams_tasks.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/pg/teams_tasks_lifecycle.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/store/pg/teams_tasks_activity.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tasks_tool.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tasks_lifecycle.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tasks_read.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/tools/team_tool_validation.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/gateway/methods/teams_tasks.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/gateway/methods/teams_tasks_mutations.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/http/team_events.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway_managed.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway_consumer_post_turn.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/gateway_lifecycle.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/channels_cmd.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/cmd/setup_cmd.go`
- `/Users/pedronauck/dev/knowledge/.resources/goclaw/internal/mcp/bridge_server.go`
