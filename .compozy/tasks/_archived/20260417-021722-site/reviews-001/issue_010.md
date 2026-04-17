---
status: resolved
file: packages/site/app/(home)/layout.tsx
line: 10
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:d6bbe201b17d
review_hash: d6bbe201b17d
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 010: Remove redundant slot spread to reduce coupling.
## Review Comment

`Line 11` spreads `baseOptions.slots`, but the shared options currently don’t define custom slots. Keeping only the explicit `header` override is clearer.

## Triage

- Decision: `invalid`
- Notes:
  - `...baseOptions.slots` is a deliberate forward-compatible inheritance point for shared layout slots.
  - With `baseOptions.slots` currently absent, the spread is a no-op and does not create a runtime bug or coupling problem.
  - Removing it would trade extensibility for a purely stylistic cleanup, so this batch leaves it unchanged.
