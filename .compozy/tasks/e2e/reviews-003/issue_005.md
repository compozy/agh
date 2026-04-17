---
status: resolved
file: internal/daemon/daemon_nightly_combined_integration_test.go
line: 233
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:9fbf77690a5d
review_hash: 9fbf77690a5d
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 005: Same pattern: diagnostics populated with expected values for capture.
## Review Comment

Similar to the environment tests, the `diagnostics` struct records expected values rather than observed runtime data. This is consistent across the E2E test suite and acceptable given the direct side-effect verification.

Also applies to: 429-438

## Triage

- Decision: `invalid`
- Notes:
  - This is the same artifact-capture pattern as issue 004: `diagnostics` is expected capture metadata, not data read back from the daemon runtime.
  - The nightly test already verifies the real system outcome through network audit assertions, task/session status checks, on-disk side effects, and persisted session metadata.
  - Converting the artifact stub into a second observed-runtime assertion would duplicate adjacent checks rather than fix a correctness gap.
