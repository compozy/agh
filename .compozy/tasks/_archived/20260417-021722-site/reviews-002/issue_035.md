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

# Issue 035: Potential nil dereference if resourceReconcile is set after this callback is created.
## Review Comment

The trigger callback captures `state.resourceReconcile` by reference, but `resourceReconcile` is set later in `bootResourceReconcile`. The nil check inside the callback handles this correctly, but it creates a subtle ordering dependency. Consider documenting this expectation or restructuring to make the dependency explicit.

```go
// The callback correctly handles nil but relies on resourceReconcile being set
// later in the boot sequence (by bootResourceReconcile).
```

## Triage

- Decision: `INVALID`
- Notes:
  - The current `internal/daemon/boot.go` has no `resourceReconcile` field or callback ordering dependency matching the comment.
  - A repo-wide search found no surviving `resourceReconcile` symbol in the live tree.
  - This review comment is stale against older boot sequencing code.
  - Result: resolved as stale after current-tree inspection; no code change required.
