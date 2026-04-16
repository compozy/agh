---
status: resolved
file: internal/daemon/bridges.go
line: 889
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQ_,comment:PRRC_kwDOR5y4QM64dqHO
---

# Issue 046: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**This transition only compensates reload failures, not the other post-write failures.**

Once the updated resource is written and projected, `UpdateInstanceState` and `triggerBridgeResourceReconcile` can still fail, but this branch only rolls back on `reloadExtensions` failure. That leaves the desired-state flip committed even though the transition reports an error.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/bridges.go` around lines 846 - 889, The code only calls
rollbackResourceTransitionState when r.reloadExtensions fails, but other
post-write operations (r.UpdateInstanceState and
r.triggerBridgeResourceReconcile) can also fail and must be compensated; after
successfully writing the updated resource (r.resourceStore.Put) and projecting
it (r.applyBridgeResourcesFromStore), wrap failures from UpdateInstanceState and
triggerBridgeResourceReconcile by invoking r.rollbackResourceTransitionState
with the original currentRecord, the updatedRecord.Version, previous, action and
the error, and return that result (same pattern used for reloadExtensions).
Ensure you perform the rollback call for both UpdateInstanceState errors and
triggerBridgeResourceReconcile errors using the same argument structure as the
existing rollback usage.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `transitionResourceInstance` already compensates reload failures with `rollbackResourceTransitionState`, but the same desired-state write happens before `UpdateInstanceState` and `triggerBridgeResourceReconcile`.
  - If either of those later steps fails, the resource-backed transition currently returns an error after the persisted desired state has already moved forward.
  - Fix approach: route `UpdateInstanceState` and reconcile failures through the existing rollback helper so the resource record and projected runtime state are restored consistently.
  - Resolution: extended the rollback path in `internal/daemon/bridges.go` to cover post-write runtime-state and reconcile failures, with regression coverage in `internal/daemon/bridges_test.go`; verification passed.
