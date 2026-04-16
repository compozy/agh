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

# Issue 015: Consider whether masking internal errors as 400 is always appropriate.
## Review Comment

The `statusForResourceRequestError` function maps `500 Internal Server Error` to `400 Bad Request`. This could mask legitimate server-side issues during request parsing. Consider whether specific error types should preserve their original status.

```go
// If StatusForResourceError returns 500, it might indicate a real server error
// rather than a client request error. Consider logging these cases.
```

## Triage

- Decision: `INVALID`
- Notes: `statusForResourceRequestError` is only used on request-shape parsing and validation paths in `internal/api/core/resources.go`. Backend service failures still bypass this helper and go through `StatusForResourceError(err)` directly. In the current call graph, this helper is not masking real server-side resource-service failures.
