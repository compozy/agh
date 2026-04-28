---
status: resolved
file: packages/ui/src/components/stories/pill.stories.tsx
line: 11
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:2450eb2c3284
review_hash: 2450eb2c3284
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 004: Typo in component description.
## Review Comment

Line 14 reads "replaces `Pill`, `Pill`, `KindChip`..." with duplicate "Pill" entries. This should list the distinct components being replaced (e.g., `MonoBadge`, `StatusDot`, `KindChip`, etc.).

## Triage

- Decision: `valid`
- Root cause: the `Pill` Storybook component description repeats `Pill` several times and names removed/legacy primitives unclearly.
- Fix approach: update the description to list distinct replacements (`MonoBadge`, `StatusDot`, `KindChip`, `WireChip`, connection indicator composition) and keep the story copy aligned with the current `Pill` + `Pill.Dot` API.
