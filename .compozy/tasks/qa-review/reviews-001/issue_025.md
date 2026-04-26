---
status: resolved
file: web/src/systems/tasks/lib/task-editor.ts
line: 134
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4176489704,nitpick_hash:b26c0ae72079
review_hash: b26c0ae72079
source_review_id: "4176489704"
source_review_submitted_at: "2026-04-26T03:49:14Z"
---

# Issue 025: Extract shared base payload mapping to avoid builder drift.
## Review Comment

`buildCreateChildTaskRequest` now mirrors `buildCreateTaskRequest` field-for-field. Consider moving shared mapping into a single helper so future field changes remain consistent across both paths.

## Triage

- Decision: `invalid`
- Notes:
  - `buildCreateTaskRequest()` and `buildCreateChildTaskRequest()` currently produce equivalent field mappings by design, but the review comment identifies a speculative future-drift risk rather than a present behavioral bug or rule violation.
  - The current tests already pin the create-task and create-child-task payload shapes independently, and the requested extraction would be a proactive refactor unrelated to any broken behavior in this batch.
  - Because this remediation run is constrained to concrete review defects and regressions, I am not widening scope for a no-op deduplication refactor.
