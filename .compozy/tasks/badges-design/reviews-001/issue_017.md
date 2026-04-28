---
status: resolved
file: web/src/systems/tasks/components/tasks-detail-header.tsx
line: 42
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:1da0698660cd
review_hash: 1da0698660cd
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 017: Stale JSDoc comment references MonoBadge.
## Review Comment

The docstring mentions `MonoBadge` which has been replaced by `Pill`. Consider updating to match the current implementation.

## Triage

- Decision: `valid`
- Root cause: the `TasksDetailHeader` docstring still mentions the old `MonoBadge` primitive even though the implementation now uses `Pill`.
- Fix approach: update the JSDoc to describe the current `Pill`/`Pill.Dot` composition.
