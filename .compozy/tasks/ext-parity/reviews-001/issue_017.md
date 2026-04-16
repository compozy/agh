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

# Issue 017: Close the success-path response bodies.
## Review Comment

`putResp.Body` and `deleteResp.Body` are only closed inside the failure branches. Because this integration runtime reuses a shared `http.Client`, leaving them open can pin connections and make later requests or shutdown behavior flaky.

## Triage

- Decision: `VALID`
- Notes: `putResp.Body` and `deleteResp.Body` are not closed on the success path in this integration test. Because the runtime reuses a shared `http.Client`, leaving those bodies open can leak connections across later requests. The fix is to close them unconditionally.
