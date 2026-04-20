---
status: resolved
file: packages/ui/src/components/command.tsx
line: 57
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4135497854,nitpick_hash:ef323f3fa718
review_hash: ef323f3fa718
source_review_id: "4135497854"
source_review_submitted_at: "2026-04-19T05:12:06Z"
---

# Issue 013: Hide decorative search icon from assistive technologies.
## Review Comment

Line 61 renders a visual-only icon; add `aria-hidden` so it isn’t announced redundantly.

## Triage

- Decision: `valid`
- Reasoning: The search icon in `CommandInput` is decorative; announcing it adds redundant noise without contributing meaning.
- Root cause: The icon is rendered without `aria-hidden`.
- Fix plan: Mark the decorative icon as hidden from assistive technologies.

## Resolution

- Marked the `CommandInput` search icon `aria-hidden="true"` in `packages/ui/src/components/command.tsx`.
- Added a regression in `packages/ui/src/components/command.test.tsx` because the existing companion test file lives outside the initial code-file list, and verified with `make verify`.
