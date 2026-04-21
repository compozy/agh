---
status: resolved
file: web/src/systems/bridges/components/stories/bridge-edit-dialog.stories.tsx
line: 7
severity: major
author: coderabbitai[bot]
provider_ref: review:4133289577,nitpick_hash:caeaf8050420
review_hash: caeaf8050420
source_review_id: "4133289577"
source_review_submitted_at: "2026-04-18T02:22:01Z"
---

# Issue 020: Use aliased import path for local component
## Review Comment

Line 7 uses `../bridge-edit-dialog`; switch to the `@/*` alias.

As per coding guidelines "Use path alias `@/*` to map to `./src/*` for all imports".

## Triage

- Decision: `VALID`
- Notes:
  - The story imports `../bridge-edit-dialog` through a relative path even though scoped web files should use `@/*` aliases.
  - The file already uses aliased imports for other bridge modules, so this lone relative import is inconsistent with the established local pattern.
  - Fix approach: change the component import to `@/systems/bridges/components/bridge-edit-dialog` and verify via the normal web pipeline.
