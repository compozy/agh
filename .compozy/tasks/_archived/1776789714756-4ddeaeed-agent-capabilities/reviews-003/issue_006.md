---
status: resolved
file: internal/network/manager_integration_test.go
line: 16
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135966430,nitpick_hash:a686244662c5
review_hash: a686244662c5
source_review_id: "4135966430"
source_review_submitted_at: "2026-04-19T12:48:57Z"
---

# Issue 006: Add t.Parallel() for independent integration test.
## Review Comment

Per coding guidelines, add `t.Parallel()` to independent tests. This integration test is self-contained and can run concurrently with other tests.

## Triage

- Decision: `valid`
- Root cause: `TestManagerJoinPublishesProjectedCapabilityBriefInInitialAndReconnectGreets` is an isolated integration test but does not call `t.Parallel()`. The test constructs its own manager with `testManagerConfig()` (`Port: -1`), uses a fresh `t.TempDir()` audit path, and tears down its own subscription and manager state, so it unnecessarily serializes against the rest of the integration suite.
- Impact: the integration suite runs more slowly than necessary and this test diverges from the package's established pattern, where comparable network integration tests already opt into parallel execution.
- Resolution: `internal/network/manager_integration_test.go` now calls `t.Parallel()` as the first statement in `TestManagerJoinPublishesProjectedCapabilityBriefInInitialAndReconnectGreets`, matching the package's other isolated integration tests without altering the test's behavior or setup.
- Verification:
  - `go test ./internal/network -tags integration -run TestManagerJoinPublishesProjectedCapabilityBriefInInitialAndReconnectGreets -count=1`
  - `make verify`
