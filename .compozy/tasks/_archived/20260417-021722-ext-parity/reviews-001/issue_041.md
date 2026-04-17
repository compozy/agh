---
status: resolved
file: internal/daemon/boot.go
line: 405
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:576b09749211
review_hash: 576b09749211
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 041: Potential nil dereference if resourceReconcile is set after this callback is created.
## Review Comment

The trigger callback captures `state.resourceReconcile` by reference, but `resourceReconcile` is set later in `bootResourceReconcile`. The nil check inside the callback handles this correctly, but it creates a subtle ordering dependency. Consider documenting this expectation or restructuring to make the dependency explicit.

```go
// The callback correctly handles nil but relies on resourceReconcile being set
// later in the boot sequence (by bootResourceReconcile).
```

## Triage

- Decision: `invalid`
- Notes:
  - `bootRuntimeServices` intentionally installs the bridge resource trigger callback before `bootResourceReconcile` constructs the driver, and the callback explicitly guards `state.resourceReconcile == nil`.
  - The bridge runtime only uses that callback after resource definitions are wired, while the daemon boot sequence calls `bootResourceReconcile` before extensions start mutating bridge resources and `RunBoot` handles the initial reconciliation pass.
  - This is an expected boot-order dependency shared by the other resource-backed publishers in the same file, not a latent nil-dereference bug.
  - Resolution: no production change required; repository verification passed after resolving the valid issues in this batch.
