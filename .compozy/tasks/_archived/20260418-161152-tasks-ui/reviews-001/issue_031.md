---
status: resolved
file: internal/task/manager.go
line: 1369
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM575lb_,comment:PRRC_kwDOR5y4QM65B8fe
---

# Issue 031: _⚠️ Potential issue_ | _🔴 Critical_
## Review Comment

_⚠️ Potential issue_ | _🔴 Critical_

**`max_attempts` and idempotency are checked non-atomically.**

`prepareEnqueueRun` reads the idempotency mapping and computes `nextAttempt` before `CreateTaskRun` and `SaveTaskRunIdempotency`. Two concurrent enqueue requests can both observe the same state and create duplicate attempts, or both miss the idempotency record and create duplicate runs. This needs store-level atomicity, e.g. a transaction plus unique constraints.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 1315 - 1369, prepareEnqueueRun is
vulnerable to race conditions because idempotency lookup (lookupIdempotentRun)
and attempt calculation (nextEnqueueAttempt which calls
ListTaskRuns/nextRunAttempt) happen outside any atomic store operation; update
the flow to perform these steps inside a single store-level atomic operation
(e.g., a DB transaction or a store method that does it atomically) so that
creating the TaskRun and saving the idempotency mapping (CreateTaskRun and
SaveTaskRunIdempotency) are done together with the attempt check (or enforce a
unique constraint and handle conflicts); concretely implement a new store method
(or transaction wrapper) that: checks for an existing idempotent run,
computes/validates next attempt against
normalizeTaskMaxAttemptsOrDefault(task.MaxAttempts), creates the run, and writes
the idempotency mapping atomically, returning a duplicate/conflict error if
another concurrent enqueuer wins.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `prepareEnqueueRun` performs idempotency lookup and next-attempt calculation before `CreateTaskRun`/`SaveTaskRunIdempotency`, so concurrent enqueue calls can race and create duplicate attempts or duplicate runs for the same idempotency key.
- Fix approach: move enqueue allocation into one store-level atomic operation that checks existing idempotency, validates the next attempt against `max_attempts`, creates the run, and writes the idempotency binding under the same immediate transaction. This likely requires a minimal store-interface expansion outside the strict batch file list; if so, the change will be kept as small as possible.

## Resolution

- Added `ReserveQueuedRun(...)` as a store-level atomic reservation API and switched `Service.EnqueueRun` to use it.
- Implemented the global DB path under one immediate transaction so idempotency lookup, attempt allocation, run insert, and idempotency binding happen atomically.
- Added a concurrent regression test proving duplicate idempotent enqueue requests collapse onto a single persisted run.
- Verification: `go test ./internal/task`, `go test ./internal/store/globaldb`, `go test -tags integration ./internal/task`, and `go test -tags integration ./internal/store/globaldb`
