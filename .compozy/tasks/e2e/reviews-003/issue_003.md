---
status: resolved
file: internal/daemon/daemon_bridge_extension_integration_test.go
line: 28
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4130744050,nitpick_hash:dedcb0144884
review_hash: dedcb0144884
source_review_id: "4130744050"
source_review_submitted_at: "2026-04-17T17:17:39Z"
---

# Issue 003: Consider adding t.Parallel() if the test can run concurrently.
## Review Comment

This comprehensive bridge ingress E2E test validates the full Telegram extension lifecycle but lacks `t.Parallel()`. Given it builds the extension binary and runs a daemon, this may be intentional due to resource constraints or test isolation requirements.

If concurrent execution is feasible, adding `t.Parallel()` would improve test suite throughput.

## Triage

- Decision: `invalid`
- Notes:
  - This integration test builds the Telegram reference extension, launches the daemon runtime, and coordinates multiple subprocesses plus filesystem markers.
  - The prior bridge-extension E2E work explicitly removed a package-parallel flake around the extension binary output. Adding `t.Parallel()` here would reintroduce cross-test contention without evidence that the runtime lane is isolated enough.
  - No correctness bug is present in the current test; the suggestion is an optional throughput tweak and is not safe to apply blindly.
