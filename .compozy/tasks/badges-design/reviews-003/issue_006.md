---
status: pending
file: internal/daemon/boot.go
line: 521
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:59d120de52a6
review_hash: 59d120de52a6
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 006: Add startup latency guardrails around boot-time session repair.
## Review Comment

Operationally, this loop can grow startup time with many stopped sessions. Consider recording duration/count metrics and optionally capping repairs per boot cycle.

## Triage

- Decision: `UNREVIEWED`
- Notes:
