---
status: resolved
file: web/src/systems/session/lib/tool-labels.ts
line: 28
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4113957510,nitpick_hash:2a13f4f8093b
review_hash: 2a13f4f8093b
source_review_id: "4113957510"
source_review_submitted_at: "2026-04-15T13:28:12Z"
---

# Issue 009: Note: getToolTone and toolToneClass are not exported from the session system's public API.

## Review Comment

Per the context snippet from `index.ts`, these functions are internal-only. If other systems need tone-based styling in the future, consider adding them to the public barrel export.

Also applies to: 36-47

## Triage

- Decision: `invalid`
- Reasoning: This comment is about a hypothetical future public API need. `getToolTone` and `toolToneClass` are only consumed inside the session system today, so not exporting them from the public barrel is intentional and not a defect.
