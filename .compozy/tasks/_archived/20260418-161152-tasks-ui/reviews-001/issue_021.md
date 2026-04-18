---
status: resolved
file: internal/observe/tasks_integration_test.go
line: 260
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:5085ce79b3ea
review_hash: 5085ce79b3ea
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 021: Restructure these integration cases into Should... subtests.
## Review Comment

The new scenarios are valuable, but each test is still a large linear flow. Splitting the state transitions/assertions into `t.Run("Should...")` cases would make regressions easier to pinpoint and keep the integration suite consistent with the repo standard.

As per coding guidelines, `Use table-driven tests with subtests (t.Run) as default in Go tests` and `MUST use t.Run("Should...") pattern for ALL test cases`.

## Triage

- Decision: `valid`
- Root cause: the new integration coverage was added as long linear test functions instead of repo-standard `t.Run("Should...")` subtests, which makes failures coarser and diverges from the workspace testing rule.
- Fix approach: split the affected observer integration scenarios into descriptive `Should...` subtests while preserving the current end-to-end assertions and lifecycle coverage.

## Resolution

- Wrapped the three observer integration scenarios in `t.Run("Should ...")` subtests in `internal/observe/tasks_integration_test.go`.
- Preserved the lifecycle setup and assertions; the change is structural only and improves failure locality.
- Verification: `go test ./internal/observe` and `go test -tags integration ./internal/observe`
