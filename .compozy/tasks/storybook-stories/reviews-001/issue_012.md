---
status: resolved
file: web/src/components/ui/stories/native-select.stories.tsx
line: 3
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:f459d1660180
review_hash: f459d1660180
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 012: Switch story imports to the @/* alias.
## Review Comment

These relative imports should use the web alias for consistency with the app import policy.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - `native-select.stories.tsx` uses relative imports for both `field` and `native-select`, which conflicts with the scoped import policy.
  - This is the same underlying defect as the other web story files in this batch.
  - Fix approach: convert those imports to `@/components/ui/field` and `@/components/ui/native-select`.
