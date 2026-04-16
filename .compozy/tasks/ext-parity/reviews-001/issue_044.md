---
status: resolved
file: internal/daemon/bridges.go
line: 223
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM57fTQ5,comment:PRRC_kwDOR5y4QM64dqHI
---

# Issue 044: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Rollback the new resource if a later create step fails.**

After `Put` succeeds, `applyBridgeResourcesFromStore`, `triggerBridgeResourceReconcile`, and even the final `GetInstance` can still fail. Returning an error without deleting the just-created resource leaves a bridge the caller believes was never created.

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/daemon/bridges.go` around lines 201 - 223, After successfully
creating the resource with r.resourceStore.Put, ensure you roll it back if any
subsequent step fails: wrap the calls to r.applyBridgeResourcesFromStore,
r.triggerBridgeResourceReconcile, and r.GetInstance so that on any error you
call the resource removal operation on the same store/actor (e.g.
r.resourceStore.Delete or equivalent using r.resourceActorForSource(spec.Source)
and id) to remove the newly created resources.Draft entry; if the delete itself
fails, return a combined/wrapped error that includes both the original failure
and the deletion failure, and only return the created instance when all
subsequent steps succeed.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Notes:
  - `createInstanceResource` writes the desired-state record before `applyBridgeResourcesFromStore`, `triggerBridgeResourceReconcile`, and `GetInstance`.
  - If any of those later steps fail, the method currently returns an error while leaving the created resource record and projected bridge instance behind.
  - Fix approach: capture the created resource version, delete that record on any post-write failure, re-apply the bridge projection, and wrap rollback failures together with the original error.
  - Resolution: implemented rollback deletion and projection restore in `internal/daemon/bridges.go`, with regression coverage in `internal/daemon/bridges_test.go`; verification passed.
