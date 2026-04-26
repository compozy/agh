---
status: resolved
file: web/src/systems/network/components/stories/network-workspace-shell.stories.tsx
line: 56
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177411700,nitpick_hash:9ca4d6bee709
review_hash: 9ca4d6bee709
source_review_id: "4177411700"
source_review_submitted_at: "2026-04-26T20:24:27Z"
---

# Issue 031: Unconventional use of globalThis.Error.
## Review Comment

Using `globalThis.Error` instead of just `Error` is unusual. While it works, `Error` is globally available and more idiomatic. This may have been intentional to avoid linting rules, but worth noting for consistency.

## Triage

- Decision: `valid`
- Notes: `globalThis.Error` is unnecessary in the Storybook fixture helper because `Error` is already a global value in this TypeScript runtime. The fix is to use idiomatic `Error` directly.
