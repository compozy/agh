---
status: resolved
file: internal/task/manager.go
line: 305
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM59RYXu,comment:PRRC_kwDOR5y4QM6622vR
---

# Issue 006: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**This delete can fail after the task is already gone.**

Once `m.store.DeleteTask` succeeds, any later `reconcileTaskCascade` error returns a failure even though the resource has already been deleted. That leaves dependent statuses stale and makes client retries observe `404` after a prior `500`. This needs an atomic delete+reconcile path in the store layer, or a guaranteed post-delete repair mechanism.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/task/manager.go` around lines 293 - 305, The current flow calls
m.store.DeleteTask(ctx, trimmedID) then runs reconcileTaskCascade for
dependents, which can return an error after the resource is already deleted; fix
by making delete+reconcile atomic at the store layer or by guaranteeing
post-delete repair: add a new store method (e.g.,
DeleteTaskAndReconcile/DeleteTaskWithCascade) that performs the delete and
dependent reconciliation inside the same transactional boundary (or returns
dependentIDs and reconciliation results) and call that from Manager instead of
m.store.DeleteTask, or if transactions are unavailable, modify manager.go to
treat a successful DeleteTask as definitive (do not return failure on
reconcileTaskCascade errors), log the reconcile error and enqueue an
asynchronous retry/repair job (or schedule background goroutine) so the API
returns success while dependents are eventually reconciled; update references to
m.store.DeleteTask and reconcileTaskCascade accordingly.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `internal/task/manager.go` deletes the primary task first and only then reconciles dependent task statuses. If reconciliation fails after the delete succeeds, the method returns an error even though the target task is already gone.
  - That creates a real success-reporting bug: callers can observe a failed delete response and then get `ErrTaskNotFound` on retry, while dependent durable statuses may remain unreconciled.
  - Planned fix: run the delete plus dependent reconciliation inside one store-backed transaction when the store supports it, so a reconcile failure rolls back the delete. This will require minimal supporting changes outside the scoped file list in task store interfaces and tests, which will be kept tightly limited and documented here.

## Resolution

- Added a transactional delete path so `internal/task/manager.go` now performs task deletion and dependent-status reconciliation inside one store-backed unit when the store supports `WithDeleteTaskTransaction(...)`.
- Extended the global task store and the task manager test harness to support that transactional path and to prove rollback when dependent reconciliation fails.
- This removes the false-failure window where the API could return an error after the task had already been permanently deleted.
- Verified with `make verify` (exit `0`).
