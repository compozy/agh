---
provider: coderabbit
pr: "120"
round: 2
round_created_at: 2026-05-07T19:41:55.305082Z
status: resolved
file: internal/api/core/perf_bench_test.go
line: 140
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4247165327,nitpick_hash:4fb18842814f
review_hash: 4fb18842814f
source_review_id: "4247165327"
source_review_submitted_at: "2026-05-07T19:37:05Z"
---

# Issue 002: BenchmarkPromptStreamEncoderEmit: encoder construction is inside b.Loop(), folding its allocations into every per-iteration measurement.
## Review Comment

`NewPromptStreamEncoder(now)` is called at line 194 inside the `b.Loop()` body, so `b.ReportAllocs()` will attribute the encoder's construction allocations to each "emit" iteration. The benchmark name implies the intent is to measure the `Emit` path. One of the explicit advantages of `testing.B.Loop` is that it automatically excludes setup and cleanup code from benchmark timing — constructing the encoder before `b.Loop()` would leverage that property and isolate only the `Emit` path.

## Triage

- Decision: `valid`
- Notes:
  - `BenchmarkPromptStreamEncoderEmit` constructs a fresh encoder inside the measured loop, so constructor/setup allocations are currently folded into the benchmark's per-iteration numbers.
  - The benchmark name implies the intent is to isolate the `Emit` path, so the timing/allocation window should exclude encoder construction while still using a fresh encoder per iteration.
  - Fix plan: move encoder construction outside the measured section for each iteration and keep the `Emit` sequence as the only timed work.
  - Resolved: the benchmark now excludes per-iteration encoder construction from the measured section while preserving a fresh encoder for each benchmark iteration.
