---
status: resolved
file: internal/daemon/bridges.go
line: 276
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQ-,comment:PRRC_kwDOR5y4QM64dqHM
---

# Issue 045: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Restore the previous resource record on post-write update failures.**

`putBridgeInstanceResource` commits the new spec before projection, runtime-state sync, reconcile, and readback. If any of those later steps fail, the API returns an error while the desired state stays updated.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/bridges.go` around lines 260 - 276, The current flow calls
putBridgeInstanceResource(current -> next) before several follow-up steps
(applyBridgeResourcesFromStore, applyBridgeUpdateOperationalState,
triggerBridgeResourceReconcile, GetInstance) and never rolls back if a later
step fails; change the logic in the routine around putBridgeInstanceResource so
you first read and keep the existing resource state (the previous record for
current.ID), then call putBridgeInstanceResource(ctx, current, next) and run the
subsequent steps, and if any follow-up call returns an error, re-apply the saved
previous resource by calling putBridgeInstanceResource (or the appropriate store
write) to restore the previous record before returning the error; ensure you
reference the same identifiers used here (putBridgeInstanceResource,
applyBridgeResourcesFromStore, applyBridgeUpdateOperationalState,
triggerBridgeResourceReconcile, GetInstance, current, next) so rollback happens
only on error and the store ends up consistent.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `updateInstanceResource` updates the canonical resource record before projection, optional runtime-state sync, reconcile, and readback.
  - A failure in any follow-up step currently leaves the new desired state committed even though the API reports the update failed.
  - Fix approach: keep the pre-update resource record and previous runtime instance, restore both on any post-write failure, and combine the original error with any rollback failure.
  - Resolution: implemented resource-record and runtime-state rollback in `internal/daemon/bridges.go`, with regression coverage in `internal/daemon/bridges_test.go`; verification passed.
