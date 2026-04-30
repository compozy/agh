---
provider: coderabbit
pr: "85"
round: 2
round_created_at: 2026-04-30T17:03:05.076488Z
status: pending
file: internal/core/run/executor/execution_acp_test.go
line: 631
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4093999324,nitpick_hash:60eee021b85e
review_hash: 60eee021b85e
source_review_id: "4093999324"
source_review_submitted_at: "2026-04-11T17:04:03Z"
---

# Issue 001: Good test coverage for finalizeUIOnCompletion behavior.
## Review Comment

The tests comprehensively cover:
1. `CloseOnComplete=false` - verifies only `Wait` is called
2. `CloseOnComplete=true` - verifies `CloseEvents`, `Shutdown`, and `Wait` are all called
3. `nil` UI - verifies graceful handling without panics

Consider adding `t.Parallel()` to these tests for consistency with other tests in the file that use it.

## Triage

- Decision: `UNREVIEWED`
- Notes:
