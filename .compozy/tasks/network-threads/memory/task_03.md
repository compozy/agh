# Task Memory: task_03.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot

- Implement runtime-only work lifecycle and direct-room identity primitives for Task 03.
- Success requires no active `interaction` lifecycle symbols in `internal/network`, deterministic domain-separated direct IDs, same-peer rejection, collision-signaling primitives, conversation-bound work lifecycle validation, duplicate-before-lifecycle idempotency, terminal rejection, tests, tracking updates, clean `make verify`, and one local commit.

## Important Decisions

- Follow `task_03.md` for execution scope because it narrows this run to primitives and explicitly defers durable SQLite rows to later store tasks; use the TechSpec/ADRs as semantic authority for direct-room identity and lifecycle invariants.
- The TechSpec MVP boundary still labels old Task 03 as store migration work; treat that as stale task-numbering context, not current scope.

## Learnings

- Task 02 shared memory says runtime already has `network.SurfaceThread`, `network.SurfaceDirect`, `network.ConversationRef`, `network.DirectRoomIdentity`, `WorkState`, `OpenWork`, and `ApplyWorkEnvelope`; implementation must verify actual code and close remaining gaps.
- Required named skills `nats`, `agh-code-guidelines`, and `agh-test-conventions` are not installed in this session; proceed with available skills and repo guidance.
- `DirectRoomIdentity("builders", "coder.sess-abc", "reviewer.sess-xyz")` now has known vector `direct_99401d24bee62651d189e5a561785466` and is peer-order independent/channel-scoped.
- Store-facing collision checks are exposed through `ErrDirectRoomCollision` and `ValidateDirectRoomBinding`, without adding store imports to `internal/network`.
- Work terminal transitions set `Work.TerminalAt`; new non-duplicate messages after terminal state now return `LifecycleActionRejectWork` with `ReasonCodeWorkClosed`.
- Cross-container continuation for an existing `work_id` now fails with `ErrWorkContainerMismatch` and router maps that to `ReasonCodeWorkContainerMismatch`.
- Exact duplicate receive replay is covered before lifecycle handling by router tests.

## Files / Surfaces

- Code surfaces: `internal/network/lifecycle.go`, `internal/network/validate.go`, `internal/network/router.go`, and network tests in `internal/network/*_test.go`.
- Tracking surfaces: `.compozy/tasks/network-threads/task_03.md`, `.compozy/tasks/network-threads/_tasks.md`, and this memory file.

## Errors / Corrections

- First `make verify` attempt failed Go lint on an unused `terminalWorkResult` parameter after terminal behavior changed; removed the unused parameter and reran verification.
- Pre-commit verification evidence: `go test -count=1 ./internal/network` passed; `go test -count=1 -cover ./internal/network` passed with 82.0% coverage; `make verify` passed exit 0.
- First post-commit `make verify` attempt hit an unrelated timeout in `sdk/typescript/src/integration.test.ts` (`SDK integration > builds an SDK-based extension and serves real JSON-RPC over stdio`, 30000ms). Targeted rerun `bunx vitest run src/integration.test.ts --config vitest.config.ts` from `sdk/typescript` passed with 2/2 tests, and the subsequent full `make verify` passed exit 0.

## Ready for Next Run

- Task 03 implementation, tracking updates, local commit, and post-commit verification are complete.
- Local commit: `78a714be feat: add network work lifecycle primitives`.
