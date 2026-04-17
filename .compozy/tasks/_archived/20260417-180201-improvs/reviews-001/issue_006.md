---
status: resolved
file: internal/environment/daytona/perf_bench_test.go
line: 67
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:82ba9c13c8f6
review_hash: 82ba9c13c8f6
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 006: Clean up extracted trees between benchmark iterations.
## Review Comment

This loop keeps every extracted workspace under `baseDest`. Long runs can skew results or exhaust temp storage in CI. Remove `dest` after each iteration with the timer stopped.

## Triage

- Decision: `VALID`
- Notes:
  `BenchmarkExtractTarWorkspaceTree` allocates a new destination tree on every
  iteration and never removes it, so long benchmark runs accumulate extracted
  workspaces and distort both IO cost and disk usage. Plan: remove each
  extracted tree with the timer stopped after the iteration completes.
