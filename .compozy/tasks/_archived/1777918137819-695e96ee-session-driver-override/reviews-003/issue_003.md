---
status: resolved
file: internal/session/manager_integration_test.go
line: 636
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167424608,nitpick_hash:96c1913a2861
review_hash: 96c1913a2861
source_review_id: "4167424608"
source_review_submitted_at: "2026-04-24T02:13:16Z"
---

# Issue 003: Add t.Parallel() to both independent subtests.
## Review Comment

Line 636 and Line 669 subtests are isolated and can run in parallel; adding `t.Parallel()` improves suite throughput and aligns with the test guideline.

As per coding guidelines, "Add `t.Parallel()` to independent subtests in Go tests".

## Triage

- Decision: `valid`
- Notes:
- The two subtests in `TestManagerIntegrationSyntheticQueueStateTransitions` each build their own `Manager` instance and do not share mutable state with each other.
- This matches the repo guideline to add `t.Parallel()` to independent subtests, and doing so improves throughput without changing semantics.
- Fix plan: add `t.Parallel()` inside both `t.Run(...)` bodies.
- Implemented: both scoped subtests now call `t.Parallel()`.
- Verified with targeted integration `go test -tags integration ./internal/session -run '^TestManagerIntegrationSyntheticQueueStateTransitions$'` and the full repository gate (`make verify`).
