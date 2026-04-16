---
status: resolved
file: internal/api/httpapi/httpapi_integration_test.go
line: 199
severity: minor
author: coderabbitai[bot]
provider_ref: review:4122628916,nitpick_hash:0cb4bfe93104
review_hash: 0cb4bfe93104
source_review_id: "4122628916"
source_review_submitted_at: "2026-04-16T16:31:31Z"
---

# Issue 014: Close the success-path response bodies.
## Review Comment

`putResp.Body` and `deleteResp.Body` are only closed inside the failure branches. Because this integration runtime reuses a shared `http.Client`, leaving them open can pin connections and make later requests or shutdown behavior flaky.

## Triage

- Decision: `INVALID`
- Reason: The success-path response-body leak described in the review is not present in the current file. The `deleteResp` bodies in [internal/api/httpapi/httpapi_integration_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/httpapi/httpapi_integration_test.go#L550), [internal/api/httpapi/httpapi_integration_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/httpapi/httpapi_integration_test.go#L687), and [internal/api/httpapi/httpapi_integration_test.go](/Users/pedronauck/Dev/compozy/_worktrees/site/internal/api/httpapi/httpapi_integration_test.go#L826) are already closed after successful assertions, and there is no `putResp` variable in the current test file.

## Resolution

- Analysis complete. No code change was required because the reviewed leak is already fixed in the current integration tests.
