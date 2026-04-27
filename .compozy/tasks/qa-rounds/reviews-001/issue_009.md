---
status: resolved
file: web/src/systems/agent/components/stories/agent-info-panel.stories.tsx
line: 36
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:9d984974c582
review_hash: 9d984974c582
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 009: Prefer an interface for Frame props.
## Review Comment

Use a named interface for the component props instead of an inline object shape in the function signature.

As per coding guidelines, `**/*.{ts,tsx}`: Use `interface` for defining object shapes in TypeScript instead of `type` declarations.

## Triage

- Decision: `VALID`
- Notes:
  - `Frame` declares its props inline and uses `React.ReactNode` without a local React type import.
  - The local TypeScript style requires named interfaces for object shapes, and React 19 automatic JSX runtime still requires explicit React type imports for `ReactNode`.
  - Fix by importing `ReactNode` as a type and replacing the inline object shape with a named `FrameProps` interface.
  - Resolution: added an explicit `ReactNode` type import and a named `FrameProps` interface.
  - Verification: `make web-typecheck` passed; `make verify` passed.
