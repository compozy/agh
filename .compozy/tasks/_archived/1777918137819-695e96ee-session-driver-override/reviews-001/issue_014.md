---
status: resolved
file: internal/session/resume_repair.go
line: 62
author: coderabbitai[bot]
provider_ref: thread:PRRT_kwDOR5y4QM581azv,comment:PRRC_kwDOR5y4QM66RFPZ
---

# Issue 014: _⚠️ Potential issue_ | _🟠 Major_
## Review Comment

_⚠️ Potential issue_ | _🟠 Major_

**Don't replace a missing caller context with `context.Background()`.**

These paths cross runtime boundaries, so falling back to a detached background context means resume/reconcile work can outlive shutdown and cancellation. It's safer to require a real caller context or fail fast when one isn't provided.


As per coding guidelines, `context.Context` as first argument to functions crossing runtime boundaries — avoid `context.Background()` outside `main` and focused tests`.


Also applies to: 100-102

<details>
<summary>🤖 Prompt for AI Agents</summary>

```
Verify each finding against the current code and only fix it if needed.

In `@internal/session/resume_repair.go` around lines 55 - 62, The function
repairInactiveMeta currently replaces a nil caller context with
context.Background(), which can let work outlive caller cancellation; instead,
change repairInactiveMeta to validate that ctx is non-nil and return an error if
it is nil (require callers to pass a real context). Update the same pattern in
the related functions referenced around lines 100-102 so none of these
boundary-crossing methods substitute context.Background(); they should all fail
fast on a nil ctx and document that callers must supply a proper context.
```

</details>

<!-- fingerprinting:phantom:medusa:grasshopper -->

<!-- This is an auto-generated comment by CodeRabbit -->

## Triage

- Decision: `valid`
- Root cause: `repairInactiveMeta` and `RepairLegacyProvider` silently replace a nil caller context with `context.Background()`, which detaches resume/reconcile work from shutdown and cancellation.
- Fix plan: reject nil contexts at those boundaries instead of inventing a detached context. This may require a minimal scope exception to extend targeted regression coverage in `internal/session/resume_repair_test.go`.
