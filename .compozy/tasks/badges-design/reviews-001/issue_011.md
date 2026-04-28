---
status: resolved
file: web/src/systems/bridges/components/bridge-detail-panel.tsx
line: 87
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:818b4723d1c8
review_hash: 818b4723d1c8
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 011: Consolidate duplicate status→tone mapping.
## Review Comment

`statusToStatusDotTone` and `statusToMonoBadgeTone` currently do the same work. Keep a single mapper to avoid drift.

## Triage

- Decision: `valid`
- Root cause: `statusToStatusDotTone` and `statusToMonoBadgeTone` are identical mappings in `bridge-detail-panel.tsx`, creating drift risk for the same bridge status signal.
- Fix approach: consolidate them into one status-to-pill-tone mapper and use the same tone for both the dot and badge.
