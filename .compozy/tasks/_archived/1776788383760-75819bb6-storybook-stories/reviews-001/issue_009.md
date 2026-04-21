---
status: resolved
file: web/src/components/ui/stories/field.stories.tsx
line: 12
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:1e4f4b29c577
review_hash: 1e4f4b29c577
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 009: Switch to alias import for Field primitives
## Review Comment

Line 12 should use the `@/*` alias instead of a relative path.

As per coding guidelines "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - The field story pulls primitives from `../field` even though the file lives under `web/src`.
  - That breaks the import-path rule in scope for web files.
  - Fix approach: replace the relative import with `@/components/ui/field` and verify with the existing pipeline.
