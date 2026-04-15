---
status: resolved
file: web/src/systems/session/components/tool-renderers/edit-content.tsx
line: 7
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4113957510,nitpick_hash:3e37c07dec59
review_hash: 3e37c07dec59
source_review_id: "4113957510"
source_review_submitted_at: "2026-04-15T13:28:12Z"
---

# Issue 006: Consider extracting shared truncation UI logic used by both tool renderers.

## Review Comment

`edit-content.tsx` and `write-content.tsx` now carry near-identical expand/truncate behavior; a shared helper/component will reduce drift and keep thresholds/button behavior aligned.

Also applies to: 21-23, 28-29

---

## Triage

- Decision: `invalid`
- Reasoning: `edit-content.tsx` and `write-content.tsx` do duplicate some truncation UI, but that duplication is not causing an observable defect in the current batch. Extracting a shared renderer here would be a cleanup refactor beyond the correctness issues under review, and the batch instructions explicitly say not to refactor unrelated code.
