---
status: resolved
file: internal/daemon/daemon_environment_sandbox_integration_test.go
line: 132
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:35cce496ec28
review_hash: 35cce496ec28
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 004: Diagnostics capture records expected values, not observed runtime data.
## Review Comment

The `diagnostics` struct is populated with expected values for artifact capture purposes. While the test correctly verifies side effects (file content at line 107), it doesn't read back runtime-emitted diagnostics to confirm the daemon actually recorded the tool-host operation. This is acceptable since the file content verification proves the behavior, but for fuller observability validation, consider reading from `harness.SessionSandboxArtifact` or a dedicated diagnostics endpoint.

## Triage

- Decision: `invalid`
- Notes:
  - The `diagnostics` value in this test is not meant to mirror runtime-emitted daemon data; it is an expected artifact payload registered for later capture.
  - The test already verifies real product behavior through the observed side effect on disk plus persisted environment/session metadata read back from the daemon.
  - Re-reading the handcrafted diagnostics struct would mostly test the artifact helper plumbing, not add new assurance about runtime behavior, so the review comment does not identify a concrete defect.
