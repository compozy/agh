---
status: resolved
file: web/src/components/ui/stories/input-group.stories.tsx
line: 5
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:7f06c58dadcf
review_hash: 7f06c58dadcf
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 010: Use @/* imports instead of relative paths in this story module.
## Review Comment

Lines 5–13 use `../...` imports. Please switch these to the project alias to match the web import contract.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - This story mixes multiple `../...` imports for `field` and `input-group`, which conflicts with the `@/*` import contract.
  - The root cause is the same as the other UI stories in scope: relative imports were used during story authoring.
  - Fix approach: switch all local component imports in this module to `@/components/ui/...`.
