---
status: resolved
file: internal/daemon/daemon_network_collaboration_integration_test.go
line: 856
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:4062478272cd
review_hash: 4062478272cd
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 009: Avoid fixed sleeps in waitForRuntimeCondition.
## Review Comment

Every waiter in this file now depends on a polling loop with `time.Sleep`, which adds CI flake and delays cancellation. Prefer a context-aware ticker/select or an event-driven wait primitive from the harness. As per coding guidelines, Never use `time.Sleep()` in orchestration — use proper synchronization primitives.

## Triage

- Decision: `VALID`
- Root cause: `waitForRuntimeCondition` polls with a fixed `time.Sleep`, which delays cancellation and adds unnecessary timing sensitivity to the daemon E2E lane.
- Fix plan: switch the helper to a timer+ticker loop so waiting is context-free but event-driven instead of sleep-driven.
- Resolution: replaced the fixed-sleep loop in `waitForRuntimeCondition` with a timer+ticker wait.
- Verification: `go test ./internal/daemon` passed. `go test -tags integration ./internal/api/httpapi ./internal/api/udsapi ./internal/daemon` was rerun but is blocked before these tests execute because the branch is missing `internal/testutil/acpmock/driver/dist/index.js`. `make verify` hits the same unrelated blocker in `internal/testutil/acpmock` and `internal/testutil/e2e`.
