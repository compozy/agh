---
status: resolved
file: packages/site/components/logos/claude.tsx
line: 13
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4123387028,nitpick_hash:fb6050d65e93
review_hash: fb6050d65e93
source_review_id: "4123387028"
source_review_submitted_at: "2026-04-16T18:17:26Z"
---

# Issue 023: Remove redundant empty class token in cn().
## Review Comment

Line 13 can be simplified; `cn("", className)` is equivalent to `cn(className)`.

## Triage

- Decision: `valid`
- Notes:
  - `ClaudeLogo` currently calls `cn("", className)`, and the empty string token contributes no class output or conditional behavior.
  - Root cause: a redundant placeholder argument was left in the helper call.
  - Fix plan: simplify the component to `cn(className)`. No dedicated regression test is needed because this is a behavior-preserving cleanup that will still be covered by the site test/typecheck/build gates.
  - Resolution: simplified `ClaudeLogo` to `cn(className)` and added a focused `ClaudeLogo` className regression assertion in `packages/site/components/logos/logos.test.tsx`.
  - Verification: `bun run test -- components/landing/__tests__/landing.test.tsx components/logos/logos.test.tsx`, `bun run typecheck`, and `make verify` all passed.
