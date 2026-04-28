---
status: pending
file: web/src/systems/knowledge/lib/knowledge-formatters.ts
line: 1
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4191605807,nitpick_hash:166fa7176ddc
review_hash: 166fa7176ddc
source_review_id: "4191605807"
source_review_submitted_at: "2026-04-28T18:57:12Z"
---

# Issue 012: Keep @agh/ui tone types out of the knowledge lib layer.
## Review Comment

`knowledge-formatters.ts` now depends on the UI package just to describe tone strings. That leaks presentation concerns into `web/src/systems/knowledge/lib` and makes the formatter harder to reuse outside the component layer. A local semantic union mapped to `PillTone` at render time would keep the boundary cleaner.

As per coding guidelines, `web/src/systems/**/*.{ts,tsx}`: Dependency flow within systems: `adapters → lib → hooks → components` (unidirectional, never reversed).

Also applies to: 39-50

## Triage

- Decision: `UNREVIEWED`
- Notes:
