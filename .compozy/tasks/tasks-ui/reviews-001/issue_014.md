---
status: resolved
file: internal/daemon/automation_task_e2e_assertions_test.go
line: 84
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4133273307,nitpick_hash:13615e94caba
review_hash: 13615e94caba
source_review_id: "4133273307"
source_review_submitted_at: "2026-04-18T02:17:09Z"
---

# Issue 014: Add a nil guard in findTaskRunInDetail to avoid panic-prone helper behavior.
## Review Comment

Since the helper now accepts a pointer, a nil input will panic on `detail.Runs`. Returning `false` is safer for test diagnostics.

## Triage

- Decision: `valid`
- Notes: Confirmed. `findTaskRunInDetail` dereferences `detail.Runs` without guarding `detail == nil`, so a nil caller would panic instead of producing a clean missing-run result. I’ll add the nil guard and a regression assertion.
