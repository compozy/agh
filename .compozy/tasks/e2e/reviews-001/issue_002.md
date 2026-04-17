---
status: resolved
file: internal/api/httpapi/prompt.go
line: 407
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:773e010321df
review_hash: 773e010321df
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 002: Prefer a typed finish payload over map[string]any.
## Review Comment

`type` and `finishReason` are known fields; a typed struct improves safety and keeps payload contracts explicit.

As per coding guidelines: `Never use interface{}/any when a concrete type is known`.

## Triage

- Decision: `VALID`
- Root cause: `promptStreamState.finish` assembles a known payload contract through `map[string]any`, which is avoidable in package-local code.
- Fix plan: replace the map with a typed finish payload struct and keep the SSE wire format unchanged.
- Resolution: replaced the finish-event `map[string]any` payload with the typed `promptFinishPayload` struct.
- Verification: `go test ./internal/api/httpapi` passed. `make verify` was rerun after the fix set and still fails in unrelated pre-existing `internal/testutil/acpmock` and `internal/testutil/e2e` packages because this branch does not contain `internal/testutil/acpmock/driver/dist/index.js`.
