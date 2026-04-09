# Task Memory: task_06.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement task_06's `internal/hooks` `Hooks` struct, typed dispatchers, atomic registry rebuild/swap, `session.Notifier` bridge, and unit tests.
- Pre-change baseline: `internal/hooks` has the base types, normalization, executors, pipeline, and async pool, but no `hooks.go`, `dispatch.go`, or `notifier.go`, and no `Hooks` implementation.

## Important Decisions
- Use four declaration-provider seams plus one executor resolver so the registry can already rebuild from native/config/agent/skill sources without pulling in task_07/task_08 package dependencies yet.
- Compare rebuild results using normalized/sorted hook metadata fingerprints instead of executor pointer identity so unchanged declarations skip the swap and preserve the version counter.
- Keep `OnAgentEvent` conservative for task_06: bridge only the notifier surface that can be derived from the current ACP event payloads, and leave broader session/input/prompt/event integration for task_10.

## Learnings
- The explicit dispatch surface is large enough that package coverage only cleared the `>=80%` gate after adding direct family-level tests for the typed patch applicators, not just smoke calls and pipeline tests.
- Matching async hooks before the sync pipeline and then executing them with the pool-owned worker context preserved the ADR requirement that already-matched async hooks still run after sync short-circuit paths.

## Files / Surfaces
- `.codex/ledger/2026-04-09-MEMORY-hooks-struct.md`
- `.compozy/tasks/hooks/memory/MEMORY.md`
- `internal/hooks/hooks.go`
- `internal/hooks/dispatch.go`
- `internal/hooks/notifier.go`
- `internal/hooks/hooks_test.go`
- `internal/hooks/dispatch_integration_test.go`
- `.compozy/tasks/hooks/memory/task_06.md`

## Errors / Corrections
- The first compile pass exposed three concrete issues: a helper name collision with `pipeline_test.go`, a missing `time` import in `notifier.go`, and an invalid direct struct comparison in a no-hooks dispatch test because the payload carried slices.
- Focused coverage initially landed at `72.3%`; corrected by adding family-level typed dispatch tests plus a full exported-dispatch smoke pass, bringing `internal/hooks` coverage to `85.2%`.
- The first local commit unintentionally included unrelated staged review-doc deletions that were already present in the worktree; corrected by restoring those docs in a separate follow-up commit instead of rewriting history.

## Ready for Next Run
- Task 06 implementation is complete and verified.
- Verification evidence:
  - `go test ./internal/hooks -count=1`
  - `go test -race -cover ./internal/hooks -count=1` with `85.2%` coverage
  - `go test -tags integration ./internal/hooks -count=1`
  - `make verify`
- Final handoff turn re-ran the same verification commands successfully before reporting completion.
- Local commits created during this run:
  - `b2b3a01 feat: implement hooks dispatcher core`
  - `2bd9796 docs: restore review docs`
- Remaining follow-up belongs to later tasks: task_09 wires `Hooks` into the daemon and task_10 expands runtime event integration beyond the current notifier bridge.
