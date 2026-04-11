---
status: resolved
file: internal/extension/host_api_channels.go
line: 223
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093721845,nitpick_hash:5b2634d1bf4b
review_hash: 5b2634d1bf4b
source_review_id: "4093721845"
source_review_submitted_at: "2026-04-11T12:28:05Z"
---

# Issue 032: Consider logging cleanup failures or returning early on non-critical path.
## Review Comment

The `maybeCleanupChannelIngestDedup` function returns an error on cleanup failure, which would fail the entire ingest operation. Since this is a periodic maintenance task, consider whether cleanup failures should be logged but not block message ingestion.

## Triage

- Decision: `invalid`
- Why: Expired dedup cleanup is part of the same persistence path that enforces idempotent inbound ingest. If cleanup keeps failing after the existing SQLite-busy retries, the dedup store is unhealthy and continuing ingest would hide a real storage fault in the critical path.
- Why not fix: Downgrading the error to logging-only would trade correctness for availability and could leave ingest running against a broken dedup store. The current fail-fast behavior is intentional for this contract.
- Resolution: Analysis complete; current fail-fast behavior is intentional.
