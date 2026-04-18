---
status: resolved
file: web/src/components/ui/stories/toggle-group.stories.tsx
line: 12
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:36043c052b12
review_hash: 36043c052b12
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 017: Switch this import to the @/* alias.
## Review Comment

Line 12 should use the source alias instead of a relative path.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - The toggle-group story imports `../toggle-group`, which is outside the scoped import convention for `web/src`.
  - This is another instance of the same root cause affecting the batch's Storybook files.
  - Fix approach: switch to `@/components/ui/toggle-group` and verify with typecheck/build.
