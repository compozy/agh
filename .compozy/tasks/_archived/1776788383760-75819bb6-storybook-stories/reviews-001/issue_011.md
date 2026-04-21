---
status: resolved
file: web/src/components/ui/stories/item.stories.tsx
line: 15
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:7be470f4a0c5
review_hash: 7be470f4a0c5
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 011: Use @/* alias for internal imports (Line 15).
## Review Comment

`../item` breaks the web import-path rule; switch to the `@/*` alias to keep imports stable during refactors.

As per coding guidelines, "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - The item story imports the local component module with `../item`, which violates the active alias policy for `web/src`.
  - This is a concrete consistency issue in a scoped file.
  - Fix approach: change the import to `@/components/ui/item` and validate through typecheck/build.
