---
status: resolved
file: web/src/components/ui/stories/select.stories.tsx
line: 3
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:09893be0fdab
review_hash: 09893be0fdab
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 015: Convert relative UI imports to the @/* alias.
## Review Comment

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - The select story uses `../field` and `../select` imports in `web/src`, which breaks the alias contract.
  - This is a real consistency issue across the newly added Storybook files.
  - Fix approach: use `@/components/ui/field` and `@/components/ui/select` instead.
