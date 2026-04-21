---
status: resolved
file: web/src/components/ui/stories/dropdown-menu.stories.tsx
line: 16
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:ac226491e6fb
review_hash: ac226491e6fb
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 007: Use @/* alias instead of relative import
## Review Comment

Line 16 uses a relative import (`../dropdown-menu`). Please switch to the project alias for consistency and guideline compliance.

As per coding guidelines "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - The file uses `../dropdown-menu` in `web/src`, which conflicts with the scoped web import contract.
  - This is a real consistency defect in a touched story file, not an external preference.
  - Fix approach: replace the relative import with `@/components/ui/dropdown-menu`.
