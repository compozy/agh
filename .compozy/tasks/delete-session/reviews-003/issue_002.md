---
status: resolved
file: internal/task/manager.go
line: 2105
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59Rqy6,comment:PRRC_kwDOR5y4QM663P3P
---

# Issue 002: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Keep canonical status calculation side-effect free.**

`canonicalTaskStatus()` is used by read paths like `GetTask` and `ListTasks`, but this helper now resolves dependencies by calling `reconcileTaskWithStore()`, which can persist via `store.UpdateTask()`. That means a read-only request can mutate dependency records and effectively perform writes under read authority.



Also applies to: 2295-2302

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 2094 - 2105, The
canonicalTaskStatusWithStore currently calls reconcileTaskWithStore (which can
call store.UpdateTask) and thus performs writes during read operations; change
it to be side-effect free by removing any calls to reconcileTaskWithStore or
other mutating helpers (and by not calling store.UpdateTask), instead deriving
status purely from read-only helpers like hasUnresolvedDependenciesWithStore,
the provided dependencies and runs, and any non-mutating logic; if
reconciliation logic is needed elsewhere, introduce a separate read-only variant
(e.g., reconcileTaskReadOnly or computeCanonicalStatusNoSideEffects) or ensure
reconcileTaskWithStore has a non-mutating mode and use that here, and update
canonicalTaskStatus and callers to use the new read-only pathway.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `canonicalTaskStatusWithStore()` asks `hasUnresolvedDependenciesWithStore()` whether dependencies are resolved, and that helper currently calls `reconcileTaskWithStore()` for each dependency. `reconcileTaskWithStore()` can persist through `store.UpdateTask()`, so read-only paths such as `GetTask`, `ListTasks`, and dependency-reference hydration can write task records while computing display status.
- Evidence: `GetTask`, `ListTasks`, `enrichTaskSummaryFromState`, and `taskReference` all route through `canonicalTaskStatus(...)`; the dependency walk in `hasUnresolvedDependenciesWithStore()` is therefore reachable under read authority and currently crosses a mutating reconciliation boundary.
- Fix approach: replace the dependency-status walk used by canonical reads with a recursive, side-effect-free status calculator that only uses `GetTask`, `ListDependencies`, and `ListTaskRuns`. Keep persistence inside explicit reconciliation paths such as `reconcileTaskWithStore()` and `reconcileTaskCascadeWithStore()`.
- Test plan: add regression coverage in `internal/task/manager_test.go` to prove `GetTask`/`ListTasks` compute dependency-derived statuses correctly without mutating stored dependency records. This requires one additional test file beyond the scoped code file because `internal/task/manager.go` has no co-located test cases for this read-only behavior.

## Resolution

- Reworked canonical dependency-status evaluation in `internal/task/manager.go` so read-time status calculation uses a recursive, side-effect-free helper instead of `reconcileTaskWithStore()`.
- Left persistence inside explicit reconciliation flows only; `GetTask`, `ListTasks`, and dependency-reference reads now derive the right status without calling `UpdateTask()`.
- Added regression coverage in `internal/task/manager_test.go` that seeds a stale dependency record, exercises both `GetTask` and `ListTasks`, and asserts the returned derived status is correct while the stored dependency status remains unchanged.
- Verified with `go test ./internal/session ./internal/task` and `make verify` (both exit `0`).
