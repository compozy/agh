# Task Memory: task_05.md

Keep only task-local execution context here. Do not duplicate facts that are obvious from the repository, task file, PRD documents, or git history.

## Objective Snapshot
- Implement the async hook worker pool in `internal/hooks/pool.go` with configurable worker count, queue capacity, drop-on-full backpressure, panic recovery, and bounded shutdown.
- Prove the task requirements with focused `internal/hooks` tests, `-race`, and package coverage >=80%.

## Important Decisions
- Kept the pool package-private as `asyncPool` with `asyncTask` and `asyncPoolConfig`; task 06 can own it from the `Hooks` struct without widening the public hooks surface.
- `Submit` uses a buffered channel protected by an `RWMutex` so non-blocking sends cannot race a concurrent `Close()` and panic on a closed channel.
- On shutdown deadline, the pool discards any still-buffered tasks before canceling worker contexts so queued async hooks are abandoned instead of running after the deadline.

## Learnings
- The worker context is pool-owned; future async hook dispatch code must wrap any per-hook timeout inside the submitted `task.run` closure.
- A closed channel plus a canceled context are both selectable, so abandoning queued work after the drain deadline requires explicitly draining the buffer before cancellation.

## Files / Surfaces
- `internal/hooks/pool.go`
- `internal/hooks/pool_test.go`
- `.compozy/tasks/hooks/memory/MEMORY.md`
- `.compozy/tasks/hooks/task_05.md`
- `.compozy/tasks/hooks/_tasks.md`

## Errors / Corrections
- Initial panic-recovery test could fill the single-slot queue before the worker started; fixed by waiting for the first task to begin before submitting the recovery task.
- Initial shutdown implementation could still run a buffered task after the drain deadline because the worker `select` could choose the closed channel over `ctx.Done()`; fixed by discarding queued tasks on timeout before canceling workers.

## Ready for Next Run
- Verification evidence: `go test -race -cover ./internal/hooks` passed with 83.9% coverage, and `make verify` passed after the pool/test changes.
- Remaining follow-up is task 06 wiring: submit async hook executions through `asyncPool` and apply per-hook timeout inside each submitted closure.
