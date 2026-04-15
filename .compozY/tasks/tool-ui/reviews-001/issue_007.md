---
status: resolved
file: web/src/systems/session/components/tool-renderers/edit-content.tsx
line: 57
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4113957510,nitpick_hash:fd011a384e57
review_hash: fd011a384e57
source_review_id: "4113957510"
source_review_submitted_at: "2026-04-15T13:28:12Z"
---

# Issue 007: Optional UX polish: make “Show full content” reversible.

## Review Comment

This currently expands only once; a toggle gives users better control when comparing large diffs.

## Triage

- Decision: `invalid`
- Reasoning: This is an optional UX enhancement rather than a defect. The current control truthfully performs a one-way “show full content” expansion, and the lack of collapse support does not break the renderer or violate the existing contract.
