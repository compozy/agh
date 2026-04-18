---
status: resolved
file: internal/daemon/restart_integration_test.go
line: 142
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:3933db0fc2d2
review_hash: 3933db0fc2d2
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 019: Assertion uses string matching for error validation.
## Review Comment

The error check uses `strings.Contains(err.Error(), "replacement daemon exited before ready")` which is brittle. Consider using `errors.Is` or `errors.As` if a sentinel error is available, or at minimum document that this error message is part of the contract.

Per coding guidelines: "Use `errors.Is()` and `errors.As()` for error matching — never compare error strings."

## Triage

- Decision: `valid`
- Notes:
  Root cause confirmed in the restart integration tests: they match `helper.run()` failures by string contents because the production path returns generic errors for replacement-daemon early exit. I will introduce a typed sentinel for this failure mode and update the tests to assert with `errors.Is` instead of brittle string matching.
