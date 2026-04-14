# Overview
Paperclip has an explicit task model. It does not rely on a loose checklist abstraction for core coordination; the primary unit is an `issue` with status, ownership, parent/child structure, blockers, comments, approvals, and execution metadata.

The strongest pattern for AGH is not "task list UI" but "issue as coordination record": a task is both a work item and the control surface for agent execution, wakeups, audit history, and dependency tracking.

# Task/Issue Mechanisms
- The core data model lives in `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issues.ts`, where issues carry `parentId`, `goalId`, single-assignee fields, checkout/execution run IDs, terminal timestamps, and execution policy/state JSON.
- Lifecycle is explicit and status-driven. Issue statuses are validated in `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/shared/src/validators/issue.ts` and the service layer treats `backlog`, `todo`, `in_progress`, `in_review`, `blocked`, `done`, and `cancelled` as first-class states.
- Checkout is required before work starts. `server/src/routes/issues.ts` exposes `POST /issues/:id/checkout`, while `server/src/services/issues.ts` enforces atomic ownership and clears locks on release or reassignment.
- Subtasks are modeled through `parentId`, not a separate task type. Child issues can inherit workspace context from the parent, and parent wakeups are triggered when all children reach a terminal state.
- Blockers are first-class relations, not comments. `issue_relations.ts` stores blocker edges, `server/src/services/issues.ts` normalizes and syncs `blockedByIssueIds`, and `GET /issues/:id` exposes `blockedBy` and `blocks`.
- Approvals are adjacent but distinct. `issue_approvals.ts` links issues to approvals, and `server/src/routes/issues.ts` exposes `GET/POST/DELETE /issues/:id/approvals` rather than embedding approval state directly into issue fields.
- Routines are recurring work, but they create execution issues rather than replacing the issue model. See `packages/db/src/schema/routines.ts`, `server/src/routes/routines.ts`, and `doc/TASKS.md` / `doc/TASKS-mcp.md`.

# Relevant Code Paths
- Task contract docs: `/Users/pedronauck/dev/knowledge/.resources/paperclip/doc/TASKS.md`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/doc/TASKS-mcp.md`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/skills/paperclip/SKILL.md`.
- Core API surfaces: `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/routes/issues.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/routes/agents.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/routes/routines.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/routes/issues-checkout-wakeup.ts`.
- Core service logic: `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/services/issues.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/services/issue-approvals.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/services/routines.ts`.
- Shared contracts: `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/shared/src/validators/issue.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/shared/src/types/issue.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/shared/src/types/routine.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/shared/src/api.ts`.
- Storage/schema: `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issues.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issue_relations.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issue_approvals.ts`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/routines.ts`.

# Transferable Patterns
- Use a single durable work record that carries ownership, dependency, ancestry, and execution metadata instead of splitting coordination across many ad hoc tables.
- Make "start work" an explicit atomic action, not a status guess. Paperclip's checkout flow is the clearest reusable idea here.
- Keep subtask propagation and dependency wakeups in the system, not in agent memory. Paperclip uses parent completion and blocker resolution to wake the right assignee.
- Expose a compact agent inbox plus a full issue context endpoint. `GET /agents/me/inbox-lite` and `GET /issues/:id/heartbeat-context` are a good split between prioritization and execution context.
- Keep comments, documents, approvals, and attachments attached to the same task record so the audit trail stays unified.

# Risks/Mismatches
- Paperclip is a company/board/agent control plane; AGH is a daemon for agent sessions. The org-chart and board governance concepts are not automatically transferable.
- Paperclip assumes a single assignee and strict checkout semantics. If AGH needs parallel session workers or fan-out subtasks, that model may need to stay looser.
- Routines, approvals, and execution policies are meaningful in Paperclip because it coordinates a business. They may be overbuilt for AGH core tasks unless AGH explicitly needs governance.
- Paperclip's issue model is PostgreSQL-centric and API-heavy. If AGH core tasks live across session state and event storage, the persistence shape may need to differ.

# Open Questions
- Should AGH have one canonical task table, or separate task records from execution/session records?
- Do AGH subtasks need parent-completion wakeups and blocker edges, or is a smaller sequential workflow enough?
- Should AGH expose a compact "my work" inbox plus a rich task context endpoint, like Paperclip does?
- Which Paperclip adjacent concepts are essential for AGH core tasks: approvals, routines, or just issues/comments/checkouts?

# Evidence
- Vault sources: `/Users/pedronauck/dev/knowledge/paperclip/CLAUDE.md`, `/Users/pedronauck/dev/knowledge/paperclip/log.md`, `/Users/pedronauck/dev/knowledge/paperclip/wiki/codebase/concepts/Codebase Overview.md`, `/Users/pedronauck/dev/knowledge/paperclip/wiki/codebase/index/Codebase Concept Index.md`.
- Repo docs: `/Users/pedronauck/dev/knowledge/.resources/paperclip/README.md`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/doc/CLI.md`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/doc/SPEC-implementation.md`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/doc/TASKS.md`, `/Users/pedronauck/dev/knowledge/.resources/paperclip/doc/TASKS-mcp.md`.
- Repo implementation: `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/routes/issues.ts` (`GET /issues/:id`, `GET /issues/:id/heartbeat-context`, checkout, comments, approvals), `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/routes/agents.ts` (`GET /agents/me/inbox-lite`), `/Users/pedronauck/dev/knowledge/.resources/paperclip/server/src/services/issues.ts` (checkout/release/blockers/ancestor lookup), `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issues.ts` (issue record), `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issue_relations.ts` (blockers), `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/issue_approvals.ts` (approval links), `/Users/pedronauck/dev/knowledge/.resources/paperclip/packages/db/src/schema/routines.ts` (recurring execution issues).
- Notable doc signal: `doc/SPEC-implementation.md` explicitly states "Tasks + comments only (no separate chat system)" and "Single assignee; atomic checkout required for `in_progress` transition", which aligns with the implementation.
