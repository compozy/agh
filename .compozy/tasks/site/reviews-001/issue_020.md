---
status: resolved
file: packages/site/components/landing/primitives/mono-badge.tsx
line: 18
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:5ab47c3bc7a6
review_hash: 5ab47c3bc7a6
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 020: Keep the component doc comment in sync with the rendered size.
## Review Comment

The comment says “11px” while the class on Line 23 is `text-[10px]`. Consider aligning one of them to avoid drift.

## Triage

- Decision: `invalid`
- Notes:
  - This is a documentation-comment drift note only; it does not affect runtime behavior, accessibility, or test correctness.
  - The batch is scoped to functional/accessibility regressions first, so I am not making a code-only change to satisfy a stale inline comment.
  - No production or test defect was found here.
