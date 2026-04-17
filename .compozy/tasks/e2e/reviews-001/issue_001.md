---
status: resolved
file: internal/api/httpapi/handlers_test.go
line: 715
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:88cef7d4dc52
review_hash: 88cef7d4dc52
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 001: Use a typed finish payload in this test instead of map[string]any.
## Review Comment

The parsed shape is known (`finishReason`, optional `stopReason`), so a typed struct is safer and clearer.

As per coding guidelines: `Never use interface{}/any when a concrete type is known`.

## Triage

- Decision: `VALID`
- Root cause: the test decodes the `done` SSE payload into `map[string]any` even though the finish payload shape is known and stable in this package.
- Fix plan: introduce a typed finish payload and decode the test fixture into that struct so the assertions become compile-time checked.
- Resolution: added `promptFinishPayload` and updated the handler test to decode the `done` event into that typed payload.
- Verification: `go test ./internal/api/httpapi` passed. `make verify` was rerun after the fix set and still fails in unrelated pre-existing `internal/testutil/acpmock` and `internal/testutil/e2e` packages because this branch does not contain `internal/testutil/acpmock/driver/dist/index.js`.
