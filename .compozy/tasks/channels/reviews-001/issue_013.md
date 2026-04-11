---
status: resolved
file: internal/api/udsapi/udsapi_integration_test.go
line: 316
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:c8b292964a7a
review_hash: c8b292964a7a
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 013: Add context-wrapped errors in lifecycle helper methods.
## Review Comment

Line 316 through Line 338 return raw errors from `UpdateInstanceState`, which makes failures harder to diagnose in integration runs.

As per coding guidelines, "Use explicit error returns with wrapped context: `fmt.Errorf("context: %w", err)`".

## Triage

- Decision: `valid`
- Notes:
  - The UDS integration lifecycle helpers return raw `UpdateInstanceState` errors, which makes transport failures harder to localize when these helpers are used in broader integration runs.
  - I will wrap the helper errors with operation context and keep the mirrored start/restart readiness behavior consistent with the HTTP integration harness.
  - Resolution: Wrapped the UDS lifecycle helper errors in [internal/api/udsapi/udsapi_integration_test.go](/Users/pedronauck/Dev/projects/_worktrees/channels/internal/api/udsapi/udsapi_integration_test.go:316) and kept the mirrored ready transition behavior aligned; verified with `make verify`.
