---
status: resolved
file: internal/e2elane/command_wiring_test.go
line: 32
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4129384275,nitpick_hash:fedb3b76784c
review_hash: fedb3b76784c
source_review_id: "4129384275"
source_review_submitted_at: "2026-04-17T13:54:50Z"
---

# Issue 017: Consider adding t.Parallel() to subtests.
## Review Comment

The subtests test independent make targets and could run concurrently to reduce test execution time.

As per coding guidelines: "Use t.Parallel() for independent subtests in Go tests"

## Triage

- Decision: `VALID`
- Root cause: the make-target subtests are read-only command invocations over independent targets, so they can safely run in parallel.
- Fix plan: add `t.Parallel()` inside the target subtests.
- Resolution: added `t.Parallel()` to the make-target subtests and to the independent top-level command-wiring tests.
- Verification: `go test ./internal/e2elane` passed. Historical note: the earlier `driver/dist/index.js` blocker was stale; the shipped mock driver is `internal/testutil/acpmock/cmd/acpmock-driver`.
