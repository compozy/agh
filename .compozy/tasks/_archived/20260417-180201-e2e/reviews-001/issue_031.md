---
status: resolved
file: internal/testutil/e2e/runtime_harness_integration_test.go
line: 36
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:8267b84050a9
review_hash: 8267b84050a9
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 031: Add t.Parallel() to integration test functions.
## Review Comment

The main test functions (`TestStartRuntimeHarnessBootsRealDaemonAndExposesClients`, `TestStartRuntimeHarnessResolvesSeededWorkspaceThroughPublicSurface`, `TestStartRuntimeHarnessCapturesTranscriptAndEventsArtifacts`) are missing `t.Parallel()` calls.

## Triage

- Decision: `valid`
- Notes:
  The three integration tests boot isolated harness instances with unique temp
  directories and do not rely on `t.Setenv` or other shared mutable state. They
  should opt into `t.Parallel()` so the integration suite matches the repo's
  test parallelism rule.

## Resolution

- Added `t.Parallel()` to the three isolated runtime-harness integration tests.
