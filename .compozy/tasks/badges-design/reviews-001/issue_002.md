---
status: resolved
file: packages/ui/src/components/pill.test.tsx
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4190677522,nitpick_hash:955795c2492d
review_hash: 955795c2492d
source_review_id: "4190677522"
source_review_submitted_at: "2026-04-28T16:30:00Z"
---

# Issue 002: Extract inline object-shape types to named interfaces.
## Review Comment

Lines 12–15 define `WithMotion` props as an inline object shape `{ reducedMotion: "always" | "never"; children: ReactNode; }`, and line 31 defines `it.each<{ tone: PillTone; bg: string; text: string }>` as an inline generic argument. Per coding guidelines (`**/*.{ts,tsx}: Prefer interface for defining object shapes in TypeScript`), extract both to named `interface` declarations above their respective scopes.

## Triage

- Decision: `valid`
- Root cause: `WithMotion` props and the `it.each` row object are inline object shapes, which weakens readability and violates the repository preference for named interfaces for object shapes.
- Fix approach: introduce named interfaces for the motion wrapper props and tone expectation table rows, then reuse them in the test.
