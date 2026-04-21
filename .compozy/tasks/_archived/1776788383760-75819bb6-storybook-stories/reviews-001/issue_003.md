---
status: resolved
file: web/src/components/ui/stories/collapsible.stories.tsx
line: 5
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:9bba9a398d66
review_hash: 9bba9a398d66
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 003: Use @/* alias import for collapsible components.
## Review Comment

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports."

## Triage

- Decision: `VALID`
- Notes:
  - The story imports `../collapsible`, but the web import contract for `web/src` requires alias-based imports.
  - This is a real policy violation in a file already in scope for the batch.
  - Fix approach: switch the import to `@/components/ui/collapsible` and verify with the normal web checks.
