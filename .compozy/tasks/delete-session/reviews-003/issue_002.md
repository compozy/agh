---
status: pending
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

- Decision: `UNREVIEWED`
- Notes:
