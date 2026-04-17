---
status: resolved
file: internal/bridgesdk/perf_bench_test.go
line: 66
severity: major
author: coderabbitai[bot]
provider_ref: review:4130502052,nitpick_hash:49e08c3fd6d0
review_hash: 49e08c3fd6d0
source_review_id: "4130502052"
source_review_submitted_at: "2026-04-17T16:38:53Z"
---

# Issue 003: Handle ignored Close/Serve errors in the peer benchmark.
## Review Comment

Several operational errors are discarded with `_`, which can mask harness failures and make benchmark outcomes misleading.

As per coding guidelines, "Never ignore errors with `_` — every error must be handled or have a written justification."

Also applies to: 87-94, 112-116

## Triage

- Decision: `VALID`
- Notes:
  The benchmark currently discards `Serve` and `Close` errors with `_`, which
  can hide transport teardown failures and leave misleading benchmark results.
  Plan: capture serve results, check close errors explicitly, and ignore only
  the expected shutdown conditions after the benchmark loop ends.
