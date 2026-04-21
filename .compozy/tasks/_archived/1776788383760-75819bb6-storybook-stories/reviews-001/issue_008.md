---
status: resolved
file: web/src/components/ui/stories/empty.stories.tsx
line: 5
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:fccb003528d9
review_hash: fccb003528d9
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 008: Use @/* path alias for local UI imports.
## Review Comment

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - `empty.stories.tsx` imports local UI primitives with a relative parent path from within `web/src`.
  - Active web instructions require alias imports for these modules.
  - Fix approach: rewrite the import to `@/components/ui/empty` and verify the file still compiles and renders in Storybook.
