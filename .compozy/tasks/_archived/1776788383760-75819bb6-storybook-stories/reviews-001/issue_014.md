---
status: resolved
file: web/src/components/ui/stories/scroll-area.stories.tsx
line: 4
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:7d4923303c48
review_hash: 7d4923303c48
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 014: Replace relative scroll-area import with @/* alias.
## Review Comment

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - `scroll-area.stories.tsx` still uses a sibling relative import for the component module.
  - This violates the scoped web import-path rule and is safe to correct in place.
  - Fix approach: replace it with `@/components/ui/scroll-area` and verify compilation.
