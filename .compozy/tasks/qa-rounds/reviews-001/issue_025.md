---
status: resolved
file: packages/ui/src/components/mono-badge.tsx
line: 20
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:cf60b7b49f37
review_hash: cf60b7b49f37
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 025: Component docs are now slightly out of sync with supported tones.
## Review Comment

`"solid-accent"` is a non-tinted variant, while the component docs still describe tinted usage only. Consider updating the comment to prevent confusion for consumers.

## Triage

- Decision: `valid`
- Notes: `MonoBadge` now supports a `solid-accent` tone, but its component comment still describes only tinted badges. The fix is a small documentation update in the component comment so consumers understand that `solid-accent` is intentionally non-tinted.
