---
status: resolved
file: web/src/systems/network/components/network-peers-list-panel.tsx
line: 22
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:c4a038022332
review_hash: c4a038022332
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 024: Move PeerListItem props into an interface.
## Review Comment

The inline object type here breaks the repository’s TypeScript shape convention. Defining a `PeerListItemProps` interface would keep this aligned with the rest of the `web/` codebase.

As per coding guidelines, "Use `interface` for defining object shapes in TypeScript (pattern is in Zod schemas and types)".

---

## Triage

- Decision: `valid`
- Root cause: `PeerListItem` uses an inline object-shape type for props inside a file that otherwise follows named-interface conventions. This is a style-consistency issue rather than a behavior bug, but it matches the project’s TypeScript shape rule.
- Fix approach: extract the props shape into a named `PeerListItemProps` interface without changing behavior.
- Resolution: extracted `PeerListItemProps` and kept the component behavior unchanged.
- Verification: `make web-lint`, `make web-typecheck`, and `make verify` passed after the refactor.
