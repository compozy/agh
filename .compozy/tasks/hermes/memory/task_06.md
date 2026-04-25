# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement Task 06: shared durable process registry and scoped interrupts for long-running tool subprocesses, including PID/start-time reconciliation, owner-scoped cancellation, integration across ACP/environment/hooks/extensions/subprocess helpers, tests, impact assessment, verification, tracking updates, and one local commit.

## Important Decisions

- Registry placement must be a shared runtime package (`internal/toolruntime` per task/ADR), not `session.Manager` or `environment.ToolHost`.
- Recovered process signaling must validate ownership plus start-time evidence; PID-only signaling is prohibited.
- Daemon boot owns construction of one shared `toolruntime.Registry` from the global DB store and passes it into sessions, environment providers, hooks, extensions, ACP, and subprocess helpers.
- `CancelPrompt` remains cooperative first; scoped registry interruption is a follow-on for only the active session turn and treats no matching process as a no-op.

## Learnings

- Task 01 foundations are available: `internal/store.RunMigrations` for durable schema work and `internal/retry` for context-aware retry/backoff if reconciliation needs transient retries.
- Current worktree has unrelated untracked Hermes tracking/memory files and an unrelated `.codex/plans/2026-04-24-site-bento-section.md`; avoid touching unrelated files except required Task 06 tracking/memory updates.
- No `web/` code or generated client update is required because Task 06 does not change HTTP/SSE/API payload contracts. `packages/site` needs operator docs for restart reconciliation and scoped interrupts.
- Remote environment terminal records have no reusable local PID evidence after daemon restart, so reconciliation retires them stale instead of signaling by PID.
- Documentation self-review corrected the `tool_processes` inspection query to use actual owner columns (`session_id`, `turn_id`).

## Files / Surfaces

- Planned inspection surfaces: `internal/toolruntime` (new), `internal/acp`, `internal/environment`, `internal/hooks`, `internal/extension`, `internal/subprocess`, `internal/procutil`, `internal/daemon`, API/web/docs impact points, and `.compozy/tasks/hermes/task_10.md` QA follow-up.
- Implemented/touched surfaces: `internal/toolruntime`, `internal/store/globaldb`, `internal/procutil`, `internal/subprocess`, `internal/acp`, `internal/environment/local`, `internal/environment/daytona`, `internal/hooks`, `internal/extension`, `internal/session`, `internal/daemon`, `packages/site/content/runtime/core/operations/{daemon,database}.mdx`, and `.compozy/tasks/hermes/task_10.md`.

## Errors / Corrections

- Corrected hook subprocess registration-failure cleanup to use bounded terminate-then-kill behavior instead of an unbounded wait.
- Corrected site database docs after review: the new `tool_processes` table stores owner IDs as `session_id` and `turn_id`, not `owner_session_id` / `owner_turn_id`.

## Ready for Next Run

- Targeted verification passed: `go test ./internal/toolruntime ./internal/procutil ./internal/subprocess ./internal/acp ./internal/hooks ./internal/extension ./internal/environment/local ./internal/environment/daytona ./internal/session ./internal/daemon ./internal/store/globaldb`.
- Final verification passed after the final docs/tracking edits: `make verify` exited 0 with web format/lint/typecheck/tests/build, Go lint, `DONE 5820 tests`, and package boundary checks passing.
- Local implementation/docs commit created: `0f3e1893 feat: add tool process registry`.
