---
status: resolved
file: web/src/systems/session/components/tool-renderers/write-content.tsx
line: 38
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4113957510,nitpick_hash:3ee4f6a388bc
review_hash: 3ee4f6a388bc
source_review_id: "4113957510"
source_review_submitted_at: "2026-04-15T13:28:12Z"
---

# Issue 008: Consider making this a true expand/collapse toggle.

## Review Comment

The control only expands once (`setShowFull(true)`), so users can’t collapse back without rerendering the row.

## Triage

- Decision: `invalid`
- Reasoning: Same as issue 007. The existing write renderer exposes a one-way expansion affordance, which is consistent with its current label and does not create incorrect output; adding collapse support would be product polish, not a required bug fix for this batch.
