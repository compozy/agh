# Task Memory: task_13.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Make the web SPA workspace-aware for task 13: add workspace list queries, adopt the new session `workspace_id`/`workspace_path` payloads, add a workspace selector for session creation, and show/filter sessions by workspace in the sidebar.

## Important Decisions
- Use a dedicated `web/src/systems/workspace/` module instead of folding workspace fetching into the session system.
- Treat the task PRD/techspec as the approved design and implement directly against the task_10 HTTP contract.
- Keep the UX minimal and shell-level: a single active workspace selection drives new-session creation, while session rows surface the workspace via badge/label and filtering.
- Keep the web shell on registered workspaces only; expose `/api/workspaces/resolve` through the new workspace system for future use, but do not add a path-entry flow in task 13.

## Learnings
- Current web code still expects `session.workspace` and does not parse `workspace_id` or `workspace_path`.
- `useCreateSession()` is only called from `web/src/components/app-sidebar.tsx`, which makes the sidebar the natural place to wire workspace selection.
- Backend contract is already available: `GET /api/workspaces` returns `{ workspaces: [...] }`, `POST /api/sessions` accepts `workspace` or `workspace_path`, and `GET /api/sessions` supports `?workspace=`.
- Session detail can stay presentational by resolving workspace names in the route layer and passing them into `ChatHeader`, while sidebar rows receive workspace labels from the shell-level workspace registry lookup.
- The new workspace system wraps both list and resolve endpoints with Zod-validated payload schemas plus TanStack Query hooks, so future web tasks can reuse it instead of touching session adapters.

## Files / Surfaces
- `web/src/components/app-sidebar.tsx`
- `web/src/components/app-sidebar.test.tsx`
- `web/src/routes/_app/session.$id.tsx`
- `web/src/routes/_app/-session.$id.test.tsx`
- `web/src/systems/session/adapters/session-api.ts`
- `web/src/systems/session/adapters/session-api.test.ts`
- `web/src/systems/session/types.ts`
- `web/src/systems/session/types.test.ts`
- `web/src/systems/session/lib/query-options.ts`
- `web/src/systems/session/lib/query-keys.ts`
- `web/src/systems/session/hooks/use-sessions.ts`
- `web/src/systems/session/hooks/use-sessions.test.tsx`
- `web/src/systems/session/hooks/use-session-actions.ts`
- `web/src/systems/session/components/session-sidebar-item.tsx`
- `web/src/systems/session/components/session-sidebar-item.test.tsx`
- `web/src/systems/session/components/chat-header.tsx`
- `web/src/systems/session/components/chat-header.test.tsx`
- `web/src/systems/agent/components/agent-sidebar-group.tsx`
- `web/src/systems/agent/components/agent-sidebar-group.test.tsx`
- New `web/src/systems/workspace/**`

## Errors / Corrections
- None after implementation. Fresh `make web-lint`, `make web-typecheck`, `make web-test`, and post-commit `make verify` all passed on the final code state.

## Ready for Next Run
- Implementation, self-review, tracking updates, and the local code-only commit are complete.
- Code commit: `db08303` (`feat: add workspace-aware web sessions`).
- Worktree after closeout still contains unrelated user-owned changes only: modified `.compozy/tasks/skills-system/_meta.md` plus the untracked `workspace-entity` PRD directory.
