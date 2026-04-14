---
status: resolved
file: web/src/systems/network/components/network-channels-list-panel.tsx
line: 17
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:e8b0bf9179d8
review_hash: e8b0bf9179d8
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 020: Extract ChannelListItem props into an interface.
## Review Comment

The inline object type here is out of step with the repository’s TypeScript shape rule. A small `ChannelListItemProps` interface keeps this consistent with the rest of `web/`.

As per coding guidelines, "Use `interface` for defining object shapes in TypeScript (pattern is in Zod schemas and types)".

---

## Triage

- Decision: `valid`
- Root cause: the local inline object type for `ChannelListItem` props diverges from the repo convention of using named interfaces for object shapes in TypeScript.
- Fix approach: extract a small `ChannelListItemProps` interface and use it in the component signature.
