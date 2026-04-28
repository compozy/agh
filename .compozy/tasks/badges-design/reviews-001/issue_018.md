---
status: resolved
file: web/src/systems/tasks/components/tasks-detail-preview-panel.tsx
line: 58
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:69e48d4556c5
review_hash: 69e48d4556c5
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 018: Stale JSDoc comment references removed components.
## Review Comment

The docstring still mentions `StatusDot` and `MonoBadge`, but these have been replaced by `Pill.Dot` and `Pill`. Update the comment to reflect the current implementation.

## Triage

- Decision: `valid`
- Root cause: the `TasksDetailPreviewPanel` docstring still references removed `StatusDot`/`MonoBadge` components.
- Fix approach: update the JSDoc to reference `Pill.Dot`, `Pill`, `Metric`, `Section`, and `CodeBlock`.
