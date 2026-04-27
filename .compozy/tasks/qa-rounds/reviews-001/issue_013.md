---
status: resolved
file: web/src/systems/session/contexts/session-create-context.tsx
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4177569108,nitpick_hash:f97e9ac4418e
review_hash: f97e9ac4418e
source_review_id: "4177569108"
source_review_submitted_at: "2026-04-26T22:35:58Z"
---

# Issue 013: Extract provider props into an interface.
## Review Comment

The provider is fine functionally, but props should be modeled with a named interface instead of an inline object shape.

As per coding guidelines, `**/*.{ts,tsx}`: Use `interface` for defining object shapes in TypeScript instead of `type` declarations.

## Triage

- Decision: `VALID`
- Notes:
  - `SessionCreateProvider` declares its props as an inline object shape and uses `React.ReactNode`.
  - The local TypeScript convention prefers named interfaces for object props and explicit type imports.
  - Fix by adding a `SessionCreateProviderProps` interface and importing `ReactNode` as a type.
  - Resolution: added `SessionCreateProviderProps` and an explicit `ReactNode` type import.
  - Verification: `make web-typecheck` passed; `make verify` passed.
