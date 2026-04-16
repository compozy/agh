---
status: resolved
file: internal/api/core/resources.go
line: 354
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:e3a49fb17ea1
review_hash: e3a49fb17ea1
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 012: Consider whether masking internal errors as 400 is always appropriate.
## Review Comment

The `statusForResourceRequestError` function maps `500 Internal Server Error` to `400 Bad Request`. This could mask legitimate server-side issues during request parsing. Consider whether specific error types should preserve their original status.

```go
// If StatusForResourceError returns 500, it might indicate a real server error
// rather than a client request error. Consider logging these cases.
```

## Triage

- Decision: `INVALID`
- Reason: The referenced file `internal/api/core/resources.go` does not exist in the current tree, and there is no current helper that remaps internal `500` resource errors to `400`. This review comment is stale against an earlier file split.

## Resolution

- Analysis complete. No code change was required because the reviewed file and error path are not present in the current source tree.
