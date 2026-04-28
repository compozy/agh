---
status: resolved
file: packages/ui/src/components/stories/pill-group.stories.tsx
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:ddb6a4664b0d
review_hash: ddb6a4664b0d
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 003: Minor: Clarify component description.
## Review Comment

The description says "Renamed from the legacy `PillGroup`" but the component is now named `PillGroup`. Consider updating to clarify what it was renamed *from* (e.g., "Replaces the legacy `Pills` segmented toggle").

## Triage

- Decision: `valid`
- Root cause: the Storybook description says `PillGroup` was renamed from `PillGroup`, which is self-contradictory and does not explain the legacy primitive being replaced.
- Fix approach: rewrite the description to state that `PillGroup` replaces the legacy segmented toggle/pills usage.
