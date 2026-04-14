---
status: pending
file: web/src/systems/workspace/hooks/use-workspaces.ts
line: 12
severity: nitpick
author: coderabbitai[bot]
provider_ref: review:4103023844,nitpick_hash:7b18b78deb69
review_hash: 7b18b78deb69
source_review_id: "4103023844"
source_review_submitted_at: "2026-04-14T02:37:32Z"
---

# Issue 031: Extract hook options into an interface.
## Review Comment

Please replace the inline object-shape type with a named `interface` for consistency.

As per coding guidelines, `web/**/*.ts?(x)`: "Use `interface` for defining object shapes in TypeScript".

---

## Triage

- Decision: `UNREVIEWED`
- Notes:
