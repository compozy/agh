---
status: resolved
file: web/src/systems/network/components/network-workspace-shell.tsx
line: 493
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4167301360,nitpick_hash:3970a05f444f
review_hash: 3970a05f444f
source_review_id: "4167301360"
source_review_submitted_at: "2026-04-24T01:39:58Z"
---

# Issue 007: Composite key may collide if multiple fields share the same label-value pair.
## Review Comment

Using `${field.label}-${field.value}` could produce duplicate keys. Consider using the array index for uniqueness.

## Triage

- Decision: `invalid`
- Reasoning: `NetworkDetailFieldList` is not rendering arbitrary user-provided field arrays. The only callers build `aboutFields` and `wireFields` from fixed, labeled arrays in `web/src/hooks/routes/use-network-page.ts`, and those labels are unique by construction (`Purpose`, `Created`, `Peer ID`, `Channel`, `Sent`, `Received`, etc.). Within the current model there is no duplicate-sibling key collision to fix.
- Why no code change: Replacing the current key with the array index would make the list less stable without addressing an observed defect. The existing key remains deterministic for the controlled field sets this component renders.
- Outcome: Analysis complete; no code change required.
