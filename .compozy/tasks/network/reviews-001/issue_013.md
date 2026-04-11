---
status: resolved
file: internal/network/audit.go
line: 140
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093722580,nitpick_hash:9441b0e35775
review_hash: 9441b0e35775
source_review_id: "4093722580"
source_review_submitted_at: "2026-04-11T12:29:15Z"
---

# Issue 013: Consider keeping the file handle open for high-frequency audit logging.
## Review Comment

The current implementation opens and closes the file for every audit entry. While this ensures durability (each write is synced), it introduces significant I/O overhead if audit events are frequent.

If audit volume is expected to be high, consider:
1. Keeping the file handle open and syncing periodically
2. Using buffered writes with periodic flushes
3. Documenting the durability-over-performance trade-off

This is acceptable if audit frequency is low or if maximum durability is required.

## Triage

- Decision: `invalid`
- Root cause analysis: this comment is a speculative performance optimization, not a correctness defect. The current implementation deliberately opens, writes, and syncs per record to maximize durability and avoid extra lifecycle/shutdown complexity in the audit sink.
- Resolution plan: no code change. Keeping the existing crash-safe semantics is the better fit for the current scope unless profiling shows audit I/O is a real bottleneck.
