---
status: resolved
file: internal/daemon/restart_integration_test.go
line: 19
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133264038,nitpick_hash:f0b8fb4b9c97
review_hash: f0b8fb4b9c97
source_review_id: "4133264038"
source_review_submitted_at: "2026-04-18T02:14:16Z"
---

# Issue 018: Consider adding t.Parallel() to independent integration tests.
## Review Comment

Per coding guidelines, independent tests should use `t.Parallel()`. These three integration tests appear to be independent (each creates its own `homePaths` via `integrationHomePaths(t)` with isolated temp directories) and could run in parallel.

---

## Triage

- Decision: `invalid`
- Notes:
  These integration tests call `integrationHomePaths(t)`, which uses `t.Setenv("AGH_HOME", ...)` and `t.Setenv("HOME", ...)`. Go forbids `t.Setenv` in parallel tests because environment mutation is process-global, so adding `t.Parallel()` here would introduce a real test-safety problem rather than an improvement.
