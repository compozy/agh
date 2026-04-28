---
status: resolved
file: web/src/storybook/packages-ui-storybook-config.test.ts
line: 31
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:fcf6b8ded5b7
review_hash: fcf6b8ded5b7
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 009: Consider asserting shadcn CSS import too.
## Review Comment

Since Storybook preview styling now depends on both shared tokens and shadcn Tailwind styles, adding an assertion for the shadcn import would strengthen regression coverage.

## Triage

- Decision: `valid`
- Root cause: the Storybook config test asserts the shared token import but does not assert the `shadcn/tailwind.css` import even though preview styling depends on both.
- Fix approach: add an explicit assertion for `@import "shadcn/tailwind.css";` in `packages/ui/.storybook/preview.css`.
