---
provider: coderabbit
pr: "105"
round: 4
round_created_at: 2026-05-06T00:08:12.899766Z
status: resolved
file: internal/network/perf_bench_test.go
line: 12
severity: minor
author: coderabbitai[bot]
provider_ref: review:4232273319,nitpick_hash:f70aa228bcbd
review_hash: f70aa228bcbd
source_review_id: "4232273319"
source_review_submitted_at: "2026-05-05T23:45:49Z"
---

# Issue 029: This benchmark no longer constructs a direct message.
## Review Comment

`BenchmarkFormatNetworkMessageDirect` now sets `KindSay`, but it does not mark the envelope as a direct surface or provide a `DirectID`. That makes it easy for the benchmark to drift into measuring the generic say formatter instead of the direct-message path its name implies. Either add the direct routing metadata here or rename the benchmark.

## Triage

- Decision: `VALID`
- Root cause: `BenchmarkFormatNetworkMessageDirect` benchmarks a `KindSay` envelope but does not mark it as a direct-surface conversation or attach a `direct_id`. That means the benchmark name and the exercised formatter path can drift apart.
- Fix approach: build the benchmark envelope as an actual direct message by setting `surface=direct` and a valid `direct_id`, while keeping the rest of the benchmark payload stable.
- Verification: fixed in scoped code and validated with fresh `make verify`.
